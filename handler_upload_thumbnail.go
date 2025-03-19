package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

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

	const maxMemory = 10 << 20 // 10MB
	r.ParseMultipartForm(maxMemory)

	// "thumbnail" should match the HTML form input name
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse thumbnail", err)
		return
	}
	defer file.Close()
	// get media type of the file
	file_type := header.Header.Get("Content-Type")

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to read file", err)
		return
	}

	// Get the video record from the database
	videorecord, err := cfg.db.GetVideo(videoID)

	// Check if the video exists and belongs to the user
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Video not found", err)
		return
	}
	if videorecord.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Video does not belong to user", nil)
		return
	}

	// Check what the file type is to determine file extension of thumbnail
	var file_extension string
	if file_type != "image/jpeg" && file_type != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", nil)
		return
	}
	if file_type == "image/jpeg" {
		file_extension = ".jpg"
	} else {
		file_extension = ".png"
	}

	// Construct the file path of the thumbnail
	thumbnail_path := filepath.Join(cfg.assetsRoot, videoID.String()+file_extension)

	fmt.Println("thumbnail path: ", thumbnail_path)

	// Create the thumbnail file
	thumbnail_file, err := os.Create(thumbnail_path)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create thumbnail file", err)
		return
	}
	defer thumbnail_file.Close()

	// Copy the image data to the thumbnail file
	if _, err := io.Copy(thumbnail_file, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to copy image data to thumbnail file", err)
		return
	}

	// Update the video record in the database
	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s%s", cfg.port, videoID.String(), file_extension)
	videorecord.ThumbnailURL = &thumbnailURL
	err = cfg.db.UpdateVideo(videorecord)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to update video record", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videorecord)
}
