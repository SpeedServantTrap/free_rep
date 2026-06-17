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
		// Also process to new L2/L3 format
		rs.historyService.ProcessARPToL2Devices(result)
		rs.historyService.LinkARPToL3Devices(result)
	case models.ICMPResponse:
		log.Printf("Processing ICMP response")
		rs.historyService.SaveICMPResponse(result)
		// Also process to new L3 format
		rs.historyService.ProcessICMPToL3Devices(result)
	case models.NmapTcpUdpResponse:
		log.Printf("Processing Nmap TCP/UDP response")
		rs.historyService.SaveNmapTcpUdpResponse(result)
		// Also process to new L3 format
		rs.historyService.ProcessNmapTcpUdpToL3Devices(result)
	case models.NmapOsDetectionResponse:
		log.Printf("Processing Nmap OS Detection response")
		rs.historyService.SaveNmapOsDetectionResponse(result)
		// Also process to new L3 format
		rs.historyService.ProcessNmapOsDetectionToL3Devices(result)
	case models.NmapHostDiscoveryResponse:
		log.Printf("Processing Nmap Host Discovery response")
		rs.historyService.SaveNmapHostDiscoveryResponse(result)
		// Also process to new L3 format
		rs.historyService.ProcessNmapHostDiscoveryToL3Devices(result)
	case models.TCPResponse:
		log.Printf("Processing TCP response")
		rs.historyService.SaveTCPResponse(result)
		// Also process to new L3 format (handled in SaveTCPResponse)
	default:
		log.Printf("Unknown response type: %T", result)
	}
}
