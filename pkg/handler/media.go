package handler

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
)

type FileHandler struct {
	DB *mongo.Database
}

func NewFileHandler(db *mongo.Database) *FileHandler {
	return &FileHandler{
		DB: db,
	}
}

func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	_, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

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

	apiResponse(w, http.StatusCreated, map[string]any{
		"file_id":  uploadStream.FileID,
		"filename": header.Filename,
	})
}

func (h *FileHandler) GetFiles(w http.ResponseWriter, r *http.Request) {
	cursor, err := h.DB.Collection("fs.files").Find(r.Context(), bson.M{})
	if err != nil {
		http.Error(w, "failed to fetch files", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(r.Context())

	var files []map[string]any
	if err := cursor.All(r.Context(), &files); err != nil {
		http.Error(w, "failed to decode files", http.StatusInternalServerError)
		return
	}

	apiResponse(w, http.StatusOK, map[string]any{
		"data": files,
	})
}

func (h *FileHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

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
