package dynamodb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/troysnowden/agent-service/internal/domain/oauth2"
)

var ErrNotFound = errors.New("token not found")

// tokenPayload is the JSON structure encrypted as a single KMS blob.
type tokenPayload struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type TokenRepository struct {
	dynamo    *dynamodb.Client
	kms       *kms.Client
	tableName string
	kmsKeyID  string
}

func NewTokenRepository(dynamo *dynamodb.Client, kms *kms.Client, tableName, kmsKeyID string) *TokenRepository {
	return &TokenRepository{
		dynamo:    dynamo,
		kms:       kms,
		tableName: tableName,
		kmsKeyID:  kmsKeyID,
	}
}

func (r *TokenRepository) Save(ctx context.Context, token *oauth2.Token) error {
	payload, err := json.Marshal(tokenPayload{
		AccessToken:  token.AccessToken(),
		RefreshToken: token.RefreshToken(),
	})
	if err != nil {
		return fmt.Errorf("marshalling token payload: %w", err)
	}

	ciphertext, err := r.encrypt(ctx, payload)
	if err != nil {
		return fmt.Errorf("encrypting token payload: %w", err)
	}

	_, err = r.dynamo.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item: map[string]types.AttributeValue{
			"email":      &types.AttributeValueMemberS{Value: token.Email()},
			"provider":   &types.AttributeValueMemberS{Value: string(token.Provider())},
			"token_data": &types.AttributeValueMemberB{Value: ciphertext},
			"expires_at": &types.AttributeValueMemberN{Value: strconv.FormatInt(token.ExpiresAt().Unix(), 10)},
		},
	})
	if err != nil {
		return fmt.Errorf("putting item: %w", err)
	}

	return nil
}

func (r *TokenRepository) FindByEmailAndProvider(ctx context.Context, email string, provider oauth2.ProviderID) (*oauth2.Token, error) {
	out, err := r.dynamo.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"email":    &types.AttributeValueMemberS{Value: email},
			"provider": &types.AttributeValueMemberS{Value: string(provider)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("getting item: %w", err)
	}
	if out.Item == nil {
		return nil, ErrNotFound
	}

	ciphertext := out.Item["token_data"].(*types.AttributeValueMemberB).Value
	plaintext, err := r.decrypt(ctx, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypting token payload: %w", err)
	}

	var payload tokenPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return nil, fmt.Errorf("unmarshalling token payload: %w", err)
	}

	expiresAtUnix, err := strconv.ParseInt(out.Item["expires_at"].(*types.AttributeValueMemberN).Value, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing expires_at: %w", err)
	}

	return oauth2.NewToken(email, provider, payload.AccessToken, payload.RefreshToken, time.Unix(expiresAtUnix, 0)), nil
}

func (r *TokenRepository) encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	out, err := r.kms.Encrypt(ctx, &kms.EncryptInput{
		KeyId:     aws.String(r.kmsKeyID),
		Plaintext: plaintext,
	})
	if err != nil {
		return nil, err
	}
	return out.CiphertextBlob, nil
}

func (r *TokenRepository) decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	out, err := r.kms.Decrypt(ctx, &kms.DecryptInput{
		CiphertextBlob: ciphertext,
	})
	if err != nil {
		return nil, err
	}
	return out.Plaintext, nil
}
