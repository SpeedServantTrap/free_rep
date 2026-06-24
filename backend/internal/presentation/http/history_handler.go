package rest

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"backend/domain/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type HistoryHandler struct {
	repo RepositoryInterface
}

type RepositoryInterface interface {
	SaveICMPHistory(record *models.ICMPHistoryRecord) error
	GetICMPHistory(limit int) ([]models.ICMPHistoryRecord, error)
	DeleteICMPHistory() error

	SaveNmapTcpUdpHistory(record *models.NmapTcpUdpHistoryRecord) error
	GetNmapTcpUdpHistory(limit int) ([]models.NmapTcpUdpHistoryRecord, error)
	DeleteNmapTcpUdpHistory() error

	SaveNmapOsDetectionHistory(record *models.NmapOsDetectionHistoryRecord) error
	GetNmapOsDetectionHistory(limit int) ([]models.NmapOsDetectionHistoryRecord, error)
	DeleteNmapOsDetectionHistory() error

	SaveNmapHostDiscoveryHistory(record *models.NmapHostDiscoveryHistoryRecord) error
	GetNmapHostDiscoveryHistory(limit int) ([]models.NmapHostDiscoveryHistoryRecord, error)
	DeleteNmapHostDiscoveryHistory() error

	SaveTCPHistory(record *models.TCPHistoryRecord) error
	GetTCPHistory(limit int) ([]models.TCPHistoryRecord, error)
	DeleteTCPHistory() error

	// New L2/L3 device methods
	GetAllL3Devices() ([]models.L3DeviceNew, error)
}

func NewHistoryHandler(repo RepositoryInterface) *HistoryHandler {
	return &HistoryHandler{repo: repo}
}

func (h *HistoryHandler) GetICMPHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get L3 devices and filter for ICMP scans
	l3Devices, err := h.repo.GetAllL3Devices()
	if err != nil {
		log.Printf("Error getting L3 devices: %v", err)
		response := models.HistoryResponse{
			Success: false,
			Error:   "Failed to retrieve ICMP history",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Convert L3 devices to ICMP history format
	var records []models.ICMPHistoryRecord
	for _, device := range l3Devices {
		// Only include devices that have ICMP scanner type and packets reached data
		if hasScannerType(device.ScannerTypes, "icmp") && len(device.PacketsReached) > 0 {
			// Parse packets reached (e.g., "5/5" -> sent: 5, received: 5)
			for _, packetsStr := range device.PacketsReached {
				var sent, received int
				fmt.Sscanf(packetsStr, "%d/%d", &received, &sent)
				lossPercent := 0.0
				if sent > 0 {
					lossPercent = float64(sent-received) / float64(sent) * 100
				}

				record := models.ICMPHistoryRecord{
					ID:        primitive.NewObjectID(),
					ScanType:  "icmp",
					CreatedAt: device.LastSeen,
					Results: []models.ICMPResult{
						{
							Target:              device.ID,
							Address:             device.ID,
							PacketsSent:         sent,
							PacketsReceived:     received,
							PacketLossPercent:   lossPercent,
						},
					},
					Status: "completed",
				}
				records = append(records, record)
			}
		}
	}

	response := models.HistoryResponse{
		Success: true,
		Data:    records,
		Count:   len(records),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *HistoryHandler) DeleteICMPHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := h.repo.DeleteICMPHistory()
	if err != nil {
		log.Printf("Error deleting ICMP history: %v", err)
		response := models.HistoryResponse{
			Success: false,
			Error:   "Failed to delete ICMP history",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := models.HistoryResponse{
		Success: true,
		Data:    "ICMP history deleted successfully",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *HistoryHandler) GetNmapHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	scanType := r.URL.Query().Get("type")
	if scanType == "" {
		scanType = "all"
	}

	// Get L3 devices
	l3Devices, err := h.repo.GetAllL3Devices()
	if err != nil {
		log.Printf("Error getting L3 devices: %v", err)
		response := models.HistoryResponse{
			Success: false,
			Error:   "Failed to retrieve Nmap history",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	result := make(map[string]interface{})

	if scanType == "all" || scanType == "tcp_udp" {
		var tcpUdpRecords []models.NmapTcpUdpHistoryRecord
		for _, device := range l3Devices {
			// Only include devices that have NMAP scanner type and TCP/UDP ports
			if hasScannerType(device.ScannerTypes, "nmap") && (len(device.TCPOpenPorts) > 0 || len(device.UDPOpenPorts) > 0) {
				// Convert L3 device ports to Nmap format
				var allPorts []uint16
				var protocols []string
				var states []string
				var serviceNames []string

				// Add TCP ports
				for _, portStr := range device.TCPOpenPorts {
					if portNum, err := strconv.ParseUint(portStr, 10, 16); err == nil {
						allPorts = append(allPorts, uint16(portNum))
						protocols = append(protocols, "tcp")
						states = append(states, "open")
						serviceNames = append(serviceNames, "unknown") // L3 doesn't store service names
					}
				}

				// Add UDP ports
				for _, portStr := range device.UDPOpenPorts {
					if portNum, err := strconv.ParseUint(portStr, 10, 16); err == nil {
						allPorts = append(allPorts, uint16(portNum))
						protocols = append(protocols, "udp")
						states = append(states, "open")
						serviceNames = append(serviceNames, "unknown") // L3 doesn't store service names
					}
				}

				if len(allPorts) > 0 {
					record := models.NmapTcpUdpHistoryRecord{
						ID:       primitive.NewObjectID(),
						ScanType: "nmap_tcp_udp",
						IP:       device.ID,
						Host:     device.ID,
						PortInfo: []models.NmapPortTcpUdpInfo{
							{
								Status:      "up",
								AllPorts:    allPorts,
								Protocols:   protocols,
								State:       states,
								ServiceName: serviceNames,
							},
						},
						Status:    "completed",
						CreatedAt: device.LastSeen,
					}
					tcpUdpRecords = append(tcpUdpRecords, record)
				}
			}
		}
		result["tcp_udp"] = tcpUdpRecords
	}

	if scanType == "all" || scanType == "os_detection" {
		var osRecords []models.NmapOsDetectionHistoryRecord
		for _, device := range l3Devices {
			// Only include devices that have NMAP scanner type and OS data
			if hasScannerType(device.ScannerTypes, "nmap") && device.OS != "" && device.OS != "-" {
				record := models.NmapOsDetectionHistoryRecord{
					ID:        primitive.NewObjectID(),
					ScanType:  "nmap_os_detection",
					IP:        device.ID,
					Host:      device.ID,
					Name:      device.OS,
					Accuracy:  0, // L3 doesn't store accuracy
					Status:    "completed",
					CreatedAt: device.LastSeen,
				}
				osRecords = append(osRecords, record)
			}
		}
		result["os_detection"] = osRecords
	}

	if scanType == "all" || scanType == "host_discovery" {
		var hostRecords []models.NmapHostDiscoveryHistoryRecord
		for _, device := range l3Devices {
			// Only include devices that have NMAP scanner type and DNS data
			if hasScannerType(device.ScannerTypes, "nmap") && device.DNS != "" && device.DNS != "-" {
				record := models.NmapHostDiscoveryHistoryRecord{
					ID:        primitive.NewObjectID(),
					ScanType:  "nmap_host_discovery",
					IP:        device.ID,
					Host:      device.ID,
					DNS:       device.DNS,
					HostUP:    1, // Device exists in L3, so it's up
					HostTotal: 1,
					Status:    "completed",
					CreatedAt: device.LastSeen,
				}
				hostRecords = append(hostRecords, record)
			}
		}
		result["host_discovery"] = hostRecords
	}

	response := models.HistoryResponse{
		Success: true,
		Data:    result,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *HistoryHandler) DeleteNmapHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	scanType := r.URL.Query().Get("type")
	if scanType == "" {
		scanType = "all"
	}

	result := make(map[string]string)

	if scanType == "all" || scanType == "tcp_udp" {
		err := h.repo.DeleteNmapTcpUdpHistory()
		if err != nil {
			result["tcp_udp"] = "Failed to delete TCP/UDP history"
		} else {
			result["tcp_udp"] = "Deleted successfully"
		}
	}

	if scanType == "all" || scanType == "os_detection" {
		err := h.repo.DeleteNmapOsDetectionHistory()
		if err != nil {
			result["os_detection"] = "Failed to delete OS Detection history"
		} else {
			result["os_detection"] = "Deleted successfully"
		}
	}

	if scanType == "all" || scanType == "host_discovery" {
		err := h.repo.DeleteNmapHostDiscoveryHistory()
		if err != nil {
			result["host_discovery"] = "Failed to delete Host Discovery history"
		} else {
			result["host_discovery"] = "Deleted successfully"
		}
	}

	response := models.HistoryResponse{
		Success: true,
		Data:    result,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *HistoryHandler) GetTCPHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get L3 devices and filter for TCP scans with banners
	l3Devices, err := h.repo.GetAllL3Devices()
	if err != nil {
		log.Printf("Error getting L3 devices: %v", err)
		response := models.HistoryResponse{
			Success: false,
			Error:   "Failed to retrieve TCP history",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Convert L3 devices to TCP history format
	var records []models.TCPHistoryRecord
	for _, device := range l3Devices {
		// Only include devices that have TCP scanner type and per-port banner data
		if hasScannerType(device.ScannerTypes, "tcp") && len(device.TCPBanners) > 0 {
			for port, banner := range device.TCPBanners {
				if port == "" || banner == "" {
					continue
				}
				record := models.TCPHistoryRecord{
					ID:          primitive.NewObjectID(),
					ScanType:    "tcp",
					Host:        device.ID,
					Port:        port,
					DecodedText: banner,
					Status:      "completed",
					CreatedAt:   device.LastSeen,
				}
				records = append(records, record)
			}
		}
	}

	response := models.HistoryResponse{
		Success: true,
		Data:    records,
		Count:   len(records),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *HistoryHandler) DeleteTCPHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := h.repo.DeleteTCPHistory()
	if err != nil {
		log.Printf("Error deleting TCP history: %v", err)
		response := models.HistoryResponse{
			Success: false,
			Error:   "Failed to delete TCP history",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := models.HistoryResponse{
		Success: true,
		Data:    "TCP history deleted successfully",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
