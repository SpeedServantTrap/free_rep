package application

import (
	"backend/domain/models"
	"backend/internal/application/services"
	"fmt"
	"log"
	rabbitmq "backend/internal/infrastructure/messaging"
	"time"
)

type App struct {
	requestService   *services.RequestService
	responseService  *services.ResponseService
	publisherService *services.PublisherService
	historyService   *services.HistoryService
}

func NewApp(publisher *rabbitmq.RPCScannerPublisher, repo services.RepositoryInterface) *App {
	historyService := services.NewHistoryService(repo)
	requestService := services.NewRequestService()
	responseService := services.NewResponseService(historyService)
	publisherService := services.NewPublisherService(publisher)

	app := &App{
		requestService:   requestService,
		responseService:  responseService,
		publisherService: publisherService,
		historyService:   historyService,
	}

	publisherService.SetResponseCallback(func(response *models.Response) {
		app.ProcessResponse(response)
	})

	return app
}

func (a *App) ProcessRequest(req *models.Request) *models.Response {
	response := a.requestService.ProcessRequest(req)

	if response.TaskID != "error" && response.TaskID != "unknown" {
		switch req.ScannerService {
		case "nmap_service":
			if nmapReq, ok := response.Result.(models.NmapTcpUdpRequest); ok {
				a.historyService.CacheRequest(nmapReq.TaskID, nmapReq)
			} else if nmapReq, ok := response.Result.(models.NmapOsDetectionRequest); ok {
				a.historyService.CacheRequest(nmapReq.TaskID, nmapReq)
			} else if nmapReq, ok := response.Result.(models.NmapHostDiscoveryRequest); ok {
				a.historyService.CacheRequest(nmapReq.TaskID, nmapReq)
			}
			return a.publisherService.PublishNmapRequest(response.Result)

		case "arp_service":
			if arpReq, ok := response.Result.(models.ARPRequest); ok {
				log.Printf("Processing ARP request: TaskID=%s, Command=%s", arpReq.TaskID, arpReq.Command)
				a.historyService.CacheRequest(arpReq.TaskID, arpReq)
				return a.publisherService.PublishARPRequest(arpReq)
			}

		case "icmp_service":
			if icmpReq, ok := response.Result.(models.ICMPRequest); ok {
				a.historyService.CacheRequest(icmpReq.TaskID, icmpReq)
				return a.publisherService.PublishICMPRequest(icmpReq)
			}

		case "tcp_service":
			if tcpReq, ok := response.Result.(models.TCPRequest); ok {
				a.historyService.CacheRequest(tcpReq.TaskID, tcpReq)
				return a.publisherService.PublishTCPRequest(tcpReq)
			}
		}
	}

	return response
}

func (a *App) ProcessResponse(response *models.Response) {
	if response == nil {
		a.responseService.ProcessResponse(response)
		return
	}

	nmapResp, ok := response.Result.(models.NmapTcpUdpResponse)
	if !ok {
		a.responseService.ProcessResponse(response)
		return
	}

	a.responseService.ProcessResponse(response)

	go a.triggerTCPBannerScansFromNmap(nmapResp.Host, nmapResp.TaskID, extractOpenTCPPorts(nmapResp))
}

func (a *App) triggerTCPBannerScansFromNmap(host, sourceTaskID string, ports []string) {
	if host == "" {
		log.Printf("[Auto-TCP] Skip: empty host for Nmap task %s", sourceTaskID)
		return
	}

	if len(ports) == 0 {
		log.Printf("[Auto-TCP] No open TCP ports for host %s (task %s)", host, sourceTaskID)
		return
	}

	log.Printf("[Auto-TCP] Triggering TCP banner scans for host %s, ports=%v (source task=%s)", host, ports, sourceTaskID)

	for _, port := range ports {
		taskID := fmt.Sprintf("%s-tcp-banner-%s-%d", sourceTaskID, port, time.Now().UnixNano())
		tcpReq := models.TCPRequest{
			TaskID: taskID,
			Host:   host,
			Port:   port,
		}

		a.historyService.CacheRequest(taskID, tcpReq)
		resp := a.publisherService.PublishTCPRequest(tcpReq)
		if resp == nil {
			log.Printf("[Auto-TCP] Empty response for %s:%s (task %s)", host, port, taskID)
			continue
		}

		if errMap, ok := resp.Result.(map[string]string); ok {
			if errMsg, hasErr := errMap["error"]; hasErr && errMsg != "" {
				log.Printf("[Auto-TCP] Failed %s:%s (task %s): %s", host, port, taskID, errMsg)
				continue
			}
		}

		log.Printf("[Auto-TCP] Completed %s:%s (task %s)", host, port, taskID)
	}
}
func extractOpenTCPPorts(result models.NmapTcpUdpResponse) []string {
	seen := map[string]struct{}{}
	ports := make([]string, 0)

	for _, info := range result.PortInfo {
		for i, p := range info.AllPorts {
			protocol := ""
			state := ""

			if i < len(info.Protocols) {
				protocol = info.Protocols[i]
			}
			if i < len(info.State) {
				state = info.State[i]
			}

			if protocol != "tcp" {
				continue
			}
			if state != "open" && state != "open|filtered" {
				continue
			}

			port := fmt.Sprintf("%d", p)
			if _, exists := seen[port]; exists {
				continue
			}
			seen[port] = struct{}{}
			ports = append(ports, port)
		}
	}

	return ports
}

func (a *App) PublishNmapRequest(req interface{}) *models.Response {
	return a.publisherService.PublishNmapRequest(req)
}

func (a *App) GetICMPHistory(limit int) ([]models.ICMPHistoryRecord, error) {
	return a.historyService.GetRepo().GetICMPHistory(limit)
}

func (a *App) GetNmapTcpUdpHistory(limit int) ([]models.NmapTcpUdpHistoryRecord, error) {
	return a.historyService.GetRepo().GetNmapTcpUdpHistory(limit)
}

func (a *App) GetNmapOsDetectionHistory(limit int) ([]models.NmapOsDetectionHistoryRecord, error) {
	return a.historyService.GetRepo().GetNmapOsDetectionHistory(limit)
}

func (a *App) GetNmapHostDiscoveryHistory(limit int) ([]models.NmapHostDiscoveryHistoryRecord, error) {
	return a.historyService.GetRepo().GetNmapHostDiscoveryHistory(limit)
}

func (a *App) DeleteICMPHistory() error {
	return a.historyService.GetRepo().DeleteICMPHistory()
}

func (a *App) DeleteNmapTcpUdpHistory() error {
	return a.historyService.GetRepo().DeleteNmapTcpUdpHistory()
}

func (a *App) DeleteNmapOsDetectionHistory() error {
	return a.historyService.GetRepo().DeleteNmapOsDetectionHistory()
}

func (a *App) DeleteNmapHostDiscoveryHistory() error {
	return a.historyService.GetRepo().DeleteNmapHostDiscoveryHistory()
}

// ──────────────────────────────────────────────────────────────────────────────
// New L2/L3 Device Management Methods
// ──────────────────────────────────────────────────────────────────────────────

func (a *App) GetAllL2Devices() ([]models.L2DeviceNew, error) {
	return a.historyService.GetRepo().GetAllL2Devices()
}

func (a *App) GetL2Device(mac string) (*models.L2DeviceNew, error) {
	return a.historyService.GetRepo().GetL2Device(mac)
}

func (a *App) DeleteAllL2Devices() error {
	return a.historyService.GetRepo().DeleteAllL2Devices()
}

func (a *App) DropL2Collection() error {
	return a.historyService.GetRepo().DropL2Collection()
}

func (a *App) GetAllL3Devices() ([]models.L3DeviceNew, error) {
	return a.historyService.GetRepo().GetAllL3Devices()
}

func (a *App) GetL3Device(ip string) (*models.L3DeviceNew, error) {
	return a.historyService.GetRepo().GetL3Device(ip)
}

func (a *App) DeleteAllL3Devices() error {
	return a.historyService.GetRepo().DeleteAllL3Devices()
}

func (a *App) DropL3Collection() error {
	return a.historyService.GetRepo().DropL3Collection()
}

// MigrateToNewFormat drops old collections and initializes new format
func (a *App) MigrateToNewFormat() error {
	log.Printf("Starting migration to new L2/L3 format...")

	// Drop old collections
	if err := a.DropL2Collection(); err != nil {
		log.Printf("Error dropping L2 collection: %v", err)
		return err
	}

	if err := a.DropL3Collection(); err != nil {
		log.Printf("Error dropping L3 collection: %v", err)
		return err
	}

	log.Printf("Migration to new format completed successfully")
	return nil
}
