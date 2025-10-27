package middleware

import (
	"github.com/valyala/fasthttp"
)

type CORSMiddleware struct {
	allowedOrigins []string
}

func NewCORSMiddleware(allowedOrigins []string) *CORSMiddleware {
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}
	return &CORSMiddleware{
		allowedOrigins: allowedOrigins,
	}
}

func (cm *CORSMiddleware) Handle(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		origin := string(ctx.Request.Header.Peek("Origin"))

		if cm.isOriginAllowed(origin) {
			ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
		} else if len(cm.allowedOrigins) == 1 && cm.allowedOrigins[0] == "*" {
			ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
		}

		ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		ctx.Response.Header.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		ctx.Response.Header.Set("Access-Control-Max-Age", "86400")

		if string(ctx.Method()) == "OPTIONS" {
			ctx.SetStatusCode(fasthttp.StatusNoContent)
			return
		}

		next(ctx)
	}
}

func (cm *CORSMiddleware) isOriginAllowed(origin string) bool {
	for _, allowed := range cm.allowedOrigins {
		if allowed == origin {
			return true
		}
	}
	return false
}
