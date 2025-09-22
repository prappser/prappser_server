package main

import (
	"github.com/prappser/prappser_server/internal"
	"github.com/prappser/prappser_server/internal/application"
	"github.com/prappser/prappser_server/internal/keys"
	"github.com/prappser/prappser_server/internal/status"
	"github.com/prappser/prappser_server/internal/user"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

func main() {
	// Initialize RSA keys (generate on first run)
	privateKey, publicKey, err := keys.GetOrGenerateRSAKeyPair()
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing RSA keys")
		return
	}
	log.Info().Msg("RSA keys initialized successfully")

	db, err := internal.NewDB()
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing database")
		return
	}
	config, err := internal.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading config")
		return
	}

	// Initialize user components
	userRepository := user.NewSQLite3UserRepository(db)
	userService := user.NewUserService(userRepository, config.Users, privateKey, publicKey)
	userEndpoints := user.NewEndpoints(userRepository, config.Users, privateKey, publicKey, userService)
	statusEndpoints := status.NewEndpoints("1.0.0")
	
	// Initialize application components
	appRepository := application.NewSQLiteRepository(db)
	appService := application.NewApplicationService(appRepository)
	appEndpoints := application.NewApplicationEndpoints(appService)

	requestHandler := internal.NewRequestHandler(userEndpoints, statusEndpoints, userService, appEndpoints)

	if err := fasthttp.ListenAndServe(":4545", requestHandler); err != nil {
		log.Fatal().Err(err).Msg("Error starting server")
	}
}
