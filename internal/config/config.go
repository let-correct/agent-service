package config

import "os"

type OauthLambdaConfig struct {
	LogLevel           string
	StateTableName     string
	TokenTableName     string
	KMSKeyARN          string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
}

func Load() OauthLambdaConfig {
	return OauthLambdaConfig{
		LogLevel:           os.Getenv("LOG_LEVEL"),
		StateTableName:     os.Getenv("STATE_TABLE_NAME"),
		TokenTableName:     os.Getenv("TOKEN_TABLE_NAME"),
		KMSKeyARN:          os.Getenv("KMS_KEY_ARN"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
	}
}
