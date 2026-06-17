package rest

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"backend/domain/models"
	api "backend/internal/application"
	"backend/internal/application/services"
)

type SearchHandler struct {
	repo services.SearchRepository
	app  *api.App
}

func NewSearchHandler(repo services.SearchRepository, app *api.App) *SearchHandler {
	return &SearchHandler{repo: repo, app: app}
}

// SetApp позволяет установить app после старта HTTP-сервера (когда RabbitMQ наконец подключился).
func (h *SearchHandler) SetApp(app *api.App) {
	h.app = app
}

func (h *SearchHandler) setCORS(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

type SearchResponse struct {
	Success   bool        `json:"success"`
	Found     bool        `json:"found"`
	FromCache bool        `json:"from_cache"`
	TaskID    string      `json:"task_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Count     int         `json:"count,omitempty"`
	Error     string      `json:"error,omitempty"`
}

func (h *SearchHandler) SearchICMP(w http.ResponseWriter, r *http.Request) {
	h.setCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Targets   []string `json:"targets"`
		PingCount int      `json:"ping_count"`
		Limit     int      `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "invalid JSON"})
		return
	}
	if len(body.Targets) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "targets required"})
		return
	}
	if body.PingCount <= 0 {
		body.PingCount = 4
	}
	if body.Limit <= 0 {
		body.Limit = 20
	}

	records, err := h.repo.GetICMPHistoryByTargets(body.Targets, body.Limit)
	if err != nil {
		log.Printf("Search ICMP by targets: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "search failed"})
		return
	}
	if len(records) == 0 {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SearchResponse{Success: true, Found: false})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SearchResponse{
		Success: true, Found: true, FromCache: true,
		Data: records, Count: len(records),
	})
}

func (h *SearchHandler) SearchNmap(w http.ResponseWriter, r *http.Request) {
	h.setCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		ScanMethod   string `json:"scan_method"`
		IP           string `json:"ip"`
		Ports        string `json:"ports"`
		ScannerType  string `json:"scanner_type"`
		Limit        int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "invalid JSON"})
		return
	}
	if body.IP == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "ip required"})
		return
	}
	if body.Limit <= 0 {
		body.Limit = 20
	}
	if body.ScanMethod == "" {
		body.ScanMethod = "tcp_udp_scan"
	}

	switch body.ScanMethod {
	case "tcp_udp_scan":
		records, err := h.repo.GetNmapTcpUdpHistoryByIP(body.IP, body.Limit)
		if err != nil {
			log.Printf("Search Nmap TCP/UDP by IP: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "search failed"})
			return
		}
		if len(records) == 0 {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(SearchResponse{Success: true, Found: false})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SearchResponse{Success: true, Found: true, FromCache: true, Data: records, Count: len(records)})

	case "os_detection":
		records, err := h.repo.GetNmapOsDetectionHistoryByIP(body.IP, body.Limit)
		if err != nil {
			log.Printf("Search Nmap OS by IP: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "search failed"})
			return
		}
		if len(records) == 0 {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(SearchResponse{Success: true, Found: false})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SearchResponse{Success: true, Found: true, FromCache: true, Data: records, Count: len(records)})

	case "host_discovery":
		records, err := h.repo.GetNmapHostDiscoveryHistoryByIP(body.IP, body.Limit)
		if err != nil {
			log.Printf("Search Nmap HostDiscovery by IP: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "search failed"})
			return
		}
		if len(records) == 0 {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(SearchResponse{Success: true, Found: false})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SearchResponse{Success: true, Found: true, FromCache: true, Data: records, Count: len(records)})

	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "unsupported scan_method"})
	}
}

func (h *SearchHandler) SearchARP(w http.ResponseWriter, r *http.Request) {
	h.setCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		InterfaceName string `json:"interface_name"`
		IPRange       string `json:"ip_range"`
		Limit         int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "invalid JSON"})
		return
	}
	if body.InterfaceName == "" || body.IPRange == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "interface_name and ip_range required"})
		return
	}
	if body.Limit <= 0 {
		body.Limit = 20
	}

	records, err := h.repo.GetARPHistoryByIPRange(body.IPRange, body.Limit)
	if err != nil {
		log.Printf("Search ARP by ip_range: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "search failed"})
		return
	}
	if len(records) == 0 {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SearchResponse{Success: true, Found: false})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SearchResponse{Success: true, Found: true, FromCache: true, Data: records, Count: len(records)})
}

func (h *SearchHandler) SearchTCP(w http.ResponseWriter, r *http.Request) {
	h.setCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Host  string `json:"host"`
		Port  string `json:"port"`
		Limit int    `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "invalid JSON"})
		return
	}
	if body.Host == "" || body.Port == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "host and port required"})
		return
	}
	if body.Limit <= 0 {
		body.Limit = 20
	}

	records, err := h.repo.GetTCPHistoryByHostPort(body.Host, body.Port, body.Limit)
	if err != nil {
		log.Printf("Search TCP by host/port: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "search failed"})
		return
	}
	if len(records) == 0 {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SearchResponse{Success: true, Found: false})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SearchResponse{Success: true, Found: true, FromCache: true, Data: records, Count: len(records)})
}

func (h *SearchHandler) GetICMPHistoryByID(w http.ResponseWriter, r *http.Request) {
	h.setCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.HistoryResponse{Success: false, Error: "id required"})
		return
	}
	rec, err := h.repo.GetICMPHistoryByID(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.HistoryResponse{Success: false, Error: "not found"})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.HistoryResponse{Success: true, Data: rec})
}

func (h *SearchHandler) GetNmapTcpUdpHistoryByID(w http.ResponseWriter, r *http.Request) {
	h.serveHistoryByID(w, r, "nmap_tcp_udp", func(id string) (interface{}, error) {
		return h.repo.GetNmapTcpUdpHistoryByID(id)
	})
}

func (h *SearchHandler) GetNmapOsDetectionHistoryByID(w http.ResponseWriter, r *http.Request) {
	h.serveHistoryByID(w, r, "nmap_os", func(id string) (interface{}, error) {
		return h.repo.GetNmapOsDetectionHistoryByID(id)
	})
}

func (h *SearchHandler) GetNmapHostDiscoveryHistoryByID(w http.ResponseWriter, r *http.Request) {
	h.serveHistoryByID(w, r, "nmap_host", func(id string) (interface{}, error) {
		return h.repo.GetNmapHostDiscoveryHistoryByID(id)
	})
}

func (h *SearchHandler) GetARPHistoryByID(w http.ResponseWriter, r *http.Request) {
	h.serveHistoryByID(w, r, "arp", func(id string) (interface{}, error) {
		return h.repo.GetARPHistoryByID(id)
	})
}

func (h *SearchHandler) GetTCPHistoryByID(w http.ResponseWriter, r *http.Request) {
	h.serveHistoryByID(w, r, "tcp", func(id string) (interface{}, error) {
		return h.repo.GetTCPHistoryByID(id)
	})
}

func (h *SearchHandler) serveHistoryByID(w http.ResponseWriter, r *http.Request, _ string, get func(id string) (interface{}, error)) {
	h.setCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.HistoryResponse{Success: false, Error: "id required"})
		return
	}
	rec, err := get(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.HistoryResponse{Success: false, Error: "not found"})
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.HistoryResponse{Success: true, Data: rec})
}

// ──────────────────────────────────────────────────────────────────────────────
// New L2/L3 Device Search Endpoints
// ──────────────────────────────────────────────────────────────────────────────

func (h *SearchHandler) SearchL2Device(w http.ResponseWriter, r *http.Request) {
	h.setCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mac := strings.TrimSpace(r.URL.Query().Get("mac"))
	if mac == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "mac parameter required"})
		return
	}

	device, err := h.app.GetL2Device(mac)
	if err != nil {
		log.Printf("Search L2 device by MAC: %v", err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(SearchResponse{Success: true, Found: false})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SearchResponse{
		Success: true, Found: true, Data: device, Count: 1,
	})
}

func (h *SearchHandler) SearchL3Device(w http.ResponseWriter, r *http.Request) {
	h.setCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := strings.TrimSpace(r.URL.Query().Get("ip"))
	if ip == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "ip parameter required"})
		return
	}

	device, err := h.app.GetL3Device(ip)
	if err != nil {
		log.Printf("Search L3 device by IP: %v", err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(SearchResponse{Success: true, Found: false})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SearchResponse{
		Success: true, Found: true, Data: device, Count: 1,
	})
}

func (h *SearchHandler) UniversalSearch(w http.ResponseWriter, r *http.Request) {
	h.setCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "invalid JSON"})
		return
	}

	query := strings.TrimSpace(body.Query)
	if query == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "query required"})
		return
	}

	// Parse query format: "mac:xx:xx:xx:xx:xx:xx" or "ip:x.x.x.x"
	var result interface{}
	var found bool

	if strings.HasPrefix(strings.ToLower(query), "mac:") {
		mac := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(query), "mac:"))
		device, err := h.app.GetL2Device(mac)
		if err == nil {
			result = device
			found = true
		}
	} else if strings.HasPrefix(strings.ToLower(query), "ip:") {
		ip := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(query), "ip:"))
		device, err := h.app.GetL3Device(ip)
		if err == nil {
			result = device
			found = true
		}
	} else {
		// Try to detect if it's a MAC or IP address
		if strings.Contains(query, ":") && !strings.Contains(query, ".") {
			// Looks like MAC address
			device, err := h.app.GetL2Device(query)
			if err == nil {
				result = device
				found = true
			}
		} else if strings.Contains(query, ".") {
			// Looks like IP address
			device, err := h.app.GetL3Device(query)
			if err == nil {
				result = device
				found = true
			}
		}
	}

	if !found {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SearchResponse{Success: true, Found: false})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SearchResponse{
		Success: true, Found: true, Data: result, Count: 1,
	})
}
