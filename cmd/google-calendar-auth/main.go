package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	handler "github.com/troysnowden/let-correct-viewing/internal/google-calendar-auth"
)

func main() {
	// Structured JSON logging — CloudWatch will index these fields.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel(),
	}))

	// Initialise your handler with its dependencies.
	// This runs once on cold start, so it's a good place to set up
	// DB connections, config, SDK clients, etc.
	h := handler.New(logger)

	// Start the Lambda runtime loop.
	lambda.StartWithOptions(
		h.Handle,
		lambda.WithContext(context.Background()),
	)
}

// logLevel reads the LOG_LEVEL env var, defaulting to Info.
func logLevel() slog.Level {
	switch os.Getenv("LOG_LEVEL") {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
