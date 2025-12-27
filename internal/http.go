package internal

import (
	"strings"

	"github.com/prappser/prappser_server/internal/application"
	"github.com/prappser/prappser_server/internal/event"
	"github.com/prappser/prappser_server/internal/health"
	"github.com/prappser/prappser_server/internal/invitation"
	"github.com/prappser/prappser_server/internal/middleware"
	"github.com/prappser/prappser_server/internal/setup"
	"github.com/prappser/prappser_server/internal/status"
	"github.com/prappser/prappser_server/internal/user"
	"github.com/valyala/fasthttp"
)

func NewRequestHandler(config *Config, userEndpoints *user.UserEndpoints, statusEndpoints *status.StatusEndpoints, healthEndpoints *health.HealthEndpoints, userService *user.UserService, appEndpoints *application.ApplicationEndpoints, invitationEndpoints *invitation.InvitationEndpoints, eventEndpoints *event.EventEndpoints, setupEndpoints *setup.SetupEndpoints) fasthttp.RequestHandler {
	authMiddleware := middleware.NewAuthMiddleware(userService)
	corsMiddleware := middleware.NewCORSMiddleware(config.AllowedOrigins)

	handler := func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())

		switch {
		// Setup/landing page - shows URL copy page before owner registration
		case path == "/":
			setupEndpoints.LandingPage(ctx)

		// Railway setup endpoint - store token for server self-management
		case path == "/setup/railway":
			method := string(ctx.Method())
			if method == "POST" {
				authMiddleware.RequireRole(user.RoleOwner, setupEndpoints.SetRailwayToken)(ctx)
			} else {
				ctx.Error("Method Not Allowed", fasthttp.StatusMethodNotAllowed)
			}

		case path == "/users/owners/register":
			userEndpoints.OwnerRegister(ctx)
		case path == "/users/challenge":
			userEndpoints.GetChallenge(ctx)
		case path == "/users/auth":
			userEndpoints.UserAuth(ctx)
		case path == "/health":
			healthEndpoints.Health(ctx)
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
		case strings.HasPrefix(path, "/applications/") && strings.Contains(path, "/invites"):
			// Invitation endpoints under /applications/{appID}/invites
			parts := strings.Split(path, "/")
			if len(parts) >= 4 && parts[3] == "invites" {
				ctx.SetUserValue("appID", parts[2])

				if len(parts) == 4 {
					// /applications/{appID}/invites
					method := string(ctx.Method())
					switch method {
					case "POST":
						// Create invite
						authMiddleware.RequireRole(user.RoleOwner, invitationEndpoints.CreateInvite)(ctx)
					case "GET":
						// List invites
						authMiddleware.RequireRole(user.RoleOwner, invitationEndpoints.ListInvites)(ctx)
					default:
						ctx.Error("Method Not Allowed", fasthttp.StatusMethodNotAllowed)
					}
				} else if len(parts) == 5 {
					// /applications/{appID}/invites/{inviteID}
					ctx.SetUserValue("inviteID", parts[4])
					method := string(ctx.Method())
					if method == "DELETE" {
						// Revoke invite
						authMiddleware.RequireRole(user.RoleOwner, invitationEndpoints.RevokeInvite)(ctx)
					} else {
						ctx.Error("Method Not Allowed", fasthttp.StatusMethodNotAllowed)
					}
				} else {
					ctx.Error("Not Found", fasthttp.StatusNotFound)
				}
			} else {
				ctx.Error("Not Found", fasthttp.StatusNotFound)
			}
		case strings.HasPrefix(path, "/applications/") && strings.HasSuffix(path, "/members/me"):
			// Leave application endpoint: DELETE /applications/{appID}/members/me
			parts := strings.Split(path, "/")
			if len(parts) == 5 && parts[3] == "members" && parts[4] == "me" {
				ctx.SetUserValue("appID", parts[2])
				method := string(ctx.Method())
				if method == "DELETE" {
					authMiddleware.RequireAuth(appEndpoints.LeaveApplication)(ctx)
				} else {
					ctx.Error("Method Not Allowed", fasthttp.StatusMethodNotAllowed)
				}
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
					authMiddleware.RequireAuth(appEndpoints.GetApplication)(ctx)
				case "DELETE":
					authMiddleware.RequireRole(user.RoleOwner, appEndpoints.DeleteApplication)(ctx)
				default:
					ctx.Error("Method Not Allowed", fasthttp.StatusMethodNotAllowed)
				}
			} else {
				ctx.Error("Not Found", fasthttp.StatusNotFound)
			}

		// Invitation endpoints
		case strings.HasPrefix(path, "/invites/") && strings.HasSuffix(path, "/info"):
			// GET /invites/{token}/info (public endpoint)
			parts := strings.Split(path, "/")
			if len(parts) == 4 && parts[3] == "info" {
				ctx.SetUserValue("token", parts[2])
				invitationEndpoints.GetInviteInfo(ctx)
			} else {
				ctx.Error("Not Found", fasthttp.StatusNotFound)
			}
		case strings.HasPrefix(path, "/invites/") && strings.HasSuffix(path, "/join"):
			// POST /invites/{token}/join (PUBLIC endpoint - no auth required)
			parts := strings.Split(path, "/")
			if len(parts) == 4 && parts[3] == "join" {
				ctx.SetUserValue("token", parts[2])
				method := string(ctx.Method())
				if method == "POST" {
					invitationEndpoints.JoinApplication(ctx) // No auth - endpoint creates user if needed
				} else {
					ctx.Error("Method Not Allowed", fasthttp.StatusMethodNotAllowed)
				}
			} else {
				ctx.Error("Not Found", fasthttp.StatusNotFound)
			}
		case path == "/invites/check":
			// POST /invites/check (PUBLIC endpoint - no auth required)
			method := string(ctx.Method())
			if method == "POST" {
				invitationEndpoints.CheckInvitation(ctx)
			} else {
				ctx.Error("Method Not Allowed", fasthttp.StatusMethodNotAllowed)
			}

		// Event endpoints
		case path == "/events":
			method := string(ctx.Method())
			if method == "GET" {
				// GET /events?since={eventId}&limit={limit}
				authMiddleware.RequireAuth(eventEndpoints.GetEvents)(ctx)
			} else if method == "POST" {
				// POST /events - Submit event for validation and processing
				authMiddleware.RequireAuth(eventEndpoints.SubmitEvent)(ctx)
			} else {
				ctx.Error("Method Not Allowed", fasthttp.StatusMethodNotAllowed)
			}

		default:
			ctx.Error("Not Found", fasthttp.StatusNotFound)
		}
	}

	return corsMiddleware.Handle(handler)
}
