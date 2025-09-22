package internal

import (
	"strings"
	
	"github.com/prappser/prappser_server/internal/application"
	"github.com/prappser/prappser_server/internal/middleware"
	"github.com/prappser/prappser_server/internal/status"
	"github.com/prappser/prappser_server/internal/user"
	"github.com/valyala/fasthttp"
)

func NewRequestHandler(userEndpoints *user.UserEndpoints, statusEndpoints *status.StatusEndpoints, userService *user.UserService, appEndpoints *application.ApplicationEndpoints) fasthttp.RequestHandler {
	authMiddleware := middleware.NewAuthMiddleware(userService)
	
	return func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())
		
		switch {
		case path == "/users/owners/register":
			userEndpoints.OwnerRegister(ctx)
		case path == "/users/challenge":
			userEndpoints.GetChallenge(ctx)
		case path == "/users/auth":
			userEndpoints.UserAuth(ctx)
		case path == "/status":
			authMiddleware.RequireAuth(statusEndpoints.Status)(ctx)
		
		// Application endpoints
		case path == "/applications/register":
			authMiddleware.RequireRole(user.RoleOwner, appEndpoints.RegisterApplication)(ctx)
		case path == "/applications":
			authMiddleware.RequireRole(user.RoleOwner, appEndpoints.ListApplications)(ctx)
		case strings.HasPrefix(path, "/applications/") && strings.HasSuffix(path, "/state"):
			// Extract application ID from path
			parts := strings.Split(path, "/")
			if len(parts) == 4 && parts[3] == "state" {
				ctx.SetUserValue("appID", parts[2])
				authMiddleware.RequireRole(user.RoleOwner, appEndpoints.GetApplicationState)(ctx)
			} else {
				ctx.Error("Not Found", fasthttp.StatusNotFound)
			}
		case strings.HasPrefix(path, "/applications/"):
			// Extract application ID from path
			parts := strings.Split(path, "/")
			if len(parts) == 3 {
				ctx.SetUserValue("appID", parts[2])
				
				// Route based on HTTP method
				method := string(ctx.Method())
				switch method {
				case "GET":
					authMiddleware.RequireRole(user.RoleOwner, appEndpoints.GetApplication)(ctx)
				case "DELETE":
					authMiddleware.RequireRole(user.RoleOwner, appEndpoints.DeleteApplication)(ctx)
				default:
					ctx.Error("Method Not Allowed", fasthttp.StatusMethodNotAllowed)
				}
			} else {
				ctx.Error("Not Found", fasthttp.StatusNotFound)
			}
		
		default:
			ctx.Error("Not Found", fasthttp.StatusNotFound)
		}
	}
}
