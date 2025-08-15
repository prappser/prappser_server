package internal

import (
	"github.com/prappser/prappser_server/internal/middleware"
	"github.com/prappser/prappser_server/internal/status"
	"github.com/prappser/prappser_server/internal/user"
	"github.com/valyala/fasthttp"
)

func NewRequestHandler(userEndpoints *user.UserEndpoints, statusEndpoints *status.StatusEndpoints, userService *user.UserService) fasthttp.RequestHandler {
	authMiddleware := middleware.NewAuthMiddleware(userService)
	
	return func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/users/owners/register":
			userEndpoints.OwnerRegister(ctx)
		case "/users/challenge":
			userEndpoints.GetChallenge(ctx)
		case "/users/auth":
			userEndpoints.UserAuth(ctx)
		case "/status":
			authMiddleware.RequireAuth(statusEndpoints.Status)(ctx)
		default:
			ctx.Error("Not Found", fasthttp.StatusNotFound)
		}
	}
}
