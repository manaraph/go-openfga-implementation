package handler

import (
	"io"
	"net/http"

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
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		apiResponse(w, http.StatusBadRequest, map[string]string{
			"message": "invalid multipart form: " + err.Error(),
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
