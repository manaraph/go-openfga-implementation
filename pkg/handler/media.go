package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/manaraph/go-openfga-implementation/internal/utils"
	"github.com/openfga/go-sdk/client"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
)

type FileHandler struct {
	DB      *sqlx.DB
	MongoDB *mongo.Database
	FGA     *client.OpenFgaClient
}

type fileRequest struct {
	Username string `json:"username"`
}

func NewFileHandler(db *sqlx.DB, mongoDB *mongo.Database, fga *client.OpenFgaClient) *FileHandler {
	return &FileHandler{
		DB:      db,
		MongoDB: mongoDB,
		FGA:     fga,
	}
}

// POST /files/upload
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	userID, ok := utils.UserIDFromContext(ctx)
	if !ok {
		apiResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	const maxSize = 10 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	if err := r.ParseMultipartForm(maxSize); err != nil {
		apiResponse(w, http.StatusBadRequest, map[string]string{
			"message":      "file too large.",
			"errorDetails": err.Error(),
		})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		apiResponse(w, http.StatusBadRequest, map[string]string{
			"message": "file is required",
		})
		return
	}
	defer file.Close()

	bucket, err := gridfs.NewBucket(h.MongoDB)
	if err != nil {
		apiResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "failed to create bucket",
		})
		return
	}

	uploadStream, err := bucket.OpenUploadStream(header.Filename)
	if err != nil {
		apiResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "failed to open upload stream",
		})
		return
	}
	defer uploadStream.Close()

	_, err = io.Copy(uploadStream, file)
	if err != nil {
		apiResponse(w, http.StatusInternalServerError, map[string]string{
			"message": "failed to save file",
		})
		return
	}

	fileID := uploadStream.FileID.(primitive.ObjectID).Hex()
	body := client.ClientWriteRequest{
		Writes: []client.ClientTupleKey{
			{
				User:     "user:" + userID,
				Relation: "owner",
				Object:   "file:" + fileID,
			},
		},
	}
	_, err = h.FGA.Write(ctx).Body(body).Execute()
	if err != nil {
		apiResponse(w, http.StatusInternalServerError, map[string]string{
			"error":        "failed to write auth relationship",
			"errorDetails": err.Error(),
		})
		return
	}

	apiResponse(w, http.StatusCreated, map[string]any{
		"file_id":  fileID,
		"filename": header.Filename,
	})
}

// GET /files
func (h *FileHandler) GetFiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := utils.UserIDFromContext(ctx)
	if !ok {
		apiResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	listRequest := client.ClientListObjectsRequest{
		User:     "user:" + userID,
		Relation: "viewer",
		Type:     "file",
	}

	fgaResp, err := h.FGA.ListObjects(ctx).Body(listRequest).Execute()
	if err != nil {
		apiResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch permissions"})
		return
	}

	var fileOIDs []primitive.ObjectID
	for _, obj := range fgaResp.GetObjects() {
		// Strip the "file:" prefix
		idStr := obj[len("file:"):]
		oid, err := primitive.ObjectIDFromHex(idStr)
		if err == nil {
			fileOIDs = append(fileOIDs, oid)
		}
	}

	if len(fileOIDs) == 0 {
		apiResponse(w, http.StatusOK, map[string]any{"data": []any{}})
		return
	}

	cursor, err := h.MongoDB.Collection("fs.files").Find(ctx, bson.M{
		"_id": bson.M{"$in": fileOIDs},
	})
	if err != nil {
		apiResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to fetch files from db"})
		return
	}
	defer cursor.Close(ctx)

	var files []map[string]any
	if err := cursor.All(ctx, &files); err != nil {
		apiResponse(w, http.StatusInternalServerError, map[string]string{"error": "failed to decode files"})
		return
	}

	apiResponse(w, http.StatusOK, map[string]any{
		"data": files,
	})
}

// GET /files/{id}
func (h *FileHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	fileID := chi.URLParam(r, "id")

	userID, ok := utils.UserIDFromContext(ctx)
	if !ok {
		apiResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	body := client.ClientCheckRequest{
		User:     "user:" + userID,
		Relation: "owner",
		Object:   "file:" + fileID,
	}

	check, err := h.FGA.Check(ctx).Body(body).Execute()
	if err != nil || !*check.Allowed {
		apiResponse(w, http.StatusForbidden, map[string]string{
			"error":        "forbidden: only owners can download file",
			"errorDetails": err.Error(),
		})
		return
	}

	oid, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		apiResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid file id"})
		return
	}

	bucket, _ := gridfs.NewBucket(h.MongoDB)
	stream, err := bucket.OpenDownloadStream(oid)
	if err != nil {
		apiResponse(w, http.StatusNotFound, map[string]string{"error": "file not found"})
		return
	}
	defer stream.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(w, stream)
}

// POST /files/{id}/share
func (h *FileHandler) ShareFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	fileID := chi.URLParam(r, "id")

	ownerID, ok := utils.UserIDFromContext(ctx)
	if !ok {
		apiResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req fileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// Verify user is owner of the file before sharing
	body := client.ClientCheckRequest{
		User:     "user:" + ownerID,
		Relation: "owner",
		Object:   "file:" + fileID,
	}
	check, err := h.FGA.Check(ctx).Body(body).Execute()
	if err != nil || !*check.Allowed {
		apiResponse(w, http.StatusForbidden, map[string]string{
			"error":        "forbidden: only owners can share access",
			"errorDetails": err.Error(),
		})
		return
	}

	// Lookup user ID in Postgres
	var targetID int
	err = h.DB.QueryRow("SELECT id FROM users WHERE username=$1", req.Username).Scan(&targetID)
	if err != nil {
		apiResponse(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	// Share access with target user
	reqBody := client.ClientWriteRequest{
		Writes: []client.ClientTupleKey{
			{
				User:     "user:" + strconv.Itoa(targetID),
				Relation: "collaborator",
				Object:   "file:" + fileID,
			},
		},
	}
	_, err = h.FGA.Write(ctx).Body(reqBody).Execute()
	if err != nil {
		apiResponse(w, http.StatusInternalServerError, map[string]string{
			"error":        "failed to write auth relationship",
			"errorDetails": err.Error(),
		})
		return
	}

	apiResponse(w, http.StatusOK, map[string]string{
		"message": "file shared",
	})
}

// POST /files/{id}/revoke
func (h *FileHandler) RevokeAccess(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	fileID := chi.URLParam(r, "id")

	ownerID, ok := utils.UserIDFromContext(ctx)
	if !ok {
		apiResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req fileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiResponse(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	// Verify user is owner of the file before sharing
	body := client.ClientCheckRequest{
		User:     "user:" + ownerID,
		Relation: "owner",
		Object:   "file:" + fileID,
	}
	check, err := h.FGA.Check(ctx).Body(body).Execute()
	if err != nil || !*check.Allowed {
		apiResponse(w, http.StatusForbidden, map[string]string{
			"error":        "forbidden: only owners can revoke access",
			"errorDetails": err.Error(),
		})
		return
	}

	// Lookup user ID in Postgres
	var targetID int
	err = h.DB.QueryRow("SELECT id FROM users WHERE username=$1", req.Username).Scan(&targetID)
	if err != nil {
		apiResponse(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}

	// Revoke access for target user
	deleteBody := client.ClientWriteRequest{
		Deletes: []client.ClientTupleKeyWithoutCondition{
			{
				User:     "user:" + strconv.Itoa(targetID),
				Relation: "collaborator",
				Object:   "file:" + fileID,
			},
		},
	}
	_, err = h.FGA.Write(ctx).Body(deleteBody).Execute()
	if err != nil {
		apiResponse(w, http.StatusInternalServerError, map[string]string{
			"error":        "failed to write auth relationship",
			"errorDetails": err.Error(),
		})
		return
	}

	apiResponse(w, http.StatusOK, map[string]string{
		"message": "access revoked",
	})
}
