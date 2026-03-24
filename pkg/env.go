package pkg

import (
	"fmt"
	"os"
)

func GetOptionalEnv(key string) string {
	value := os.Getenv(key)

	return value
}

func GetEnvOr(key string, defaultValue string) string {
	value := GetOptionalEnv(key)

	if value == "" {
		return defaultValue
	}

	return value
}

func GetRequiredEnv(key string) string {
	value := GetOptionalEnv(key)

	if value == "" {
		panic(fmt.Errorf("missing %s in env", key))
	}

	return value
}
