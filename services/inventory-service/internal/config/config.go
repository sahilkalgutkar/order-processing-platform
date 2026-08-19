package config

import "os"

type Config struct {
	HTTPPort    string
	MongoURI    string
	MongoDBName string
	SQSQueueURL string
	AWSEndpoint string
	AWSRegion   string
}

func Load() Config {
	return Config{
		HTTPPort:    getEnv("HTTP_PORT", "8081"),
		MongoURI:    getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDBName: getEnv("MONGO_DB_NAME", "inventory"),
		SQSQueueURL: getEnv("SQS_QUEUE_URL", ""),
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
