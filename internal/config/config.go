package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// TODO :: add logging levels to config
type Config struct {
	DB_HOST               string
	DB_PORT               string
	DB_NAME               string
	DB_USER               string
	DB_PASSWORD           string
	REDIS_HOST            string
	REDIS_PASSWORD        string
	REDIS_DB              int
	TELEGRAM_APITOKEN     string
	APP_CLIENT_ID         string
	APP_CLIENT_SECRET     string
	ENABLE_CONSOLE_LOGGER bool
	ENABLE_FILE_LOGGER    bool
	ENABLE_BOT_LOGGER     bool
}

func LoadConfig(filenames ...string) (*Config, error) {
	err := godotenv.Load(filenames...)
	if err != nil {
		return nil, fmt.Errorf("error in loading: %w", err)
	}
	cfg := &Config{
		DB_HOST:               getStringEnv("DB_HOST", "localhost"),
		DB_PORT:               getStringEnv("DB_PORT", "5432"),
		DB_NAME:               getStringEnv("DB_NAME", "tracker"),
		DB_USER:               getStringEnv("DB_USER", "root"),
		DB_PASSWORD:           getStringEnv("DB_PASS", "password"),
		REDIS_HOST:            getStringEnv("REDIS_HOST", ""),
		REDIS_PASSWORD:        getStringEnv("REDIS_PASSWORD", ""),
		REDIS_DB:              getIntEnv("REDIS_DB", 0),
		TELEGRAM_APITOKEN:     getStringEnv("TELEGRAM_APITOKEN", ""),
		APP_CLIENT_ID:         getStringEnv("TODOIST_CLIENT_ID", ""),
		APP_CLIENT_SECRET:     getStringEnv("TODOIST_CLIENT_SECRET", ""),
		ENABLE_CONSOLE_LOGGER: getBoolEnv("ENABLE_CONSOLE_LOGGER", true),
		ENABLE_FILE_LOGGER:    getBoolEnv("ENABLE_FILE_LOGGER", false),
		ENABLE_BOT_LOGGER:     getBoolEnv("ENABLE_BOT_LOGGER", false),
	}
	err = validateStruct(*cfg)
	if err != nil {
		return nil, fmt.Errorf("error during validating struct: %w", err)
	}
	return cfg, nil
}

func (c *Config) GetDBConnString() string {
	return fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s",
		c.DB_HOST,
		c.DB_PORT,
		c.DB_NAME,
		c.DB_USER,
		c.DB_PASSWORD,
	)
}

type LoggerConfig struct {
	ENABLE_CONSOLE_LOGGER bool
	ENABLE_FILE_LOGGER    bool
	ENABLE_BOT_LOGGER     bool
}

func (c *Config) GetLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		ENABLE_CONSOLE_LOGGER: c.ENABLE_CONSOLE_LOGGER,
		ENABLE_FILE_LOGGER:    c.ENABLE_FILE_LOGGER,
		ENABLE_BOT_LOGGER:     c.ENABLE_BOT_LOGGER,
	}
}

func getIntEnv(key string, defaultValue int) int {
	if valueStr, exists := os.LookupEnv(key); exists {
		value, err := strconv.Atoi(valueStr)
		if err != nil {
			return defaultValue
		}
		return value
	}
	return defaultValue
}

func getStringEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	valueStr, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	valueStr = strings.ToUpper(valueStr)
	switch valueStr {
	case "TRUE", "YES", "1":
		return true
	case "FALSE", "NO", "0":
		return false
	default:
		return defaultValue
	}
}

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

		if field.Kind() == reflect.Bool {
			continue
		}

		if field.IsZero() || !field.IsValid() {
			err = fmt.Errorf("%v%s in not set; ", err, fieldName)
		}
	}
	return err
}
