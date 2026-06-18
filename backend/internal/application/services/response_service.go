package services

import (
	"backend/domain/models"
	"log"
)

type ResponseService struct {
	historyService *HistoryService
}

func NewResponseService(historyService *HistoryService) *ResponseService {
	return &ResponseService{
		historyService: historyService,
	}
}

func (rs *ResponseService) ProcessResponse(response *models.Response) {
	if response == nil {
		log.Printf("ProcessResponse: response is nil")
		return
	}

	log.Printf("ProcessResponse: processing response for task %s", response.TaskID)
	log.Printf("ProcessResponse: response result type: %T", response.Result)

	switch result := response.Result.(type) {
	case models.ARPResponse:
		log.Printf("Processing ARP response")
		rs.historyService.SaveARPResponse(result)
		// SaveARPResponse now handles ProcessARPToL2Devices internally
		rs.historyService.LinkARPToL3Devices(result)
	case models.ICMPResponse:
		log.Printf("Processing ICMP response")
		// Only process to new L3 format, don't save separate history records
		rs.historyService.ProcessICMPToL3Devices(result)
		rs.historyService.RemoveCachedRequest(result.TaskID)
	case models.NmapTcpUdpResponse:
		log.Printf("Processing Nmap TCP/UDP response")
		// Only process to new L3 format, don't save separate history records
		rs.historyService.ProcessNmapTcpUdpToL3Devices(result)
		rs.historyService.RemoveCachedRequest(result.TaskID)
	case models.NmapOsDetectionResponse:
		log.Printf("Processing Nmap OS Detection response")
		// Only process to new L3 format, don't save separate history records
		rs.historyService.ProcessNmapOsDetectionToL3Devices(result)
		rs.historyService.RemoveCachedRequest(result.TaskID)
	case models.NmapHostDiscoveryResponse:
		log.Printf("Processing Nmap Host Discovery response")
		// Only process to new L3 format, don't save separate history records
		rs.historyService.ProcessNmapHostDiscoveryToL3Devices(result)
		rs.historyService.RemoveCachedRequest(result.TaskID)
	case models.TCPResponse:
		log.Printf("Processing TCP response")
		// Only process to new L3 format, don't save separate history records
		rs.historyService.ProcessTCPToL3Devices(result)
		rs.historyService.RemoveCachedRequest(result.TaskID)
	default:
		log.Printf("Unknown response type: %T", result)
	}
}
