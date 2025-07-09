package internal

import (
	"github.com/prappser/prappser_server/internal/owner"
	"github.com/prappser/prappser_server/internal/status"
	"github.com/valyala/fasthttp"
)

func NewRequestHandler(ownersEndpoints *owner.OwnerEndpoints, statusEndpoints *status.StatusEndpoints) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/owners/register":
			ownersEndpoints.OwnersRegister(ctx)
		case "/status":
			statusEndpoints.Status(ctx)
		default:
			ctx.Error("Not Found", fasthttp.StatusNotFound)
		}
	}
}
