package services

import (
	"backend/domain/models"
	"fmt"
	"log"
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
	GetARPHistoryByIPRange(ipRange string, limit int) ([]models.ARPHistoryRecord, error)
	GetARPHistoryByID(id string) (*models.ARPHistoryRecord, error)
	GetTCPHistoryByHostPort(host, port string, limit int) ([]models.TCPHistoryRecord, error)
	GetTCPHistoryByID(id string) (*models.TCPHistoryRecord, error)
}

type RepositoryInterface interface {
	SaveARPHistory(record *models.ARPHistoryRecord) error
	GetARPHistory(limit int) ([]models.ARPHistoryRecord, error)
	DeleteARPHistory() error

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
	if hs.repo == nil {
		return
	}

	var interfaceName, ipRange string
	if cachedReq := hs.GetCachedRequest(result.TaskID); cachedReq != nil {
		if arpReq, ok := cachedReq.(models.ARPRequest); ok {
			interfaceName = arpReq.InterfaceName
			ipRange = arpReq.IPRange
		}
	}

	var onlineDevices, offlineDevices []models.ARPDevice
	for _, device := range result.Devices {
		if device.Status == "online" {
			onlineDevices = append(onlineDevices, device)
		} else {
			offlineDevices = append(offlineDevices, device)
		}
	}

	historyRecord := &models.ARPHistoryRecord{
		TaskID:         result.TaskID,
		InterfaceName:  interfaceName,
		IPRange:        ipRange,
		Status:         result.Status,
		Devices:        result.Devices,
		OnlineDevices:  onlineDevices,
		OfflineDevices: offlineDevices,
		TotalCount:     result.TotalCount,
		OnlineCount:    result.OnlineCount,
		OfflineCount:   result.OfflineCount,
		Error:          result.Error,
	}

	if err := hs.repo.SaveARPHistory(historyRecord); err != nil {
		log.Printf("Failed to save ARP history: %v", err)
	} else {
		log.Printf("Successfully saved ARP history for task %s", result.TaskID)
		hs.RemoveCachedRequest(result.TaskID)
	}
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
	if result.Status == "success" && host != "" {
		scanTime := time.Now().Format(time.RFC3339)
		l3Device := &models.L3DeviceNew{
			ID:            host,
			TCPBanner:     result.DecodedText,
			ScanTimes:     []string{scanTime},
			ScannerTypes:  []string{"tcp"},
		}
		if err := hs.repo.SaveOrUpdateL3Device(l3Device); err != nil {
			log.Printf("Failed to save L3 device from TCP response: %v", err)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// New L2/L3 Device Processing Methods
// ──────────────────────────────────────────────────────────────────────────────

// ProcessARPToL2Devices processes ARP response and saves to L2 devices in new format
func (hs *HistoryService) ProcessARPToL2Devices(result models.ARPResponse) {
	if hs.repo == nil {
		return
	}

	scanTime := time.Now().Format(time.RFC3339)

	// Only process online devices (filter out offline as per requirements)
	for _, device := range result.Devices {
		if device.Status != "online" {
			continue
		}

		// Skip if MAC is empty (not found)
		if device.MAC == "" {
			continue
		}

		l2Device := &models.L2DeviceNew{
			ID:           device.MAC,
			Vendor:       device.Vendor,
			ScanTimes:    []string{scanTime},
			ScannerTypes: []string{"arp"},
		}

		if device.IP != "" {
			l2Device.IPAddresses = []string{device.IP}
		}

		if err := hs.repo.SaveOrUpdateL2Device(l2Device); err != nil {
			log.Printf("Failed to save L2 device from ARP response: %v", err)
		} else {
			log.Printf("Successfully saved L2 device: %s", device.MAC)
		}
	}
}

// ProcessICMPToL3Devices processes ICMP response and saves to L3 devices in new format
func (hs *HistoryService) ProcessICMPToL3Devices(result models.ICMPResponse) {
	if hs.repo == nil {
		return
	}

	scanTime := time.Now().Format(time.RFC3339)

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
			ScanTimes:       []string{scanTime},
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

	scanTime := time.Now().Format(time.RFC3339)

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
		ScanTimes:    []string{scanTime},
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

	scanTime := time.Now().Format(time.RFC3339)

	if result.Host == "" {
		return
	}

	l3Device := &models.L3DeviceNew{
		ID:          result.Host,
		OS:          result.Name,
		ScanTimes:   []string{scanTime},
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

	scanTime := time.Now().Format(time.RFC3339)

	if result.Host == "" {
		return
	}

	l3Device := &models.L3DeviceNew{
		ID:           result.Host,
		DNS:          result.DNS,
		ScanTimes:    []string{scanTime},
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
			ID:  device.IP,
			MAC: device.MAC,
		}

		if err := hs.repo.SaveOrUpdateL3Device(l3Device); err != nil {
			log.Printf("Failed to link MAC to L3 device: %v", err)
		} else {
			log.Printf("Successfully linked MAC %s to L3 device %s", device.MAC, device.IP)
		}
	}
}
