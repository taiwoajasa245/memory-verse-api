// Env loader
package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv       string
	Port         string
	DBHost       string
	DBPort       string
	DBName       string
	DBUser       string
	DBPassword   string
	DBSchema     string
	JWTSecret    string
	SmtpFrom     string
	SmtpPassword string
	SmtpHost     string
	SmtpPort     string
	SwaggerHost  string
}

// LoadConfig loads environment variables from the .env file
func LoadConfig() *Config {

	appEnv := os.Getenv("APP_ENV")

	switch appEnv {
	case "production":
		if err := godotenv.Load(".env.production"); err == nil {
			fmt.Println("Loaded .env.production")
		}
	default:
		if err := godotenv.Load(".env.development"); err == nil {
			fmt.Println("Loaded .env.development")
		}
	}

	// Load .env file (only for local/dev)
	// _ = godotenv.Load(envFile)

	// fmt.Println("loaded: ", envFile)

	// dbPort, err := strconv.Atoi(getEnv("BLUEPRINT_DB_PORT", "5432"))
	// if err != nil {
	// 	log.Fatalf("Invalid database port: %v", err)
	// }

	cfg := &Config{
		AppEnv:       getEnv("APP_ENV", "development"),
		Port:         getEnv("PORT", "8080"),
		DBHost:       getEnv("BLUEPRINT_DB_HOST", "localhost"),
		DBPort:       getEnv("BLUEPRINT_DB_PORT", "5432"),
		DBName:       getEnv("BLUEPRINT_DB_DATABASE", "memory_verse"),
		DBUser:       getEnv("BLUEPRINT_DB_USERNAME", "postgres"),
		DBPassword:   getEnv("BLUEPRINT_DB_PASSWORD", ""),
		DBSchema:     getEnv("BLUEPRINT_DB_SCHEMA", "public"),
		JWTSecret:    getEnv("JWT_SECRET", ""),
		SmtpFrom:     getEnv("SMTP_FROM", ""),
		SmtpPassword: getEnv("SMTP_PASSWORD", ""),
		SmtpHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		SmtpPort:     getEnv("SMTP_PORT", "587"),
		SwaggerHost:  getEnv("SWAGGER_HOST", "localhost:8080"),
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func GetAppEnv() string {
	if value, exists := os.LookupEnv("APP_ENV"); exists {
		return value
	}
	return "development"
}
