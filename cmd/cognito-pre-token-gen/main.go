package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handle(_ context.Context, event events.CognitoEventUserPoolsPreTokenGenV2_0) (events.CognitoEventUserPoolsPreTokenGenV2_0, error) {
	email := event.Request.UserAttributes["email"]
	if email == "" {
		return event, nil
	}

	event.Response.ClaimsAndScopeOverrideDetails = events.ClaimsAndScopeOverrideDetailsV2_0{
		AccessTokenGeneration: events.AccessTokenGenerationV2_0{
			ClaimsToAddOrOverride: map[string]interface{}{
				"email": email,
			},
		},
	}

	return event, nil
}

func main() {
	lambda.Start(handle)
}
