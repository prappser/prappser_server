package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"

	"github.com/prappser/prappser_server/internal"
	"github.com/prappser/prappser_server/internal/application"
	"github.com/prappser/prappser_server/internal/event"
	"github.com/prappser/prappser_server/internal/health"
	"github.com/prappser/prappser_server/internal/invitation"
	"github.com/prappser/prappser_server/internal/keys"
	"github.com/prappser/prappser_server/internal/setup"
	"github.com/prappser/prappser_server/internal/status"
	"github.com/prappser/prappser_server/internal/user"
	"github.com/prappser/prappser_server/internal/websocket"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/valyala/fasthttp"
)

func initLogging() {
	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = "info" // Default level
	}

	switch strings.ToLower(level) {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		log.Warn().Str("level", level).Msg("Unknown log level, defaulting to info")
	}

	log.Info().Str("level", level).Msg("Logging initialized")
}

func main() {
	initLogging()

	// Load config first (needed for key derivation)
	config, err := internal.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading config")
		return
	}

	// Derive RSA keys deterministically from master password + external URL
	privateKey, publicKey, err := keys.DeriveRSAKeyPair(config.MasterPassword, config.ExternalURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Error deriving RSA keys")
		return
	}
	log.Info().Msg("RSA keys derived successfully")

	db, err := internal.NewDB()
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing database")
		return
	}

	// Initialize user components
	userRepository := user.NewUserRepository(db)
	userService := user.NewUserService(userRepository, config.Users, privateKey, publicKey)
	userEndpoints := user.NewEndpoints(userRepository, config.Users, privateKey, publicKey, userService)
	statusEndpoints := status.NewEndpoints("1.0.0")
	healthEndpoints := health.NewEndpoints("1.0.0")

	// Initialize application repository
	appRepository := application.NewRepository(db)

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run()
	log.Info().Msg("WebSocket hub started")

	// Initialize event components (must be before application service, needs app repository)
	eventRepository := event.NewEventRepository(db)
	eventService := event.NewEventService(eventRepository, appRepository, wsHub)
	eventEndpoints := event.NewEventEndpoints(eventService)

	// Initialize application service (events are now client-produced via POST /events)
	appService := application.NewApplicationService(appRepository)

	// Convert server public key to PEM format for application responses
	publicKeyPEM := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(publicKey),
	}
	serverPublicKeyString := string(pem.EncodeToMemory(publicKeyPEM))

	appEndpoints := application.NewApplicationEndpoints(appService, serverPublicKeyString)

	// Start event cleanup scheduler (runs daily at 2 AM, 7 days retention)
	cleanupScheduler := event.NewCleanupScheduler(eventService, 7)
	cleanupScheduler.Start()
	log.Info().Msg("Event cleanup scheduler started (daily at 2 AM, 7 days retention)")

	// Initialize invitation components (needs app repo, user repo, and event service)
	invitationRepository := invitation.NewInvitationRepository(db)
	invitationService := invitation.NewInvitationService(invitationRepository, privateKey, publicKey, appRepository, db, config.ExternalURL, userRepository, eventService)
	invitationEndpoints := invitation.NewInvitationEndpoints(invitationService)

	// Initialize setup endpoints (railway token management)
	setupEndpoints := setup.NewSetupEndpoints(db)

	log.Info().
		Str("port", config.Port).
		Str("externalURL", config.ExternalURL).
		Msg("Server configuration")

	// Initialize WebSocket handler (integrated with FastHTTP on same port)
	wsHandler := websocket.NewHandler(wsHub, userService)

	requestHandler := internal.NewRequestHandler(config, userEndpoints, statusEndpoints, healthEndpoints, userService, appEndpoints, invitationEndpoints, eventEndpoints, setupEndpoints, wsHandler)

	// Start unified FastHTTP server (REST API + WebSocket on same port)
	serverAddr := fmt.Sprintf(":%s", config.Port)
	log.Info().Str("addr", serverAddr).Msg("Starting unified HTTP server (REST + WebSocket)")
	if err := fasthttp.ListenAndServe(serverAddr, requestHandler); err != nil {
		log.Fatal().Err(err).Msg("Error starting HTTP server")
	}
}
