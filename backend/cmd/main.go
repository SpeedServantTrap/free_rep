package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"
	"unsafe"

	"backend/domain/models"
	"backend/internal/application"
	database "backend/internal/infrastructure/database"
	rabbitmq "backend/internal/infrastructure/messaging"
	rest "backend/internal/presentation/http"
	wb "backend/internal/presentation/websocket"
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// appHolder holds a pointer to *application.App atomically so it can be
// swapped in once RabbitMQ is available without restarting the HTTP server.
var appHolder unsafe.Pointer // stores *application.App

func loadApp() *application.App {
	p := atomic.LoadPointer(&appHolder)
	if p == nil {
		return nil
	}
	return (*application.App)(p)
}

func storeApp(a *application.App) {
	atomic.StorePointer(&appHolder, unsafe.Pointer(a))
}

func main() {
	rabbitMQURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/")
	mongoURI    := getEnv("MONGODB_URI",      "mongodb://mongodb:27017")
	mongoDB     := getEnv("MONGODB_DATABASE", "network_scanner")

	// ── MongoDB ──────────────────────────────────────────────────────────────
	log.Println("[Main] Connecting to MongoDB…")
	db, err := database.NewDatabase(mongoURI, mongoDB)
	if err != nil {
		log.Fatalf("[Main] Failed to connect to MongoDB: %v", err)
	}
	defer db.Close()
	repo := database.NewRepository(db)
	log.Println("[Main] MongoDB connected")

	// ── HTTP routes that do NOT need RabbitMQ ────────────────────────────────
	historyHandler := rest.NewHistoryHandler(repo)
	searchHandler  := rest.NewSearchHandler(repo, nil) // app set later
	changesHandler := rest.NewChangesHandler(repo)

	// Change Detection endpoints
	http.HandleFunc("/api/changes",        changesHandler.GetChanges)
	http.HandleFunc("/api/changes/delete", changesHandler.DeleteChanges)
	http.HandleFunc("/api/changes/stream", changesHandler.StreamChanges)

	http.HandleFunc("/api/history/icmp",   historyHandler.GetICMPHistory)
	http.HandleFunc("/api/history/nmap",   historyHandler.GetNmapHistory)
	http.HandleFunc("/api/history/tcp",    historyHandler.GetTCPHistory)

	http.HandleFunc("/api/history/icmp/delete", historyHandler.DeleteICMPHistory)
	http.HandleFunc("/api/history/nmap/delete", historyHandler.DeleteNmapHistory)
	http.HandleFunc("/api/history/tcp/delete",  historyHandler.DeleteTCPHistory)

	// New L2/L3 device search endpoints
	http.HandleFunc("/api/search/l2", searchHandler.SearchL2Device)
	http.HandleFunc("/api/search/l3", searchHandler.SearchL3Device)
	http.HandleFunc("/api/search/universal", searchHandler.UniversalSearch)

	// New L2/L3 device display endpoints
	http.HandleFunc("/api/devices/l2", searchHandler.GetAllL2Devices)
	http.HandleFunc("/api/devices/l3", searchHandler.GetAllL3Devices)

	// ── /ws — proxied through appHolder; returns 503 while RabbitMQ not ready ─
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		app := loadApp()
		if app == nil {
			log.Printf("[WS] RabbitMQ not ready yet — returning 503")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "backend not ready, RabbitMQ connecting…",
			})
			return
		}
		wb.NewWSHandler(app).WsHandler(w, r)
	})

	// ── /health — always responds 200 ────────────────────────────────────────
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		ready := loadApp() != nil
		w.Header().Set("Content-Type", "application/json")
		if ready {
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "starting", "detail": "waiting for rabbitmq"})
		}
	})

	// ── Start HTTP server IMMEDIATELY ────────────────────────────────────────
	log.Println("[Main] HTTP server starting on :8080 (RabbitMQ connecting in background)")
	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalf("[Main] ListenAndServe error: %v", err)
		}
	}()

	// ── Connect to RabbitMQ in background ────────────────────────────────────
	log.Println("[Main] Starting RabbitMQ connection in background…")
	publisher, err := rabbitmq.GetRPCconnection(rabbitMQURL)
	if err != nil {
		log.Printf("[Main] WARNING: RabbitMQ connection failed after all retries: %v", err)
		log.Printf("[Main] Scan endpoints will be unavailable. Restarting container…")
		// Exit so Docker restarts us; HTTP history endpoints remain up until then
		os.Exit(1)
	}

	// ── Wire up the full app once RabbitMQ is available ──────────────────────
	app := application.NewApp(publisher, repo)
	storeApp(app)
	// also update searchHandler's app reference
	searchHandler.SetApp(app)
	log.Println("[Main] RabbitMQ connected — WebSocket scan endpoint is now active")

	// ── Change Events consumer ────────────────────────────────────────────────
	// Consumes from the `change_events` queue (published by the Python
	// change_detector service), saves each event to MongoDB and broadcasts
	// to all connected WebSocket clients via the Hub.
	go func() {
		deliveries, err := publisher.ConsumeChangeEvents("change_events")
		if err != nil {
			log.Printf("[ChangeEvents] Failed to start consumer: %v", err)
			return
		}
		log.Println("[ChangeEvents] Consumer started — waiting for change events…")

		for msg := range deliveries {
			var event models.ChangeEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				log.Printf("[ChangeEvents] Cannot parse event: %v — body: %s", err, string(msg.Body))
				msg.Nack(false, false) // dead-letter
				continue
			}

			// Broadcast to all connected WebSocket clients
			wb.GetHub().Broadcast(wb.Message{
				Type:   "change_event",
				Change: &event,
			})

			log.Printf("[ChangeEvents] Broadcasted [%s] %s — %s", event.Severity, event.EventType, event.Title)
			msg.Ack(false)
		}
		log.Println("[ChangeEvents] Delivery channel closed")
	}()

	// ── ARP Auto Scan Results consumer ────────────────────────────────────────
	// Consumes from the `arp_auto_scan_results` queue (published by the ARP scanner
	// auto-scanner), processes results and saves to MongoDB.
	go func() {
		deliveries, err := publisher.ConsumeChangeEvents("arp_auto_scan_results")
		if err != nil {
			log.Printf("[ARP AutoScan] Failed to start consumer: %v", err)
			return
		}
		log.Println("[ARP AutoScan] Consumer started — waiting for auto scan results…")

		for msg := range deliveries {
			log.Printf("[ARP AutoScan] Received auto scan result")
			var arpResp models.ARPResponse
			if err := json.Unmarshal(msg.Body, &arpResp); err != nil {
				log.Printf("[ARP AutoScan] Cannot parse ARP response: %v — body: %s", err, string(msg.Body))
				msg.Nack(false, false) // dead-letter
				continue
			}

			log.Printf("[ARP AutoScan] Processing ARP auto scan: TaskID=%s, Total=%d, Online=%d",
				arpResp.TaskID, arpResp.TotalCount, arpResp.OnlineCount)

			// Process the ARP response the same way as regular responses
			response := &models.Response{
				TaskID: arpResp.TaskID,
				Result: arpResp,
			}
			app.ProcessResponse(response)

			log.Printf("[ARP AutoScan] Processed auto scan result: %s", arpResp.TaskID)
			msg.Ack(false)
		}
		log.Println("[ARP AutoScan] Delivery channel closed")
	}()

	// Keep main goroutine alive
	select {}
}

// keep process alive by sleeping forever after http server launched
func keepAlive() {
	for {
		time.Sleep(24 * time.Hour)
	}
}
