package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port           string
	Env            string
	FrontendURL    string
	APIVersion     string
	TimeoutSeconds int
	DatabaseURL    string
	JWTSecret             string
	JWTExpiresIn           string
	RefreshTokenExpiresIn  string
	BCryptCost             int
	RateLimitRate  float64
	RateLimitBurst float64
}

// Load reads settings from the environment and loads a .env file if available.
func Load() *Config {
	// Attempt to load .env file from the current directory
	_ = loadEnvFile(".env")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = os.Getenv("ENV") // fallback to ENV
		if env == "" {
			env = "development"
		}
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	apiVersion := os.Getenv("API_VERSION")
	if apiVersion == "" {
		apiVersion = "v1"
	}

	timeoutStr := os.Getenv("TIMEOUT_SECONDS")
	timeoutSec := 60 // default to 60s
	if timeoutStr != "" {
		if val, err := strconv.Atoi(timeoutStr); err == nil && val > 0 {
			timeoutSec = val
		}
	}

	dbURL := os.Getenv("DATABASE_URL")
	// default connection string if not specified
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/fleetcontrol?sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "supersecretchangeinproduction"
	}

	jwtExpiresIn := os.Getenv("JWT_EXPIRES_IN")
	if jwtExpiresIn == "" {
		jwtExpiresIn = "24h"
	}

	refreshTokenExpiresIn := os.Getenv("REFRESH_TOKEN_EXPIRES_IN")
	if refreshTokenExpiresIn == "" {
		refreshTokenExpiresIn = "168h" // 7 days
	}

	bcryptCostStr := os.Getenv("BCRYPT_COST")
	bcryptCost := 10 // default
	if bcryptCostStr != "" {
		if val, err := strconv.Atoi(bcryptCostStr); err == nil && val >= 4 && val <= 31 {
			bcryptCost = val
		}
	}

	rateLimitRateStr := os.Getenv("RATE_LIMIT_RATE")
	rateLimitRate := 0.1667 // default: 10 requests per minute
	if rateLimitRateStr != "" {
		if val, err := strconv.ParseFloat(rateLimitRateStr, 64); err == nil && val > 0 {
			rateLimitRate = val
		}
	}

	rateLimitBurstStr := os.Getenv("RATE_LIMIT_BURST")
	rateLimitBurst := 10.0 // default: 10 requests burst capacity
	if rateLimitBurstStr != "" {
		if val, err := strconv.ParseFloat(rateLimitBurstStr, 64); err == nil && val > 0 {
			rateLimitBurst = val
		}
	}

	return &Config{
		Port:           port,
		Env:            env,
		FrontendURL:    frontendURL,
		APIVersion:     apiVersion,
		TimeoutSeconds: timeoutSec,
		DatabaseURL:    dbURL,
		JWTSecret:             jwtSecret,
		JWTExpiresIn:           jwtExpiresIn,
		RefreshTokenExpiresIn:  refreshTokenExpiresIn,
		BCryptCost:             bcryptCost,
		RateLimitRate:  rateLimitRate,
		RateLimitBurst: rateLimitBurst,
	}
}

// loadEnvFile reads a .env file and sets environment variables if they aren't already set.
func loadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err // file probably doesn't exist, safe to ignore
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines or comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Strip quotes if present
		if len(value) >= 2 {
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
				(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
		}

		// Set value in environment if not already defined
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}

	return scanner.Err()
}
