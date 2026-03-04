package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awsdynamo "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	awskms "github.com/aws/aws-sdk-go-v2/service/kms"
	dynamoAdapter "github.com/troysnowden/let-correct-viewing/internal/adapters/dynamodb"
	googleOAuth "github.com/troysnowden/let-correct-viewing/internal/adapters/oauth2/google"
	"github.com/troysnowden/let-correct-viewing/internal/application"
	"github.com/troysnowden/let-correct-viewing/internal/config"
	"github.com/troysnowden/let-correct-viewing/internal/domain/oauth2"
	"github.com/troysnowden/let-correct-viewing/internal/handler"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel(cfg.LogLevel),
	}))

	ctx := context.Background()

	// Load AWS config once at cold start — picks up the Lambda execution role automatically.
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "failed to load AWS config", "error", err)
		os.Exit(1)
	}

	dynamoClient := awsdynamo.NewFromConfig(awsCfg)
	kmsClient := awskms.NewFromConfig(awsCfg)

	stateRepo := dynamoAdapter.NewStateRepository(dynamoClient, cfg.StateTableName)
	tokenRepo := dynamoAdapter.NewTokenRepository(dynamoClient, kmsClient, cfg.TokenTableName, cfg.KMSKeyARN)

	googleClient := googleOAuth.NewClient(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL)

	clients := map[oauth2.ProviderID]oauth2.Client{
		oauth2.ProviderGoogle: googleClient,
	}

	initiateAuth := application.NewInitiateAuth(clients, stateRepo)
	exchangeCode := application.NewExchangeCode(clients, tokenRepo, stateRepo)

	h := handler.New(logger, initiateAuth, exchangeCode)

	lambda.StartWithOptions(
		h.Handle,
		lambda.WithContext(ctx),
	)
}

func logLevel(level string) slog.Level {
	switch level {
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
