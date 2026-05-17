package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}
	fmt.Println("uploading video", videoID, "by user", userID)
	const maxMemory = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, maxMemory)
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error finding video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}
	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()
	contentType := header.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "bad request", err)
		return
	}
	var ext string
	switch mediaType {
	case "video/mp4":
		ext = ".mp4"
	default:
		respondWithError(w, http.StatusBadRequest, "Unsupported File Type", nil)
		return
	}
	f, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error Creating Temp File", err)
		return
	}
	defer os.Remove(f.Name())
	defer f.Close()
	_, err = io.Copy(f, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error copying file", err)
		return
	}
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error", err)
		return
	}
	key := make([]byte, 32)
	rand.Read(key)
	encoded := base64.RawURLEncoding.EncodeToString(key)
	clientKey := encoded + ext
	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:      awsString("tubely-67867"),
		Key:         "clientKey",
		Body:        file,
		ContentType: awsString("video/mp4"),
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error uploading video", err)
		return
	}
}
