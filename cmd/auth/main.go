package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awsdynamo "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	awskms "github.com/aws/aws-sdk-go-v2/service/kms"
	arthurOAuth "github.com/troysnowden/agent-service/internal/adapters/oauth2/arthur"
	oauth2dynamo "github.com/troysnowden/agent-service/internal/adapters/oauth2/dynamodb"
	googleOAuth "github.com/troysnowden/agent-service/internal/adapters/oauth2/google"
	apioauth2 "github.com/troysnowden/agent-service/internal/application/oauth2"
	"github.com/troysnowden/agent-service/internal/config"
	"github.com/troysnowden/agent-service/internal/domain/oauth2"
	"github.com/troysnowden/agent-service/internal/handler/auth"
)

func main() {
	cfg, err := config.LoadAuth()
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

	stateRepo := oauth2dynamo.NewStateRepository(dynamoClient, cfg.StateTableName)
	tokenRepo := oauth2dynamo.NewTokenRepository(dynamoClient, kmsClient, cfg.TokenTableName, cfg.KMSKeyARN)

	googleClient := googleOAuth.NewClient(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL)
	arthurClient := arthurOAuth.NewClient(cfg.ArthurClientID, cfg.ArthurClientSecret, cfg.ArthurRedirectURL, cfg.ArthurAuthURL, cfg.ArthurTokenURL)

	clients := map[oauth2.ProviderID]oauth2.Client{
		oauth2.ProviderGoogle: googleClient,
		oauth2.ProviderArthur: arthurClient,
	}

	initiateAuth := apioauth2.NewInitiateAuth(clients, stateRepo)
	exchangeCode := apioauth2.NewExchangeCode(clients, tokenRepo, stateRepo)
	getToken := apioauth2.NewGetToken(clients, tokenRepo)

	h := auth.New(logger, initiateAuth, exchangeCode, getToken)

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
