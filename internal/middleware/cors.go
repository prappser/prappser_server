package middleware

import (
	"regexp"
	"github.com/valyala/fasthttp"
)

type CORSMiddleware struct {
	allowedOrigins []string
	localhostRegex *regexp.Regexp
}

func NewCORSMiddleware(allowedOrigins []string) *CORSMiddleware {
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}
	// Compile regex for localhost with any port
	localhostRegex := regexp.MustCompile(`^https?://localhost:\d+$`)
	return &CORSMiddleware{
		allowedOrigins: allowedOrigins,
		localhostRegex: localhostRegex,
	}
}

func (cm *CORSMiddleware) Handle(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		origin := string(ctx.Request.Header.Peek("Origin"))

		// Set CORS headers based on origin
		if cm.isOriginAllowed(origin) {
			ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
			// When using credentials, we must set the specific origin (not *)
			ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
		} else if len(cm.allowedOrigins) == 1 && cm.allowedOrigins[0] == "*" {
			ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
			// Cannot use credentials with wildcard origin
		}

		ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		ctx.Response.Header.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		ctx.Response.Header.Set("Access-Control-Expose-Headers", "Authorization, Content-Type")
		ctx.Response.Header.Set("Access-Control-Max-Age", "86400")

		if string(ctx.Method()) == "OPTIONS" {
			ctx.SetStatusCode(fasthttp.StatusNoContent)
			return
		}

		next(ctx)
	}
}

func (cm *CORSMiddleware) isOriginAllowed(origin string) bool {
	// Check exact match first
	for _, allowed := range cm.allowedOrigins {
		if allowed == origin {
			return true
		}
		// Check if allowed origin is a localhost pattern
		if allowed == "http://localhost:*" || allowed == "https://localhost:*" {
			if cm.localhostRegex.MatchString(origin) {
				return true
			}
		}
	}
	// Also allow any localhost origin if no specific origins are configured (dev mode)
	if len(cm.allowedOrigins) == 1 && cm.allowedOrigins[0] == "*" {
		return cm.localhostRegex.MatchString(origin)
	}
	return false
}
