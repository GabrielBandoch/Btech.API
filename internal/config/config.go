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
		frontendURL = "http://localhost:5174"
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

	return &Config{
		Port:           port,
		Env:            env,
		FrontendURL:    frontendURL,
		APIVersion:     apiVersion,
		TimeoutSeconds: timeoutSec,
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
