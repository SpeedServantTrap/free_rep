package services

import (
	"backend/domain/models"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type SearchRepository interface {
	RepositoryInterface
	GetICMPHistoryByTargets(targets []string, limit int) ([]models.ICMPHistoryRecord, error)
	GetICMPHistoryByID(id string) (*models.ICMPHistoryRecord, error)
	GetNmapTcpUdpHistoryByIP(ip string, limit int) ([]models.NmapTcpUdpHistoryRecord, error)
	GetNmapTcpUdpHistoryByID(id string) (*models.NmapTcpUdpHistoryRecord, error)
	GetNmapOsDetectionHistoryByIP(ip string, limit int) ([]models.NmapOsDetectionHistoryRecord, error)
	GetNmapOsDetectionHistoryByID(id string) (*models.NmapOsDetectionHistoryRecord, error)
	GetNmapHostDiscoveryHistoryByIP(ip string, limit int) ([]models.NmapHostDiscoveryHistoryRecord, error)
	GetNmapHostDiscoveryHistoryByID(id string) (*models.NmapHostDiscoveryHistoryRecord, error)
	GetTCPHistoryByHostPort(host, port string, limit int) ([]models.TCPHistoryRecord, error)
	GetTCPHistoryByID(id string) (*models.TCPHistoryRecord, error)
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
	SaveOrUpdateL2Device(device *models.L2DeviceNew) error
	GetL2Device(mac string) (*models.L2DeviceNew, error)
	GetAllL2Devices() ([]models.L2DeviceNew, error)
	DeleteAllL2Devices() error
	DropL2Collection() error

	SaveOrUpdateL3Device(device *models.L3DeviceNew) error
	GetL3Device(ip string) (*models.L3DeviceNew, error)
	GetAllL3Devices() ([]models.L3DeviceNew, error)
	DeleteAllL3Devices() error
	DropL3Collection() error
}

// DeviceRepository — removed: L2/L3 inventory is now handled by the
// l2_devices / l3_devices collection structure, not a separate interface.

type HistoryService struct {
	repo         RepositoryInterface
	requestCache map[string]interface{}
	mu           sync.RWMutex
}

func NewHistoryService(repo RepositoryInterface) *HistoryService {
	return &HistoryService{
		repo:         repo,
		requestCache: make(map[string]interface{}),
	}
}

func (hs *HistoryService) CacheRequest(taskID string, request interface{}) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	hs.requestCache[taskID] = request
}

func (hs *HistoryService) GetCachedRequest(taskID string) interface{} {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.requestCache[taskID]
}

func (hs *HistoryService) RemoveCachedRequest(taskID string) {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	delete(hs.requestCache, taskID)
}

func (hs *HistoryService) GetRepo() RepositoryInterface {
	return hs.repo
}

func (hs *HistoryService) SaveARPResponse(result models.ARPResponse) {
	log.Printf("Processing ARP response: TaskID=%s, Total=%d, Online=%d, Offline=%d",
		result.TaskID, result.TotalCount, result.OnlineCount, result.OfflineCount)

	if hs.repo == nil {
		return
	}

	hs.ProcessARPToL2Devices(result)
	hs.LinkARPToL3Devices(result)
	hs.RemoveCachedRequest(result.TaskID)
}

func (hs *HistoryService) SaveICMPResponse(result models.ICMPResponse) {
	if hs.repo == nil {
		return
	}

	var targets []string
	var pingCount int
	if cachedReq := hs.GetCachedRequest(result.TaskID); cachedReq != nil {
		if icmpReq, ok := cachedReq.(models.ICMPRequest); ok {
			targets = icmpReq.Targets
			pingCount = icmpReq.PingCount
		}
	}

	historyRecord := &models.ICMPHistoryRecord{
		TaskID:    result.TaskID,
		Targets:   targets,
		PingCount: pingCount,
		Status:    result.Status,
		Results:   result.Results,
		Error:     result.Error,
	}

	if err := hs.repo.SaveICMPHistory(historyRecord); err != nil {
		log.Printf("Failed to save ICMP history: %v", err)
	} else {
		log.Printf("Successfully saved ICMP history for task %s", result.TaskID)
		hs.RemoveCachedRequest(result.TaskID)
	}
}

func (hs *HistoryService) SaveNmapTcpUdpResponse(result models.NmapTcpUdpResponse) {
	if hs.repo == nil {
		return
	}

	var ip, scannerType, ports string
	if cachedReq := hs.GetCachedRequest(result.TaskID); cachedReq != nil {
		if tcpUdpReq, ok := cachedReq.(models.NmapTcpUdpRequest); ok {
			ip = tcpUdpReq.IP
			scannerType = tcpUdpReq.ScannerType
			ports = tcpUdpReq.Ports
		}
	}

	if ip == "" && result.Host != "" {
		ip = result.Host
	}

	historyRecord := &models.NmapTcpUdpHistoryRecord{
		TaskID:      result.TaskID,
		IP:          ip,
		ScannerType: scannerType,
		Ports:       ports,
		Host:        result.Host,
		PortInfo:    result.PortInfo,
		Status:      result.Status,
		Error:       result.Error,
	}

	if err := hs.repo.SaveNmapTcpUdpHistory(historyRecord); err != nil {
		log.Printf("Failed to save Nmap TCP/UDP history: %v", err)
	} else {
		log.Printf("Successfully saved Nmap TCP/UDP history for task %s", result.TaskID)
		hs.RemoveCachedRequest(result.TaskID)
	}
}

func (hs *HistoryService) SaveNmapOsDetectionResponse(result models.NmapOsDetectionResponse) {
	if hs.repo == nil {
		return
	}

	var ip string
	if cachedReq := hs.GetCachedRequest(result.TaskID); cachedReq != nil {
		if osReq, ok := cachedReq.(models.NmapOsDetectionRequest); ok {
			ip = osReq.IP
		}
	}

	historyRecord := &models.NmapOsDetectionHistoryRecord{
		TaskID:   result.TaskID,
		IP:       ip,
		Host:     result.Host,
		Name:     result.Name,
		Accuracy: result.Accuracy,
		Vendor:   result.Vendor,
		Family:   result.Family,
		Type:     result.Type,
		Status:   result.Status,
		Error:    result.Error,
	}

	if err := hs.repo.SaveNmapOsDetectionHistory(historyRecord); err != nil {
		log.Printf("Failed to save Nmap OS Detection history: %v", err)
	} else {
		log.Printf("Successfully saved Nmap OS Detection history for task %s", result.TaskID)
		hs.RemoveCachedRequest(result.TaskID)
	}
}

func (hs *HistoryService) SaveNmapHostDiscoveryResponse(result models.NmapHostDiscoveryResponse) {
	if hs.repo == nil {
		return
	}

	var ip string
	if cachedReq := hs.GetCachedRequest(result.TaskID); cachedReq != nil {
		if hostReq, ok := cachedReq.(models.NmapHostDiscoveryRequest); ok {
			ip = hostReq.IP
		}
	}

	if ip == "" && result.Host != "" {
		ip = result.Host
	}
	host := result.Host
	if host == "" {
		host = ip
	}

	historyRecord := &models.NmapHostDiscoveryHistoryRecord{
		TaskID:    result.TaskID,
		IP:        ip,
		Host:      host,
		HostUP:    result.HostUP,
		HostTotal: result.HostTotal,
		Status:    result.Status,
		DNS:       result.DNS,
		Reason:    result.Reason,
		Error:     result.Error,
	}

	if err := hs.repo.SaveNmapHostDiscoveryHistory(historyRecord); err != nil {
		log.Printf("Failed to save Nmap Host Discovery history: %v", err)
	} else {
		log.Printf("Successfully saved Nmap Host Discovery history for task %s", result.TaskID)
		hs.RemoveCachedRequest(result.TaskID)
	}
}

func (hs *HistoryService) SaveTCPResponse(result models.TCPResponse) {
	if hs.repo == nil {
		return
	}

	var host, port string
	if cachedReq := hs.GetCachedRequest(result.TaskID); cachedReq != nil {
		if tcpReq, ok := cachedReq.(models.TCPRequest); ok {
			host = tcpReq.Host
			port = tcpReq.Port
		}
	}
	if host == "" {
		host = result.Host
	}
	if port == "" {
		port = result.Port
	}

	historyRecord := &models.TCPHistoryRecord{
		TaskID:       result.TaskID,
		Host:         host,
		Port:         port,
		HexObjectKey: result.HexObjectKey,
		DecodedText:  result.DecodedText,
		Status:       result.Status,
		Error:        result.Error,
	}

	if err := hs.repo.SaveTCPHistory(historyRecord); err != nil {
		log.Printf("Failed to save TCP history: %v", err)
	} else {
		log.Printf("Successfully saved TCP history for task %s", result.TaskID)
		hs.RemoveCachedRequest(result.TaskID)
	}

	// Also save to L3 device in new format
	if result.Status == "completed" && host != "" {
		portBanners := map[string]string{}
		if port != "" {
			portBanners[port] = result.DecodedText
		}
		l3Device := &models.L3DeviceNew{
			ID:            host,
			TCPBanners:    portBanners,
			ScannerTypes:  []string{"tcp"},
		}
		if err := hs.repo.SaveOrUpdateL3Device(l3Device); err != nil {
			log.Printf("Failed to save L3 device from TCP response: %v", err)
		}
	}
}

// ProcessTCPToL3Devices processes TCP response and saves to L3 devices in new format
func (hs *HistoryService) ProcessTCPToL3Devices(result models.TCPResponse) {
	if hs.repo == nil {
		return
	}

	var host, port string
	if cachedReq := hs.GetCachedRequest(result.TaskID); cachedReq != nil {
		if tcpReq, ok := cachedReq.(models.TCPRequest); ok {
			host = tcpReq.Host
			port = tcpReq.Port
		}
	}
	if host == "" {
		host = result.Host
	}
	if port == "" {
		port = result.Port
	}

	if result.Status != "completed" || host == "" {
		return
	}

	portBanners := map[string]string{}
	if port != "" {
		portBanners[port] = result.DecodedText
	}

	l3Device := &models.L3DeviceNew{
		ID:           host,
		TCPBanners:   portBanners,
		ScannerTypes: []string{"tcp"},
	}

	if err := hs.repo.SaveOrUpdateL3Device(l3Device); err != nil {
		log.Printf("Failed to save L3 device from TCP response: %v", err)
	} else {
		log.Printf("Successfully saved L3 device from TCP response: %s", host)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// New L2/L3 Device Processing Methods
// ──────────────────────────────────────────────────────────────────────────────

// ProcessARPToL2Devices processes ARP response and saves to L2 devices in new format
func (hs *HistoryService) ProcessARPToL2Devices(result models.ARPResponse) {
	if hs.repo == nil {
		log.Printf("ERROR: Repository is nil in ProcessARPToL2Devices")
		return
	}

	log.Printf("Processing ARP scan: Total devices=%d", len(result.Devices))

	savedCount := 0
	skippedCount := 0

	// Only process online devices with valid MAC and IP addresses to prevent database pollution
	for _, device := range result.Devices {
		// Skip offline devices
		if device.Status != "online" {
			skippedCount++
			continue
		}

		// Skip if MAC is empty or invalid format
		if device.MAC == "" || !isValidMACAddress(device.MAC) {
			skippedCount++
			continue
		}

		// Skip if IP is empty or invalid format
		if device.IP == "" || !isValidIPAddress(device.IP) {
			skippedCount++
			continue
		}

		now := time.Now()
		l2Device := &models.L2DeviceNew{
			ID:           device.MAC,
			Vendor:       device.Vendor,
			ScannerTypes: []string{"arp"},
			IPAddresses: []models.IPAddressInfo{
				{
					IP:        device.IP,
					FirstSeen: now,
					LastSeen:  now,
				},
			},
		}

		if err := hs.repo.SaveOrUpdateL2Device(l2Device); err != nil {
			log.Printf("Failed to save L2 device from ARP response: %v", err)
		} else {
			savedCount++
		}
	}

	log.Printf("ARP scan processed: Saved=%d, Skipped=%d", savedCount, skippedCount)
}

// isValidMACAddress checks if the MAC address has a valid format
func isValidMACAddress(mac string) bool {
	if mac == "" {
		return false
	}
	
	// MAC should contain colons
	if !strings.Contains(mac, ":") && !strings.Contains(mac, "-") {
		return false
	}
	
	// Normalize to colon format for validation
	normalized := strings.ReplaceAll(mac, "-", ":")
	parts := strings.Split(normalized, ":")
	
	// Should have 6 parts
	if len(parts) != 6 {
		return false
	}
	
	return true
}

// isValidIPAddress checks if the IP address has a basic valid format
func isValidIPAddress(ip string) bool {
	if ip == "" {
		return false
	}
	
	// Simple validation: IP should contain dots
	if !strings.Contains(ip, ".") {
		return false
	}
	
	// Basic IPv4 validation
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}
	
	return true
}

// ProcessICMPToL3Devices processes ICMP response and saves to L3 devices in new format
func (hs *HistoryService) ProcessICMPToL3Devices(result models.ICMPResponse) {
	if hs.repo == nil {
		return
	}

	for _, icmpResult := range result.Results {
		// Skip if packets didn't reach (not found)
		if icmpResult.PacketsReceived == 0 {
			continue
		}

		if icmpResult.Address == "" {
			continue
		}

		packetsReached := fmt.Sprintf("%d/%d", icmpResult.PacketsReceived, icmpResult.PacketsSent)
		l3Device := &models.L3DeviceNew{
			ID:             icmpResult.Address,
			PacketsReached:  []string{packetsReached},
			ScannerTypes:    []string{"icmp"},
		}

		if err := hs.repo.SaveOrUpdateL3Device(l3Device); err != nil {
			log.Printf("Failed to save L3 device from ICMP response: %v", err)
		} else {
			log.Printf("Successfully saved L3 device: %s", icmpResult.Address)
		}
	}
}

// ProcessNmapTcpUdpToL3Devices processes Nmap TCP/UDP response and saves to L3 devices in new format
func (hs *HistoryService) ProcessNmapTcpUdpToL3Devices(result models.NmapTcpUdpResponse) {
	if hs.repo == nil {
		return
	}

	if result.Host == "" {
		log.Printf("Skipping L3 device save - empty host")
		return
	}

	var tcpPorts []string
	var udpPorts []string

	log.Printf("Processing Nmap TCP/UDP response for host: %s, PortInfo count: %d", result.Host, len(result.PortInfo))

	for _, portInfo := range result.PortInfo {
		log.Printf("Processing PortInfo: Status=%s, TotalPorts=%d", portInfo.Status, len(portInfo.AllPorts))

		for i, port := range portInfo.AllPorts {
			if i < len(portInfo.Protocols) && i < len(portInfo.State) {
				protocol := portInfo.Protocols[i]
				state := portInfo.State[i]

				log.Printf("Port %d: Protocol=%s, State=%s", port, protocol, state)

				// Check for open ports (including open|filtered for UDP)
				isOpen := state == "open" || state == "open|filtered"

				if protocol == "tcp" && isOpen {
					portStr := fmt.Sprintf("%d", port)
					tcpPorts = append(tcpPorts, portStr)
					log.Printf("Added TCP open port: %s", portStr)
				} else if protocol == "udp" && isOpen {
					portStr := fmt.Sprintf("%d", port)
					udpPorts = append(udpPorts, portStr)
					log.Printf("Added UDP open port: %s", portStr)
				}
			}
		}
	}

	log.Printf("Final TCP ports: %v, UDP ports: %v", tcpPorts, udpPorts)

	l3Device := &models.L3DeviceNew{
		ID:           result.Host,
		TCPOpenPorts: tcpPorts,
		UDPOpenPorts: udpPorts,
		ScannerTypes: []string{"nmap"},
	}

	if err := hs.repo.SaveOrUpdateL3Device(l3Device); err != nil {
		log.Printf("Failed to save L3 device from Nmap TCP/UDP response: %v", err)
	} else {
		log.Printf("Successfully saved L3 device: %s with TCP ports: %v, UDP ports: %v", result.Host, tcpPorts, udpPorts)
	}
}

// ProcessNmapOsDetectionToL3Devices processes Nmap OS detection response and saves to L3 devices in new format
func (hs *HistoryService) ProcessNmapOsDetectionToL3Devices(result models.NmapOsDetectionResponse) {
	if hs.repo == nil {
		return
	}

	if result.Host == "" {
		return
	}

	l3Device := &models.L3DeviceNew{
		ID:          result.Host,
		OS:          result.Name,
		ScannerTypes: []string{"nmap"},
	}

	if err := hs.repo.SaveOrUpdateL3Device(l3Device); err != nil {
		log.Printf("Failed to save L3 device from Nmap OS detection response: %v", err)
	} else {
		log.Printf("Successfully saved L3 device: %s", result.Host)
	}
}

// ProcessNmapHostDiscoveryToL3Devices processes Nmap host discovery response and saves to L3 devices in new format
func (hs *HistoryService) ProcessNmapHostDiscoveryToL3Devices(result models.NmapHostDiscoveryResponse) {
	if hs.repo == nil {
		return
	}

	if result.Host == "" {
		return
	}

	l3Device := &models.L3DeviceNew{
		ID:           result.Host,
		DNS:          result.DNS,
		ScannerTypes: []string{"nmap"},
	}

	if err := hs.repo.SaveOrUpdateL3Device(l3Device); err != nil {
		log.Printf("Failed to save L3 device from Nmap host discovery response: %v", err)
	} else {
		log.Printf("Successfully saved L3 device: %s", result.Host)
	}
}

// LinkARPToL3Devices links MAC addresses from ARP to L3 devices
func (hs *HistoryService) LinkARPToL3Devices(result models.ARPResponse) {
	if hs.repo == nil {
		return
	}

	for _, device := range result.Devices {
		if device.Status != "online" || device.IP == "" || device.MAC == "" {
			continue
		}

		l3Device := &models.L3DeviceNew{
			ID:           device.IP,
			MAC:          device.MAC,
			ScannerTypes: []string{"arp"},
		}

		if err := hs.repo.SaveOrUpdateL3Device(l3Device); err != nil {
			log.Printf("Failed to link MAC to L3 device: %v", err)
		} else {
			log.Printf("Successfully linked MAC %s to L3 device %s", device.MAC, device.IP)
		}
	}
}
