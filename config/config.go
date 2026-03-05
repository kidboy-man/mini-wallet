package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort    string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBURL         string
	JWTSecret     string
	JWTExpiryMins int
	BcryptCost    int
	AppEnv        string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	jwtExpiry, _ := strconv.Atoi(getEnv("JWT_EXPIRY_MINS", "15"))
	bcryptCost, _ := strconv.Atoi(getEnv("BCRYPT_COST", "12"))

	cfg := &Config{
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", ""),
		DBName:        getEnv("DB_NAME", "mini_wallet"),
		DBURL:         getEnv("DATABASE_URL", ""),
		JWTSecret:     getEnv("JWT_SECRET", ""),
		JWTExpiryMins: jwtExpiry,
		BcryptCost:    bcryptCost,
		AppEnv:        getEnv("APP_ENV", "development"),
	}

	if cfg.DBURL == "" {
		cfg.DBURL = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=disable",
			cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName,
		)
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
