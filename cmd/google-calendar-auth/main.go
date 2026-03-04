package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awsdynamo "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamoAdapter "github.com/troysnowden/let-correct-viewing/internal/adapters/dynamodb"
	googleOAuth "github.com/troysnowden/let-correct-viewing/internal/adapters/oauth2/google"
	"github.com/troysnowden/let-correct-viewing/internal/application"
	"github.com/troysnowden/let-correct-viewing/internal/domain/oauth2"
	"github.com/troysnowden/let-correct-viewing/internal/handler"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel(),
	}))

	ctx := context.Background()

	// Load AWS config once at cold start — picks up the Lambda execution role automatically.
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "failed to load AWS config", "error", err)
		os.Exit(1)
	}

	dynamoClient := awsdynamo.NewFromConfig(awsCfg)
	stateRepo := dynamoAdapter.NewStateRepository(dynamoClient, os.Getenv("STATE_TABLE_NAME"))

	googleClient := googleOAuth.NewClient(
		os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		os.Getenv("GOOGLE_REDIRECT_URL"),
	)

	initiateAuth := application.NewInitiateAuth(
		map[oauth2.ProviderID]oauth2.Client{
			oauth2.ProviderGoogle: googleClient,
		},
		stateRepo,
	)

	h := handler.New(logger, initiateAuth)

	lambda.StartWithOptions(
		h.Handle,
		lambda.WithContext(ctx),
	)
}

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
