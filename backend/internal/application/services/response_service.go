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

	switch result := response.Result.(type) {
	case models.ARPResponse:
		rs.historyService.SaveARPResponse(result)
	case models.ICMPResponse:
		rs.historyService.ProcessICMPToL3Devices(result)
		rs.historyService.RemoveCachedRequest(result.TaskID)
	case models.NmapTcpUdpResponse:
		rs.historyService.ProcessNmapTcpUdpToL3Devices(result)
		rs.historyService.RemoveCachedRequest(result.TaskID)
	case models.NmapOsDetectionResponse:
		rs.historyService.ProcessNmapOsDetectionToL3Devices(result)
		rs.historyService.RemoveCachedRequest(result.TaskID)
	case models.NmapHostDiscoveryResponse:
		rs.historyService.ProcessNmapHostDiscoveryToL3Devices(result)
		rs.historyService.RemoveCachedRequest(result.TaskID)
	case models.TCPResponse:
		rs.historyService.ProcessTCPToL3Devices(result)
		rs.historyService.RemoveCachedRequest(result.TaskID)
	default:
		log.Printf("Unknown response type: %T", result)
	}
}
