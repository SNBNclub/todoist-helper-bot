package config

import (
	"errors"
	"fmt"
	"log"
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

// TODO how to fix it to work from any dir
func LoadConfig() (*Config, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	cmdDir := filepath.Dir(currentDir) // ../cmd
	modDir := filepath.Dir(cmdDir)     // ..
	log.Printf("%s\n", modDir)
	err = godotenv.Load(filepath.Join(modDir, ".env"))
	if err != nil {
		return nil, err
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
	err = validateStruct(*cfg)
	if err != nil {
		return nil, err
	}
	log.Printf("%v\n", cfg)
	return cfg, nil
}

// TODO work with errrors
// TODO delete get from
// get from here - https://medium.com/@anajankow/fast-check-if-all-struct-fields-are-set-in-golang-bba1917213d2
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
			err = fmt.Errorf("%v%s in not set; ", err, fieldName)
		}
	}
	return err
}
