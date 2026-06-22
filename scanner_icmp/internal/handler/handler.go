package handler

import (
	"context"
	"encoding/json"
	"scanner_icmp/internal/config"
	"scanner_icmp/internal/scanner"
	"scanner_icmp/pkg/logger"
	"scanner_icmp/pkg/queue"
)

// AutoScannerInterface defines the interface for auto scanner control
type AutoScannerInterface interface {
	Start()
	Stop()
	IsRunning() bool
}

// AutoScannerInstance is a global reference to the auto scanner instance
var AutoScannerInstance AutoScannerInterface

func SetAutoScannerInstance(autoScanner AutoScannerInterface) {
	AutoScannerInstance = autoScanner
}

func HandleMessage(ctx context.Context, msg queue.Delivery, rabbitMQ *queue.RabbitMQ, log logger.Logger, cfg *config.Config) {
	var req queue.PingRequest
	if err := json.Unmarshal(msg.Body, &req); err != nil {
		log.Errorf("Failed to unmarshal Ping scan request: %v", err)
		return
	}

	// Handle auto-scan control commands
	if req.Command == "start" {
		log.Infof("Received start command for auto scanner")
		if AutoScannerInstance != nil {
			AutoScannerInstance.Start()
			// Send response confirming the command was processed
			if msg.ReplyTo != "" {
				response := queue.PingResponse{
					TaskID: req.TaskID,
					Status: "started",
				}
				sendControlResponse(rabbitMQ, msg, response, log)
			}
		} else {
			log.Error("Auto scanner instance is not set")
			if msg.ReplyTo != "" {
				response := queue.PingResponse{
					TaskID: req.TaskID,
					Status: "failed",
					Error:  "Auto scanner instance is not set",
				}
				sendControlResponse(rabbitMQ, msg, response, log)
			}
		}
		return
	}

	if req.Command == "stop" {
		log.Infof("=== RECEIVED STOP COMMAND FOR AUTO SCANNER ===")
		if AutoScannerInstance != nil {
			log.Infof("Auto scanner instance found, calling Stop()")
			AutoScannerInstance.Stop()
			log.Infof("Stop() called successfully")
			// Send response confirming the command was processed
			if msg.ReplyTo != "" {
				response := queue.PingResponse{
					TaskID: req.TaskID,
					Status: "stopped",
				}
				sendControlResponse(rabbitMQ, msg, response, log)
			}
		} else {
			log.Error("Auto scanner instance is not set")
			if msg.ReplyTo != "" {
				response := queue.PingResponse{
					TaskID: req.TaskID,
					Status: "failed",
					Error:  "Auto scanner instance is not set",
				}
				sendControlResponse(rabbitMQ, msg, response, log)
			}
		}
		return
	}

	log.Infof("Received Ping scan request for targets: %v", req.Targets)

	pingScanner := scanner.NewPingScanner(req.PingCount, cfg.PingTimeout)

	results := make([]scanner.PingResult, 0, len(req.Targets))
	for _, target := range req.Targets {
		select {
		case <-ctx.Done():
			return
		default:
			result := pingScanner.Ping(ctx, target)
			results = append(results, result)
		}
	}

	if msg.ReplyTo != "" {
		sendResponse(rabbitMQ, msg, req, results, log)
	}

	log.Infof("Ping scan completed, scanned %d targets", len(results))
}

func sendResponse(rabbitMQ *queue.RabbitMQ, msg queue.Delivery, req queue.PingRequest, results []scanner.PingResult, log logger.Logger) {
	response := queue.PingResponse{
		TaskID:  req.TaskID,
		Status:  "completed",
		Results: results,
	}

	if err := rabbitMQ.SendResponse(msg.ReplyTo, msg.CorrelationId, response); err != nil {
		log.Errorf("Failed to send RPC response: %v", err)
	} else {
		log.Infof("Successfully sent Ping response for task %s", response.TaskID)
	}
}

func sendControlResponse(rabbitMQ *queue.RabbitMQ, msg queue.Delivery, response queue.PingResponse, log logger.Logger) {
	log.Infof("Sending control response: TaskID=%s, Status=%s", response.TaskID, response.Status)
	if err := rabbitMQ.SendResponse(msg.ReplyTo, msg.CorrelationId, response); err != nil {
		log.Errorf("Failed to send control response: %v", err)
	} else {
		log.Infof("Successfully sent control response for task %s", response.TaskID)
	}
}

