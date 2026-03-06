package config

import "github.com/caarlos0/env/v11"

type OauthLambdaConfig struct {
	LogLevel           string `env:"LOG_LEVEL"            envDefault:"INFO"`
	StateTableName     string `env:"STATE_TABLE_NAME,required"`
	TokenTableName     string `env:"TOKEN_TABLE_NAME,required"`
	KMSKeyARN          string `env:"KMS_KEY_ARN,required"`
	GoogleClientID     string `env:"GOOGLE_CLIENT_ID,required"`
	GoogleClientSecret string `env:"GOOGLE_CLIENT_SECRET,required"`
	GoogleRedirectURL  string `env:"GOOGLE_REDIRECT_URL,required"`
	ArthurClientID     string `env:"ARTHUR_CLIENT_ID,required"`
	ArthurClientSecret string `env:"ARTHUR_CLIENT_SECRET,required"`
	ArthurRedirectURL  string `env:"ARTHUR_REDIRECT_URL,required"`
	ArthurAuthURL      string `env:"ARTHUR_AUTH_URL,required"`
	ArthurTokenURL     string `env:"ARTHUR_TOKEN_URL,required"`
}

func Load() (OauthLambdaConfig, error) {
	var cfg OauthLambdaConfig
	return cfg, env.Parse(&cfg)
}
