package status

import (
	"github.com/goccy/go-json"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

type StorageUsageGetter interface {
	GetTotalUsedBytes() (int64, error)
}

type StatusEndpoints struct {
	version          string
	maxFileSizeBytes int64
	chunkSizeBytes   int64
	storageRepo      StorageUsageGetter
}

func NewEndpoints(version string, maxFileSizeBytes, chunkSizeBytes int64, storageRepo StorageUsageGetter) *StatusEndpoints {
	return &StatusEndpoints{
		version:          version,
		maxFileSizeBytes: maxFileSizeBytes,
		chunkSizeBytes:   chunkSizeBytes,
		storageRepo:      storageRepo,
	}
}

type StatusResponse struct {
	Health           string `json:"health"`
	Version          string `json:"version"`
	MaxFileSizeBytes int64  `json:"maxFileSizeBytes"`
	ChunkSizeBytes   int64  `json:"chunkSizeBytes"`
	StorageUsedBytes int64  `json:"storageUsedBytes"`
}

func (se *StatusEndpoints) Status(ctx *fasthttp.RequestCtx) {
	var storageUsedBytes int64
	if se.storageRepo != nil {
		used, err := se.storageRepo.GetTotalUsedBytes()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get storage used bytes")
		} else {
			storageUsedBytes = used
		}
	}

	response := StatusResponse{
		Health:           "OK",
		Version:          se.version,
		MaxFileSizeBytes: se.maxFileSizeBytes,
		ChunkSizeBytes:   se.chunkSizeBytes,
		StorageUsedBytes: storageUsedBytes,
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)

	responseJSON, err := json.Marshal(response)
	if err != nil {
		ctx.Error("Internal Server Error", fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetBody(responseJSON)
}
