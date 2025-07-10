package status

import (
	"github.com/goccy/go-json"
	"github.com/valyala/fasthttp"
)

type StatusEndpoints struct {
	version string
}

func NewEndpoints(version string) *StatusEndpoints {
	return &StatusEndpoints{
		version: version,
	}
}

type StatusResponse struct {
	Health  string `json:"health"`
	Version string `json:"version"`
}

func (se *StatusEndpoints) Status(ctx *fasthttp.RequestCtx) {
	response := StatusResponse{
		Health:  "OK",
		Version: se.version,
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
