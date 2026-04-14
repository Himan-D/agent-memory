package main

import (
	"log"

	"github.com/getsentry/sentry-go"

	"agent-memory/internal/config"
)

var sentryEnabled = false

func initSentry(cfg *config.AppConfig) {
	if cfg.SentryDSN == "" {
		log.Println("Sentry: DSN not configured, error tracking disabled")
		return
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.SentryDSN,
		Environment:      cfg.Environment,
		Release:          "hystersis@1.0.0",
		TracesSampleRate: 0.1,
	})
	if err != nil {
		log.Printf("Sentry: Failed to initialize: %v", err)
		return
	}

	sentryEnabled = true
	log.Printf("Sentry: Initialized (environment=%s)", cfg.Environment)
}

func captureError(err error) {
	if !sentryEnabled {
		return
	}
	sentry.CaptureException(err)
}

func captureMessage(msg string) {
	if !sentryEnabled {
		return
	}
	sentry.CaptureMessage(msg)
}

func flushSentry() {
	if sentryEnabled {
		sentry.Flush(5)
	}
}
