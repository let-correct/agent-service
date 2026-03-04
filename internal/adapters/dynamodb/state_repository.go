package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/troysnowden/let-correct-viewing/internal/domain/oauth2"
)

const stateTTL = 10 * time.Minute

type StateRepository struct {
	dynamo    *dynamodb.Client
	tableName string
}

func NewStateRepository(dynamo *dynamodb.Client, tableName string) *StateRepository {
	return &StateRepository{dynamo: dynamo, tableName: tableName}
}

func (r *StateRepository) Save(ctx context.Context, state string) error {
	expiresAt := strconv.FormatInt(time.Now().Add(stateTTL).Unix(), 10)
	_, err := r.dynamo.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item: map[string]types.AttributeValue{
			"state_value": &types.AttributeValueMemberS{Value: state},
			"expires_at":  &types.AttributeValueMemberN{Value: expiresAt},
		},
	})
	if err != nil {
		return fmt.Errorf("saving state: %w", err)
	}
	return nil
}

func (r *StateRepository) Consume(ctx context.Context, state string) error {
	_, err := r.dynamo.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"state_value": &types.AttributeValueMemberS{Value: state},
		},
		ConditionExpression: aws.String("attribute_exists(state_value)"),
	})
	if err != nil {
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			return oauth2.ErrStateNotFound
		}
		return fmt.Errorf("consuming state: %w", err)
	}
	return nil
}
