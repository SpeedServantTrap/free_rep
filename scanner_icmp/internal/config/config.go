package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	RabbitMQURL         string
	ScannerName         string
	PingTimeout         time.Duration
	AutoScanEnabled     bool
	AutoScanTargets     []string
	AutoScanPingCount   int
	AutoScanInterval    time.Duration
}

func Load() Config {
	err := godotenv.Load()
	if err != nil {
		log.Printf("No .env file found: %v", err)
	}

	targetsStr := getEnv("ICMP_AUTO_SCAN_TARGETS", "8.8.8.8,1.1.1.1")
	targets := parseTargets(targetsStr)

	return Config{
		RabbitMQURL:         getEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"),
		ScannerName:         getEnv("SCANNER_NAME", "scanner_icmp"),
		PingTimeout:         getDurationEnv("PING_TIMEOUT", 5*time.Second),
		AutoScanEnabled:     getBoolEnv("ICMP_AUTO_SCAN_ENABLED", false),
		AutoScanTargets:     targets,
		AutoScanPingCount:   getIntEnv("ICMP_AUTO_SCAN_PING_COUNT", 4),
		AutoScanInterval:    getDurationEnv("ICMP_AUTO_SCAN_INTERVAL", 1*time.Minute),
	}
}

func parseTargets(targetsStr string) []string {
	if targetsStr == "" {
		return []string{"8.8.8.8", "1.1.1.1"}
	}

	targets := strings.Split(targetsStr, ",")
	result := make([]string, 0, len(targets))
	for _, target := range targets {
		trimmed := strings.TrimSpace(target)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		return []string{"8.8.8.8", "1.1.1.1"}
	}

	return result
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	value := getEnv(key, "")
	if value == "" {
		return defaultValue
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return result
}

func getBoolEnv(key string, defaultValue bool) bool {
	value := getEnv(key, "")
	if value == "" {
		return defaultValue
	}
	result, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return result
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if dur, err := time.ParseDuration(value); err == nil {
			return dur
		}
	}
	return defaultValue
}
