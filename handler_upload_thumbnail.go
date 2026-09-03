package main

import (
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxMemory int64 = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid thumbnail", err)
		return
	}

	file, fileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid thumbnail", err)
		return
	}

	// Check media type
	mediaType, _, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content Type", err)
		return
	}

	if mediaType != "image/png" && mediaType != "image/jpeg" {
		respondWithError(w, http.StatusBadRequest, "Data is not an image", err)
		return
	}

	// Form the path
	parts := strings.Split(mediaType, "/")
	extension := parts[1]
	fileName := fmt.Sprintf("%s.%s", videoID.String(), extension)
	path := filepath.Join(cfg.assetsRoot, fileName)

	// Store the thumbnail
	storeThumbnail(file, path)

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "User is not aithorized to watchi this video", err)
		return
	}

	thumbnailUrl := fmt.Sprintf("http://localhost:%s/%s", cfg.port, path)
	video.ThumbnailURL = &thumbnailUrl

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Video update failed", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}

func storeThumbnail(thumbnailFile multipart.File, path string) error {
	newFile, err := os.Create(path)
	if err != nil {
		return err
	}

	_, err = io.Copy(newFile, thumbnailFile)
	if err != nil {
		return err
	}

	return nil
}
