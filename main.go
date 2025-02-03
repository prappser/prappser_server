package main

import (
	"github.com/prappser/prappser_server/internal"
	"github.com/prappser/prappser_server/internal/user"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

func main() {
	_, err := internal.NewDB()
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing database")
		return
	}

	if err := fasthttp.ListenAndServe(":8080", requestHandler); err != nil {
		log.Fatal().Err(err).Msg("Error starting server")
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	switch string(ctx.Path()) {
	case "/users/owners/register":
		user.HandleUsersOwnersRegister(ctx)
	default:
		ctx.Error("Not Found", fasthttp.StatusNotFound)
	}
}
