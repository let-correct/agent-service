package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awsdynamo "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"

	dynamosync "github.com/troysnowden/agent-service/internal/adapters/calendar/dynamodb/sync"
	sqsdispatcher "github.com/troysnowden/agent-service/internal/adapters/calendar/sqs/sync"
	calendarsync "github.com/troysnowden/agent-service/internal/application/calendar/sync"
	"github.com/troysnowden/agent-service/internal/config"
	handler "github.com/troysnowden/agent-service/internal/handler/calendar-sync-dispatcher"
)

func main() {
	cfg, err := config.LoadCalendarSyncDispatcher()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel(cfg.LogLevel),
	}))

	ctx := context.Background()

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "failed to load AWS config", "error", err)
		os.Exit(1)
	}

	dynamoClient := awsdynamo.NewFromConfig(awsCfg)
	sqsClient := awssqs.NewFromConfig(awsCfg)

	syncRepo := dynamosync.NewRepository(dynamoClient, cfg.SyncTableName)
	dispatcher := sqsdispatcher.NewDispatcher(sqsClient, cfg.SQSQueueURL)

	calendarSyncDispatcher := calendarsync.NewCalendarSyncDispatcher(syncRepo, dispatcher)

	h := handler.New(logger, calendarSyncDispatcher)

	lambda.StartWithOptions(h.Handle, lambda.WithContext(ctx))
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
