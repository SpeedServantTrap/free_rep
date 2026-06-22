package service

import (
	"scanner_icmp/internal/config"
	"scanner_icmp/internal/scanner"
	"scanner_icmp/pkg/logger"
	"scanner_icmp/pkg/queue"
	"context"
	"encoding/json"
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

	// Create new context and stop channel for this run
	ctx, cancel := context.WithCancel(context.Background())
	as.cancelFunc = cancel

	// Recreate stopChan if it was closed
	select {
	case <-as.stopChan:
		// Channel is closed, create a new one
		as.stopChan = make(chan struct{})
	default:
		// Channel is open, reuse it
	}

	as.running = true
	as.log.Info("Starting auto ICMP scanner")

	go as.runLoop(ctx)
}

func (as *AutoScanner) Stop() {
	as.log.Infof("=== STOP METHOD CALLED ===")
	as.mu.Lock()
	defer as.mu.Unlock()

	as.log.Infof("Current running state: %v", as.running)
	if !as.running {
		as.log.Info("Auto scanner is not running")
		return
	}

	as.log.Infof("Setting running to false, closing stopChan, and cancelling context")
	as.running = false

	// Cancel the context to stop any ongoing scans
	if as.cancelFunc != nil {
		as.cancelFunc()
	}

	close(as.stopChan)
	as.stopChan = make(chan struct{})
	as.log.Info("Stopped auto ICMP scanner successfully")
}

func (as *AutoScanner) IsRunning() bool {
	as.mu.Lock()
	defer as.mu.Unlock()
	return as.running
}

func (as *AutoScanner) runLoop(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			as.log.Errorf("Panic in runLoop: %v", r)
		}
	}()

	ticker := time.NewTicker(as.cfg.AutoScanInterval)
	defer ticker.Stop()

	as.log.Infof("Auto scanner loop started with interval: %v", as.cfg.AutoScanInterval)

	// Perform initial scan immediately
	as.performScan(ctx)

	for {
		select {
		case <-ticker.C:
			as.performScan(ctx)
		case <-as.stopChan:
			as.log.Info("Auto scanner loop stopped")
			return
		case <-ctx.Done():
			as.log.Info("Auto scanner loop stopped via context cancellation")
			return
		}
	}
}

func (as *AutoScanner) performScan(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			as.log.Errorf("Panic in performScan: %v", r)
		}
	}()

	as.mu.Lock()
	running := as.running
	as.mu.Unlock()

	if !running {
		as.log.Info("Auto scan is stopped, skipping scan")
		return
	}

	as.log.Info("Performing scheduled ICMP scan")

	// Check if context is cancelled before starting scan
	select {
	case <-ctx.Done():
		as.log.Info("Context cancelled before scan, skipping scan")
		return
	default:
	}

	pingScanner := scanner.NewPingScanner(as.cfg.AutoScanPingCount, as.cfg.PingTimeout)
	results := make([]scanner.PingResult, 0, len(as.cfg.AutoScanTargets))

	for _, target := range as.cfg.AutoScanTargets {
		select {
		case <-ctx.Done():
			as.log.Info("Context cancelled during scan, stopping")
			return
		default:
			result := pingScanner.Ping(ctx, target)
			results = append(results, result)
		}
	}

	// Check again after scan in case stop was called during scan
	as.mu.Lock()
	wasRunning := as.running
	as.mu.Unlock()

	if !wasRunning {
		as.log.Info("Auto scan was stopped during scan, discarding results")
		return
	}

	as.log.Infof("Scheduled ICMP scan completed, scanned %d targets", len(results))

	// Convert to queue format and publish to auto scan results queue
	response := queue.PingResponse{
		TaskID:  "auto_scan_" + time.Now().Format("20060102_150405"),
		Status:  "completed",
		Results: results,
	}

	// Publish to the auto scan results queue (backend will consume and save to DB)
	queueName := "icmp_auto_scan_results"
	body, err := json.Marshal(response)
	if err != nil {
		as.log.Errorf("Failed to marshal auto scan response: %v", err)
		return
	}

	// Declare queue for auto scan results
	_, err = as.rabbitMQ.Channel().QueueDeclare(
		queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		as.log.Errorf("Failed to declare auto scan results queue: %v", err)
		return
	}

	// Publish to the auto scan results queue
	err = as.rabbitMQ.Channel().Publish(
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		as.log.Errorf("Failed to publish auto scan results to RabbitMQ: %v", err)
	} else {
		as.log.Infof("Successfully published auto scan results to queue: %s", queueName)
	}
}
