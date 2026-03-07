package calendarsync

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	domain "github.com/troysnowden/agent-service/internal/domain/calendar/sync"
)

type Repository struct {
	dynamo    *dynamodb.Client
	tableName string
}

func NewRepository(dynamo *dynamodb.Client, tableName string) *Repository {
	return &Repository{dynamo: dynamo, tableName: tableName}
}

func (r *Repository) FindAll(ctx context.Context) ([]*domain.Sync, error) {
	var syncs []*domain.Sync
	var lastKey map[string]types.AttributeValue

	for {
		out, err := r.dynamo.Scan(ctx, &dynamodb.ScanInput{
			TableName:         aws.String(r.tableName),
			ExclusiveStartKey: lastKey,
		})
		if err != nil {
			return nil, fmt.Errorf("scanning sync table: %w", err)
		}

		for _, item := range out.Items {
			s, err := itemToSync(item)
			if err != nil {
				return nil, err
			}
			syncs = append(syncs, s)
		}

		// scan returns a limited amount of items, items left determined by this attribute
		if out.LastEvaluatedKey == nil {
			break
		}
		lastKey = out.LastEvaluatedKey
	}

	return syncs, nil
}

func (r *Repository) Save(ctx context.Context, sync *domain.Sync) error {
	_, err := r.dynamo.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item: map[string]types.AttributeValue{
			"email":          &types.AttributeValueMemberS{Value: sync.Email()},
			"calendar_id":    &types.AttributeValueMemberS{Value: sync.CalendarID()},
			"sync_token":     &types.AttributeValueMemberS{Value: sync.SyncToken()},
			"last_synced_at": &types.AttributeValueMemberN{Value: strconv.FormatInt(sync.LastSyncedAt().Unix(), 10)},
		},
	})
	if err != nil {
		return fmt.Errorf("saving sync: %w", err)
	}

	return nil
}

func itemToSync(item map[string]types.AttributeValue) (*domain.Sync, error) {
	email := item["email"].(*types.AttributeValueMemberS).Value
	calendarID := item["calendar_id"].(*types.AttributeValueMemberS).Value
	syncToken := item["sync_token"].(*types.AttributeValueMemberS).Value

	lastSyncedAtUnix, err := strconv.ParseInt(item["last_synced_at"].(*types.AttributeValueMemberN).Value, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing last_synced_at for %s: %w", email, err)
	}

	var lastSyncedAt time.Time
	if lastSyncedAtUnix > 0 {
		lastSyncedAt = time.Unix(lastSyncedAtUnix, 0)
	}

	return domain.NewSync(email, calendarID, syncToken, lastSyncedAt), nil
}
