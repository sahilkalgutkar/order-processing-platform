package config

import "os"

type Config struct {
	HTTPPort     string
	DatabaseURL  string
	SNSTopicARN  string
	AWSEndpoint  string // set for localstack; empty in real AWS
	AWSRegion    string
}

func Load() Config {
	return Config{
		HTTPPort:    getEnv("HTTP_PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/orders?sslmode=disable"),
		SNSTopicARN: getEnv("SNS_TOPIC_ARN", ""),
		AWSEndpoint: getEnv("AWS_ENDPOINT_URL", ""),
		AWSRegion:   getEnv("AWS_REGION", "us-east-1"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
