package service

import (
	"context"
	"encoding/json"
	"fmt"
	"scanner_nmap/internal/config"
	"scanner_nmap/internal/domain"
	"scanner_nmap/internal/usecases"
	"scanner_nmap/pkg/logger"
	"scanner_nmap/pkg/queue"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

type AutoScanner struct {
	cfg        config.Config
	rabbitMQ   *queue.RabbitMQ
	log        logger.Logger
	mu         sync.Mutex
	running    bool
	stopChan   chan struct{}
	cancelFunc context.CancelFunc
}

func NewAutoScanner(cfg config.Config, rabbitMQ *queue.RabbitMQ, log logger.Logger) *AutoScanner {
	return &AutoScanner{
		cfg:      cfg,
		rabbitMQ: rabbitMQ,
		log:      log,
		stopChan: make(chan struct{}),
	}
}

func (as *AutoScanner) Start() {
	as.mu.Lock()
	defer as.mu.Unlock()

	if as.running {
		as.log.Info("Auto scanner is already running")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	as.cancelFunc = cancel

	select {
	case <-as.stopChan:
		as.stopChan = make(chan struct{})
	default:
	}

	as.running = true
	as.log.Info("Starting auto Nmap scanner")
	go as.runLoop(ctx)
}

func (as *AutoScanner) Stop() {
	as.mu.Lock()
	defer as.mu.Unlock()

	if !as.running {
		as.log.Info("Auto scanner is not running")
		return
	}

	as.running = false
	if as.cancelFunc != nil {
		as.cancelFunc()
	}
	close(as.stopChan)
	as.stopChan = make(chan struct{})
	as.log.Info("Stopped auto Nmap scanner")
}

func (as *AutoScanner) IsRunning() bool {
	as.mu.Lock()
	defer as.mu.Unlock()
	return as.running
}

func (as *AutoScanner) runLoop(ctx context.Context) {
	ticker := time.NewTicker(as.cfg.AutoScanInterval)
	defer ticker.Stop()

	as.log.Infof("Auto Nmap scanner loop started with interval: %v", as.cfg.AutoScanInterval)
	as.performScan(ctx)

	for {
		select {
		case <-ticker.C:
			as.performScan(ctx)
		case <-as.stopChan:
			as.log.Info("Auto Nmap scanner loop stopped")
			return
		case <-ctx.Done():
			as.log.Info("Auto Nmap scanner loop stopped via context cancellation")
			return
		}
	}
}

func (as *AutoScanner) performScan(ctx context.Context) {
	as.mu.Lock()
	running := as.running
	as.mu.Unlock()
	if !running {
		return
	}
	if as.cfg.AutoScanInput == "" {
		as.log.Info("Auto Nmap input is empty, skipping scheduled scan")
		return
	}

	taskID := "auto_scan_" + time.Now().Format("20060102_150405")
	as.log.Infof("Performing scheduled Nmap comprehensive scan: task=%s input=%s", taskID, as.cfg.AutoScanInput)

	finalResponse, err := usecases.ComprehensiveScannerStream(ctx, domain.ComprehensiveScanRequest{
		TaskID:     taskID,
		Input:      as.cfg.AutoScanInput,
		ScanMethod: "comprehensive_scan",
	}, func(target domain.ComprehensiveTargetResult) {
		partial := domain.ComprehensiveScanResponse{
			TaskID:  taskID,
			Results: []domain.ComprehensiveTargetResult{target},
			Status:  "partial",
		}
		if err := as.publishAutoResult(partial); err != nil {
			as.log.Errorf("Failed to publish partial auto Nmap result: %v", err)
		}
	})
	if err != nil {
		as.log.Errorf("Scheduled Nmap scan failed: %v", err)
		_ = as.publishAutoResult(domain.ComprehensiveScanResponse{TaskID: taskID, Status: "failed", Error: err.Error()})
		return
	}

	if err := as.publishAutoResult(finalResponse); err != nil {
		as.log.Errorf("Failed to publish completed auto Nmap result: %v", err)
	}
}

func (as *AutoScanner) publishAutoResult(response domain.ComprehensiveScanResponse) error {
	body, err := json.Marshal(response)
	if err != nil {
		return err
	}

	queueName := "nmap_auto_scan_results"
	_, err = as.rabbitMQ.Channel().QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare auto result queue: %w", err)
	}

	return as.rabbitMQ.Channel().Publish(
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}
