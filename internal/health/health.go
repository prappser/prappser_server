package health

import (
	"github.com/goccy/go-json"
	"github.com/valyala/fasthttp"
)

type HealthEndpoints struct {
	version string
}

func NewEndpoints(version string) *HealthEndpoints {
	return &HealthEndpoints{
		version: version,
	}
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

func (h *HealthEndpoints) Health(ctx *fasthttp.RequestCtx) {
	response := HealthResponse{
		Status:  "ok",
		Version: h.version,
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
