package handler

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/manaraph/go-openfga-implementation/internal/utils"
	"github.com/openfga/go-sdk/client"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
)

type FileHandler struct {
	DB  *mongo.Database
	FGA *client.OpenFgaClient
}

func NewFileHandler(db *mongo.Database, fga *client.OpenFgaClient) *FileHandler {
	return &FileHandler{
		DB:  db,
		FGA: fga,
	}
}

func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	userID, ok := utils.UserIDFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	const maxSize = 10 << 20
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	if err := r.ParseMultipartForm(maxSize); err != nil {
		apiResponse(w, http.StatusBadRequest, map[string]string{
			"message": "file too large: " + err.Error(),
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

	bucket, err := gridfs.NewBucket(h.DB)
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
		http.Error(w, "failed to write auth relationship", http.StatusInternalServerError)
		return
	}

	apiResponse(w, http.StatusCreated, map[string]any{
		"file_id":  fileID,
		"filename": header.Filename,
	})
}

func (h *FileHandler) GetFiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := utils.UserIDFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	listRequest := client.ClientListObjectsRequest{
		User:     "user:" + userID,
		Relation: "owner",
		Type:     "file",
	}

	fgaResp, err := h.FGA.ListObjects(ctx).Body(listRequest).Execute()
	if err != nil {
		http.Error(w, "failed to fetch permissions", http.StatusInternalServerError)
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

	cursor, err := h.DB.Collection("fs.files").Find(ctx, bson.M{
		"_id": bson.M{"$in": fileOIDs},
	})
	if err != nil {
		http.Error(w, "failed to fetch files from db", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var files []map[string]any
	if err := cursor.All(ctx, &files); err != nil {
		http.Error(w, "failed to decode files", http.StatusInternalServerError)
		return
	}

	apiResponse(w, http.StatusOK, map[string]any{
		"data": files,
	})
}

func (h *FileHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	userID, ok := utils.UserIDFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	body := client.ClientCheckRequest{
		User:     "user:" + userID,
		Relation: "viewer",
		Object:   "file:" + id,
	}

	check, err := h.FGA.Check(ctx).Body(body).Execute()
	if err != nil {
		http.Error(w, "authorization error", http.StatusInternalServerError)
		return
	}

	if check.Allowed == nil || !*check.Allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "invalid file id", http.StatusBadRequest)
		return
	}

	bucket, _ := gridfs.NewBucket(h.DB)
	stream, err := bucket.OpenDownloadStream(oid)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	defer stream.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(w, stream)
}
