package rest

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

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
	log.Printf("SearchL2Device: searching for MAC='%s'", mac)
	if mac == "" {
		log.Printf("SearchL2Device: empty MAC parameter")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "mac parameter required"})
		return
	}

	device, err := h.app.GetL2Device(mac)
	if err != nil {
		log.Printf("SearchL2Device: device not found for MAC='%s', error: %v", mac, err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(SearchResponse{Success: true, Found: false})
		return
	}

	log.Printf("SearchL2Device: found device for MAC='%s'", mac)
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
	log.Printf("SearchL3Device: searching for IP='%s'", ip)
	if ip == "" {
		log.Printf("SearchL3Device: empty IP parameter")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "ip parameter required"})
		return
	}

	device, err := h.app.GetL3Device(ip)
	if err != nil {
		log.Printf("SearchL3Device: device not found for IP='%s', error: %v", ip, err)
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(SearchResponse{Success: true, Found: false})
		return
	}

	log.Printf("SearchL3Device: found device for IP='%s'", ip)
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
		log.Printf("UniversalSearch: invalid JSON error: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "invalid JSON"})
		return
	}

	query := strings.TrimSpace(body.Query)
	log.Printf("UniversalSearch: received query='%s'", query)
	if query == "" {
		log.Printf("UniversalSearch: empty query")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "query required"})
		return
	}

	// Parse query format: "mac:xx:xx:xx:xx:xx:xx" or "ip:x.x.x.x"
	var result interface{}
	var found bool
	var searchType string
	var searchValue string

	if strings.HasPrefix(strings.ToLower(query), "mac:") {
		searchType = "mac"
		searchValue = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(query), "mac:"))
		log.Printf("UniversalSearch: detected MAC search, value='%s'", searchValue)
		if searchValue == "" {
			log.Printf("UniversalSearch: empty MAC value")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "MAC address required after 'mac:'"})
			return
		}
		device, err := h.app.GetL2Device(searchValue)
		if err != nil {
			log.Printf("UniversalSearch: L2 device not found for MAC='%s', error: %v", searchValue, err)
		} else {
			log.Printf("UniversalSearch: found L2 device for MAC='%s'", searchValue)
			result = device
			found = true
		}
	} else if strings.HasPrefix(strings.ToLower(query), "ip:") {
		searchType = "ip"
		searchValue = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(query), "ip:"))
		log.Printf("UniversalSearch: detected IP search, value='%s'", searchValue)
		if searchValue == "" {
			log.Printf("UniversalSearch: empty IP value")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "IP address required after 'ip:'"})
			return
		}
		device, err := h.app.GetL3Device(searchValue)
		if err != nil {
			log.Printf("UniversalSearch: L3 device not found for IP='%s', error: %v", searchValue, err)
		} else {
			log.Printf("UniversalSearch: found L3 device for IP='%s'", searchValue)
			result = device
			found = true
		}
	} else {
		// Try to detect if it's a MAC or IP address
		log.Printf("UniversalSearch: trying auto-detection for query='%s'", query)
		if strings.Contains(query, ":") && !strings.Contains(query, ".") {
			// Looks like MAC address
			searchType = "mac"
			searchValue = query
			log.Printf("UniversalSearch: auto-detected MAC address")
			device, err := h.app.GetL2Device(query)
			if err != nil {
				log.Printf("UniversalSearch: L2 device not found for auto-detected MAC='%s', error: %v", query, err)
			} else {
				log.Printf("UniversalSearch: found L2 device for auto-detected MAC='%s'", query)
				result = device
				found = true
			}
		} else if strings.Contains(query, ".") {
			// Looks like IP address
			searchType = "ip"
			searchValue = query
			log.Printf("UniversalSearch: auto-detected IP address")
			device, err := h.app.GetL3Device(query)
			if err != nil {
				log.Printf("UniversalSearch: L3 device not found for auto-detected IP='%s', error: %v", query, err)
			} else {
				log.Printf("UniversalSearch: found L3 device for auto-detected IP='%s'", query)
				result = device
				found = true
			}
		} else {
			log.Printf("UniversalSearch: unable to detect query type for '%s'", query)
		}
	}

	if !found {
		log.Printf("UniversalSearch: no results found for %s='%s'", searchType, searchValue)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SearchResponse{Success: true, Found: false})
		return
	}

	log.Printf("UniversalSearch: returning success for %s='%s'", searchType, searchValue)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SearchResponse{
		Success: true, Found: true, Data: result, Count: 1,
	})
}

func (h *SearchHandler) GetAllL2Devices(w http.ResponseWriter, r *http.Request) {
	h.setCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	devices, err := h.app.GetAllL2Devices()
	if err != nil {
		log.Printf("Error getting all L2 devices: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "failed to retrieve devices"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SearchResponse{
		Success: true, Data: devices, Count: len(devices),
	})
}

func (h *SearchHandler) GetAllL3Devices(w http.ResponseWriter, r *http.Request) {
	h.setCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	devices, err := h.app.GetAllL3Devices()
	if err != nil {
		log.Printf("Error getting all L3 devices: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(SearchResponse{Success: false, Error: "failed to retrieve devices"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SearchResponse{
		Success: true, Data: devices, Count: len(devices),
	})
}

// Helper function to check if a scanner type exists in the list
func hasScannerType(scannerTypes []string, scannerType string) bool {
	for _, st := range scannerTypes {
		if st == scannerType {
			return true
		}
	}
	return false
}
