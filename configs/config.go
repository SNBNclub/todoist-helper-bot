package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/joho/godotenv"
)

type Config struct {
	DB_HOST           string
	DB_PORT           string
	DB_NAME           string
	DB_USER           string
	DB_PASSWORD       string
	TELEGRAM_APITOKEN string
	APP_CLIENT_ID     string
	APP_CLIENT_SECRET string
}

func LoadConfig() (*Config, error) {
	possibleLocations := []string{
		".env",
		filepath.Join("..", ".env"),
		filepath.Join("..", "..", ".env"),
	}

	var loadErr error
	for _, location := range possibleLocations {
		err := godotenv.Load(location)
		if err == nil {
			loadErr = nil
			break
		}
		loadErr = err
	}

	if loadErr != nil {
		return nil, fmt.Errorf("could not load environment variables: %w", loadErr)
	}

	cfg := &Config{
		DB_HOST:           os.Getenv("DB_HOST"),
		DB_PORT:           os.Getenv("DB_PORT"),
		DB_NAME:           os.Getenv("DB_NAME"),
		DB_USER:           os.Getenv("DB_USER"),
		DB_PASSWORD:       os.Getenv("DB_PASS"),
		TELEGRAM_APITOKEN: os.Getenv("TELEGRAM_APITOKEN"),
		APP_CLIENT_ID:     os.Getenv("TODOIST_CLIENT_ID"),
		APP_CLIENT_SECRET: os.Getenv("TODOIST_CLIENT_SECRET"),
	}

	err := validateStruct(*cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// validateStruct checks if all fields in the struct are set
func validateStruct(s any) (err error) {
	structType := reflect.TypeOf(s)
	if structType.Kind() != reflect.Struct {
		return errors.New("not struct")
	}

	structVal := reflect.ValueOf(s)
	fieldsCnt := structVal.NumField()

	for i := range fieldsCnt {
		field := structVal.Field(i)
		fieldName := structType.Field(i).Name

		if field.IsZero() || !field.IsValid() {
			err = fmt.Errorf("%v%s is not set; ", err, fieldName)
		}
	}
	return err
}
