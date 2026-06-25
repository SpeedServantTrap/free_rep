package adapter

import (
	"context"
	"fmt"
	"scanner_nmap/internal/config"
	"scanner_nmap/internal/handler"
	service "scanner_nmap/internal/service"
	"scanner_nmap/pkg/logger"
	"scanner_nmap/pkg/queue"
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

	autoScanner := service.NewAutoScanner(cfg, rabbitMQ, log)
	handler.SetAutoScannerInstance(autoScanner)
	if cfg.AutoScanEnabled {
		log.Info("Auto scan enabled from config, starting auto scanner...")
		autoScanner.Start()
	}

	msgs, err := rabbitMQ.ConsumeScanRequests(ctx)
	if err != nil {
		return fmt.Errorf("failed to consume scan requests: %w", err)
	}

	log.Infof("Scanner service started (%s), waiting for tasks...", cfg.ScannerName)
	return processMessages(ctx, msgs, rabbitMQ, log, autoScanner)
}

func processMessages(ctx context.Context, msgs <-chan queue.Delivery, rabbitMQ *queue.RabbitMQ, log logger.Logger, autoScanner *service.AutoScanner) error {
	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return nil
			}
			// Run in goroutine so long scans don't block new messages.
			go handler.HandleMessage(ctx, msg, rabbitMQ, log)

		case <-ctx.Done():
			if autoScanner.IsRunning() {
				autoScanner.Stop()
			}
			return nil
		}
	}
}
