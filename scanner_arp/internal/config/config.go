package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	RabbitMQURL         string
	ScannerName         string
	Timeout             time.Duration
	MaxRetries          int
	RetryDelay          time.Duration
	AutoScanEnabled     bool
	AutoScanInterface   string
	AutoScanIPRange     string
	AutoScanInterval    time.Duration
}

func Load() Config {
	err := godotenv.Load()
	if err != nil {
		log.Printf("No .env file found or error loading it: %v", err)
	}
	return Config{
		RabbitMQURL:         getEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"),
		ScannerName:         getEnv("SCANNER_NAME", "arp_scanner"),
		Timeout:             getDurationEnv("SCANNER_TIMEOUT", 500*time.Millisecond),
		MaxRetries:          getIntEnv("SCANNER_MAX_RETRIES", 1),
		RetryDelay:          getDurationEnv("SCANNER_RETRY_DELAY", 500*time.Millisecond),
		AutoScanEnabled:     getBoolEnv("ARP_AUTO_SCAN_ENABLED", false),
		AutoScanInterface:   getEnv("ARP_INTERFACE", "eth0"),
		AutoScanIPRange:     getEnv("ARP_IP_RANGE", "192.168.1.0/24"),
		AutoScanInterval:    getDurationEnv("ARP_SCAN_INTERVAL", 1*time.Minute),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
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
	value := getEnv(key, "")
	if value == "" {
		return defaultValue
	}
	result, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return result
}
