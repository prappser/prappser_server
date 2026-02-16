package storage

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"github.com/prappser/prappser_server/internal/application"
	"github.com/prappser/prappser_server/internal/user"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

type Endpoints struct {
	service *Service
	appRepo *application.Repository
}

func NewEndpoints(service *Service, appRepo *application.Repository) *Endpoints {
	return &Endpoints{
		service: service,
		appRepo: appRepo,
	}
}

func (e *Endpoints) Upload(ctx *fasthttp.RequestCtx) {
	appID, publicKey, ok := e.checkAuthorization(ctx)
	if !ok {
		return
	}

	contentType := string(ctx.Request.Header.ContentType())
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		ctx.Error("Content-Type must be multipart/form-data", fasthttp.StatusBadRequest)
		return
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		ctx.Error("Failed to parse multipart form", fasthttp.StatusBadRequest)
		return
	}

	files := form.File["file"]
	if len(files) == 0 {
		ctx.Error("No file uploaded", fasthttp.StatusBadRequest)
		return
	}

	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		ctx.Error("Failed to open uploaded file", fasthttp.StatusInternalServerError)
		return
	}
	defer file.Close()

	storageID := ""
	if ids := form.Value["id"]; len(ids) > 0 {
		storageID = ids[0]
	}
	if storageID == "" {
		ctx.Error("Storage ID is required", fasthttp.StatusBadRequest)
		return
	}

	checksum := ""
	if checksums := form.Value["checksum"]; len(checksums) > 0 {
		checksum = checksums[0]
	}

	req := &UploadRequest{
		ID:          storageID,
		Filename:    fileHeader.Filename,
		ContentType: fileHeader.Header.Get("Content-Type"),
		SizeBytes:   fileHeader.Size,
		Checksum:    checksum,
	}

	if req.ContentType == "" || req.ContentType == "application/octet-stream" {
		req.ContentType = detectContentType(fileHeader.Filename)
	}

	stored, err := e.service.Upload(ctx, appID, publicKey, req, file)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upload file")
		ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		return
	}

	response, _ := json.Marshal(stored)
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusCreated)
	ctx.SetBody(response)
}

func (e *Endpoints) InitChunkedUpload(ctx *fasthttp.RequestCtx) {
	appID, publicKey, ok := e.checkAuthorization(ctx)
	if !ok {
		return
	}

	var req ChunkedUploadInitRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.Error("Invalid request body", fasthttp.StatusBadRequest)
		return
	}

	response, err := e.service.InitChunkedUpload(ctx, appID, publicKey, &req)
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		return
	}

	responseBody, _ := json.Marshal(response)
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusCreated)
	ctx.SetBody(responseBody)
}

func (e *Endpoints) UploadChunk(ctx *fasthttp.RequestCtx) {
	authenticatedUser, ok := ctx.UserValue("user").(*user.User)
	if !ok || authenticatedUser == nil {
		ctx.Error("Unauthorized", fasthttp.StatusUnauthorized)
		return
	}
	publicKey := authenticatedUser.PublicKey

	storageID, ok := ctx.UserValue("storageID").(string)
	if !ok || storageID == "" {
		ctx.Error("Storage ID is required", fasthttp.StatusBadRequest)
		return
	}

	chunkIndexStr, ok := ctx.UserValue("chunkIndex").(string)
	if !ok {
		ctx.Error("Chunk index is required", fasthttp.StatusBadRequest)
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		ctx.Error("Invalid chunk index", fasthttp.StatusBadRequest)
		return
	}

	stored, err := e.service.Get(ctx, storageID)
	if err != nil {
		ctx.Error("Storage not found", fasthttp.StatusNotFound)
		return
	}

	if stored.UploaderPublicKey != publicKey {
		ctx.Error("Not authorized", fasthttp.StatusForbidden)
		return
	}

	body := ctx.PostBody()
	if err := e.service.UploadChunk(ctx, storageID, chunkIndex, bytes.NewReader(body)); err != nil {
		ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
}

func (e *Endpoints) CompleteChunkedUpload(ctx *fasthttp.RequestCtx) {
	authenticatedUser, ok := ctx.UserValue("user").(*user.User)
	if !ok || authenticatedUser == nil {
		ctx.Error("Unauthorized", fasthttp.StatusUnauthorized)
		return
	}
	publicKey := authenticatedUser.PublicKey

	storageID, ok := ctx.UserValue("storageID").(string)
	if !ok || storageID == "" {
		ctx.Error("Storage ID is required", fasthttp.StatusBadRequest)
		return
	}

	stored, err := e.service.Get(ctx, storageID)
	if err != nil {
		ctx.Error("Storage not found", fasthttp.StatusNotFound)
		return
	}

	if stored.UploaderPublicKey != publicKey {
		ctx.Error("Not authorized", fasthttp.StatusForbidden)
		return
	}

	completedStorage, err := e.service.CompleteChunkedUpload(ctx, storageID)
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusBadRequest)
		return
	}

	response, _ := json.Marshal(completedStorage)
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBody(response)
}

func (e *Endpoints) GetFile(ctx *fasthttp.RequestCtx) {
	stored, _, ok := e.getStorageAndCheckAccess(ctx)
	if !ok {
		return
	}

	storageID := stored.ID
	reader, stored, err := e.service.GetData(ctx, storageID)
	if err != nil {
		ctx.Error("Failed to retrieve file", fasthttp.StatusInternalServerError)
		return
	}
	defer reader.Close()

	ctx.SetContentType(stored.ContentType)
	ctx.Response.Header.Set("Content-Disposition", "inline; filename=\""+stored.Filename+"\"")
	ctx.Response.Header.Set("Content-Length", strconv.FormatInt(stored.SizeBytes, 10))

	if _, err := io.Copy(ctx, reader); err != nil {
		log.Error().Err(err).Msg("Failed to stream file")
	}
}

func (e *Endpoints) GetThumbnail(ctx *fasthttp.RequestCtx) {
	stored, _, ok := e.getStorageAndCheckAccess(ctx)
	if !ok {
		return
	}

	storageID := stored.ID
	reader, _, err := e.service.GetThumbnail(ctx, storageID)
	if err != nil {
		ctx.Error("Thumbnail not available", fasthttp.StatusNotFound)
		return
	}
	defer reader.Close()

	ctx.SetContentType("image/jpeg")

	if _, err := io.Copy(ctx, reader); err != nil {
		log.Error().Err(err).Msg("Failed to stream thumbnail")
	}
}

func (e *Endpoints) DeleteFile(ctx *fasthttp.RequestCtx) {
	authenticatedUser, ok := ctx.UserValue("user").(*user.User)
	if !ok || authenticatedUser == nil {
		ctx.Error("Unauthorized", fasthttp.StatusUnauthorized)
		return
	}
	publicKey := authenticatedUser.PublicKey

	storageID, ok := ctx.UserValue("storageID").(string)
	if !ok || storageID == "" {
		ctx.Error("Storage ID is required", fasthttp.StatusBadRequest)
		return
	}

	err := e.service.Delete(ctx, storageID, publicKey)
	if err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "not authorized"):
			ctx.Error("Not authorized to delete this file", fasthttp.StatusForbidden)
		case strings.Contains(errMsg, "not found"):
			ctx.Error("Storage not found", fasthttp.StatusNotFound)
		default:
			ctx.Error("Failed to delete file", fasthttp.StatusInternalServerError)
		}
		return
	}

	ctx.SetStatusCode(fasthttp.StatusNoContent)
}

func (e *Endpoints) checkAuthorization(ctx *fasthttp.RequestCtx) (appID, publicKey string, ok bool) {
	authenticatedUser, ok := ctx.UserValue("user").(*user.User)
	if !ok || authenticatedUser == nil {
		ctx.Error("Unauthorized", fasthttp.StatusUnauthorized)
		return "", "", false
	}
	publicKey = authenticatedUser.PublicKey

	appID = string(ctx.QueryArgs().Peek("applicationId"))
	if appID == "" {
		ctx.Error("applicationId is required", fasthttp.StatusBadRequest)
		return "", "", false
	}

	isMember, err := e.appRepo.IsMember(appID, publicKey)
	if err != nil {
		ctx.Error("Failed to verify membership", fasthttp.StatusInternalServerError)
		return "", "", false
	}
	if !isMember {
		ctx.Error("Not a member of this application", fasthttp.StatusForbidden)
		return "", "", false
	}

	return appID, publicKey, true
}

func (e *Endpoints) getStorageAndCheckAccess(ctx *fasthttp.RequestCtx) (stored *Storage, publicKey string, ok bool) {
	authenticatedUser, ok := ctx.UserValue("user").(*user.User)
	if !ok || authenticatedUser == nil {
		ctx.Error("Unauthorized", fasthttp.StatusUnauthorized)
		return nil, "", false
	}
	publicKey = authenticatedUser.PublicKey

	storageID, ok := ctx.UserValue("storageID").(string)
	if !ok || storageID == "" {
		ctx.Error("Storage ID is required", fasthttp.StatusBadRequest)
		return nil, "", false
	}

	stored, err := e.service.Get(ctx, storageID)
	if err != nil {
		ctx.Error("Storage not found", fasthttp.StatusNotFound)
		return nil, "", false
	}

	isMember, err := e.appRepo.IsMember(stored.ApplicationID, publicKey)
	if err != nil {
		ctx.Error("Failed to verify membership", fasthttp.StatusInternalServerError)
		return nil, "", false
	}
	if !isMember {
		ctx.Error("Not a member of this application", fasthttp.StatusForbidden)
		return nil, "", false
	}

	return stored, publicKey, true
}

func detectContentType(filename string) string {
	dotIndex := strings.LastIndex(filename, ".")
	if dotIndex == -1 || dotIndex == len(filename)-1 {
		return "application/octet-stream"
	}
	ext := strings.ToLower(filename[dotIndex+1:])
	switch ext {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "mp4":
		return "video/mp4"
	case "webm":
		return "video/webm"
	case "mov":
		return "video/mov"
	default:
		return "application/octet-stream"
	}
}
