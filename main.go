package main

import (
	"github.com/prappser/prappser_server/internal"
	"github.com/prappser/prappser_server/internal/application"
	"github.com/prappser/prappser_server/internal/event"
	"github.com/prappser/prappser_server/internal/invitation"
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

	// Initialize event components
	eventRepository := event.NewEventRepository(db)
	eventService := event.NewEventService(eventRepository)
	eventEndpoints := event.NewEventEndpoints(eventService)

	// Start event cleanup scheduler (runs daily at 2 AM, 7 days retention)
	cleanupScheduler := event.NewCleanupScheduler(eventService, 7)
	cleanupScheduler.Start()
	log.Info().Msg("Event cleanup scheduler started (daily at 2 AM, 7 days retention)")

	// Initialize invitation components (needs event service and app repo)
	invitationRepository := invitation.NewSQLiteInvitationRepository(db)
	invitationService := invitation.NewInvitationService(invitationRepository, privateKey, publicKey, appRepository, eventService, db)
	invitationEndpoints := invitation.NewInvitationEndpoints(invitationService)

	requestHandler := internal.NewRequestHandler(userEndpoints, statusEndpoints, userService, appEndpoints, invitationEndpoints, eventEndpoints)

	if err := fasthttp.ListenAndServe(":4545", requestHandler); err != nil {
		log.Fatal().Err(err).Msg("Error starting server")
	}
}
