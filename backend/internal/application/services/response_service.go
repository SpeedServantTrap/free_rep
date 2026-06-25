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
	case models.NmapComprehensiveResponse:
		for _, target := range result.Results {
			if target.Host == "" {
				continue
			}

			if len(target.TCPPortInfo) > 0 {
				rs.historyService.ProcessNmapTcpUdpToL3Devices(models.NmapTcpUdpResponse{
					TaskID:   result.TaskID,
					Host:     target.Host,
					PortInfo: target.TCPPortInfo,
					Status:   result.Status,
					Error:    target.TCPError,
				})
			}

			if len(target.UDPPortInfo) > 0 {
				rs.historyService.ProcessNmapTcpUdpToL3Devices(models.NmapTcpUdpResponse{
					TaskID:   result.TaskID,
					Host:     target.Host,
					PortInfo: target.UDPPortInfo,
					Status:   result.Status,
					Error:    target.UDPError,
				})
			}

			if target.OSName != "" || target.OSVendor != "" || target.OSFamily != "" || target.OSType != "" || target.OSError != "" {
				rs.historyService.ProcessNmapOsDetectionToL3Devices(models.NmapOsDetectionResponse{
					TaskID:   result.TaskID,
					Host:     target.Host,
					Name:     target.OSName,
					Accuracy: target.OSAccuracy,
					Vendor:   target.OSVendor,
					Family:   target.OSFamily,
					Type:     target.OSType,
					Status:   result.Status,
					Error:    target.OSError,
				})
			}

			if target.DNS != "" || target.DiscoveryStatus != "" || target.DiscoveryReason != "" || target.DNSError != "" {
				rs.historyService.ProcessNmapHostDiscoveryToL3Devices(models.NmapHostDiscoveryResponse{
					TaskID: result.TaskID,
					Host:   target.Host,
					Status: target.DiscoveryStatus,
					DNS:    target.DNS,
					Reason: target.DiscoveryReason,
					Error:  target.DNSError,
				})
			}
		}
		if result.Status == "completed" || result.Status == "failed" {
			rs.historyService.RemoveCachedRequest(result.TaskID)
		}
	case models.TCPResponse:
		rs.historyService.ProcessTCPToL3Devices(result)
		rs.historyService.RemoveCachedRequest(result.TaskID)
	default:
		log.Printf("Unknown response type: %T", result)
	}
}
