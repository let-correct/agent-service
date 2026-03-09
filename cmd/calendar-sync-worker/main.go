package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awsdynamo "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	awseventbridge "github.com/aws/aws-sdk-go-v2/service/eventbridge"
	awskms "github.com/aws/aws-sdk-go-v2/service/kms"

	calendarsyncrepo "github.com/troysnowden/agent-service/internal/adapters/calendar/dynamodb/sync"
	eventbridge "github.com/troysnowden/agent-service/internal/adapters/calendar/eventbridge/sync"
	googlesync "github.com/troysnowden/agent-service/internal/adapters/calendar/google/sync"
	oauth2dynamo "github.com/troysnowden/agent-service/internal/adapters/oauth2/dynamodb"
	apicalendarsync "github.com/troysnowden/agent-service/internal/application/calendar/sync"
	"github.com/troysnowden/agent-service/internal/config"
	"github.com/troysnowden/agent-service/internal/domain/oauth2"
	calendarworker "github.com/troysnowden/agent-service/internal/handler/calendar-sync-worker"
)

// googleTokenRepository uses the adapter pattern to provide access to an interface
// the application layer can't satisfy (providing email and provider)
type googleTokenRepository struct {
	repo *oauth2dynamo.TokenRepository
}

func (r *googleTokenRepository) FindByEmail(ctx context.Context, email string) (*oauth2.Token, error) {
	return r.repo.FindByEmailAndProvider(ctx, email, oauth2.ProviderGoogle)
}

func main() {
	cfg, err := config.LoadCalendarSyncWorker()
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
	kmsClient := awskms.NewFromConfig(awsCfg)
	eventbridgeClient := awseventbridge.NewFromConfig(awsCfg)

	tokenRepo := oauth2dynamo.NewTokenRepository(dynamoClient, kmsClient, cfg.TokenTableName, cfg.KMSKeyARN)
	syncRepo := calendarsyncrepo.NewRepository(dynamoClient, cfg.SyncTableName)
	publisher := eventbridge.New(eventbridgeClient, cfg.EventBusName)
	calendarClient := googlesync.NewClient()

	googleTokenRepo := &googleTokenRepository{repo: tokenRepo}

	syncer := apicalendarsync.NewSyncCalendar(calendarClient, publisher, syncRepo, googleTokenRepo)

	h := calendarworker.New(logger, syncer)

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
