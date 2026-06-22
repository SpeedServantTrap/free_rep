package service

import (
	"context"
	"fmt"
	"scanner_icmp/internal/config"
	"scanner_icmp/internal/handler"
	"scanner_icmp/pkg/logger"
	"scanner_icmp/pkg/queue"
)

func Run(ctx context.Context, cfg config.Config, log logger.Logger) error {
	rabbitMQ, err := queue.NewRabbitMQ(queue.RabbitMQConfig{
		URL:          cfg.RabbitMQURL,
		ScannerQueue: cfg.ScannerName,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	defer rabbitMQ.Close()

	// Initialize auto scanner
	autoScanner := NewAutoScanner(cfg, rabbitMQ, log)
	handler.SetAutoScannerInstance(autoScanner)

	// Start auto scanner if enabled in config
	if cfg.AutoScanEnabled {
		log.Info("Auto scan enabled from config, starting auto scanner...")
		autoScanner.Start()
	}

	msgs, err := rabbitMQ.ConsumeScanRequests(ctx)
	if err != nil {
		return fmt.Errorf("failed to consume scan requests: %w", err)
	}

	log.Infof("Ping Scanner service started (%s), waiting for tasks...", cfg.ScannerName)
	return processMessages(ctx, msgs, rabbitMQ, log, autoScanner)
}

func processMessages(ctx context.Context, msgs <-chan queue.Delivery, rabbitMQ *queue.RabbitMQ, log logger.Logger, autoScanner *AutoScanner) error {
	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return nil
			}
			handler.HandleMessage(ctx, msg, rabbitMQ, log, nil)

		case <-ctx.Done():
			// Stop auto scanner on context shutdown
			if autoScanner.IsRunning() {
				autoScanner.Stop()
			}
			return nil
		}
	}
}

