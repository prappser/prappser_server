package event

import (
	"time"

	"github.com/rs/zerolog/log"
)

// CleanupScheduler manages periodic cleanup of old events
type CleanupScheduler struct {
	eventService  *EventService
	retentionDays int
	ticker        *time.Ticker
	done          chan bool
}

// NewCleanupScheduler creates a new cleanup scheduler
func NewCleanupScheduler(eventService *EventService, retentionDays int) *CleanupScheduler {
	if retentionDays <= 0 {
		retentionDays = 7 // Default 7 days
	}

	return &CleanupScheduler{
		eventService:  eventService,
		retentionDays: retentionDays,
		done:          make(chan bool),
	}
}

// Start begins the cleanup scheduler (runs daily at 2 AM)
func (cs *CleanupScheduler) Start() {
	// Calculate time until next 2 AM
	now := time.Now()
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
	if now.After(nextRun) {
		// If it's already past 2 AM today, schedule for tomorrow
		nextRun = nextRun.Add(24 * time.Hour)
	}

	// Wait until first run
	durationUntilFirstRun := time.Until(nextRun)
	log.Info().
		Str("nextRun", nextRun.Format("2006-01-02 15:04:05")).
		Msg("Event cleanup scheduler started")

	// Start timer for first run
	time.AfterFunc(durationUntilFirstRun, func() {
		cs.runCleanup()

		// After first run, run every 24 hours
		cs.ticker = time.NewTicker(24 * time.Hour)
		go cs.loop()
	})
}

// loop runs the cleanup task on a schedule
func (cs *CleanupScheduler) loop() {
	for {
		select {
		case <-cs.ticker.C:
			cs.runCleanup()
		case <-cs.done:
			cs.ticker.Stop()
			return
		}
	}
}

// runCleanup executes the cleanup task
func (cs *CleanupScheduler) runCleanup() {
	log.Info().
		Int("retentionDays", cs.retentionDays).
		Msg("Starting event cleanup")

	deletedCount, err := cs.eventService.CleanupOldEvents(cs.retentionDays)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to cleanup old events")
		return
	}

	log.Info().
		Int64("deletedCount", deletedCount).
		Msg("Event cleanup completed successfully")
}

// Stop stops the cleanup scheduler
func (cs *CleanupScheduler) Stop() {
	log.Info().Msg("Stopping event cleanup scheduler")
	if cs.ticker != nil {
		cs.done <- true
	}
}

// RunNow executes cleanup immediately (useful for testing or manual triggers)
func (cs *CleanupScheduler) RunNow() {
	cs.runCleanup()
}
