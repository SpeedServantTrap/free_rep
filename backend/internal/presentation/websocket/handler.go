package websocket

import (
	"encoding/json"
	"log"
	"net/http"

	"backend/domain/models"
	api "backend/internal/application"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type WSHandler struct {
	app *api.App
}

func NewWSHandler(app *api.App) *WSHandler {
	return &WSHandler{
		app: app,
	}
}

type Message struct {
	Type           string               `json:"type"`
	Req            *models.Request      `json:"request,omitempty"`
	Resp           *models.Response     `json:"response,omitempty"`
	Change         *models.ChangeEvent  `json:"change,omitempty"`
	ScannerService string               `json:"scanner_service,omitempty"`
}

type Client struct {
	conn *websocket.Conn
	send chan Message
	app  *api.App
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *WSHandler) WsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	client := &Client{
		conn: conn,
		send: make(chan Message, 256),
		app:  h.app,
	}

	globalHub.Register(client)
	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		globalHub.Unregister(c)
		c.conn.Close()
	}()

	for {
		var msg Message

		if err := c.conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		scannerService := ""
		if msg.Req != nil {
			scannerService = msg.Req.ScannerService
		}
		log.Printf("Received message type=%s, scanner_service=%s", msg.Type, scannerService)

		if msg.Req != nil {
			response := c.processRequest(msg.Req)

			c.send <- Message{
				Type:           "response",
				Resp:           response,
				ScannerService: msg.Req.ScannerService,
			}
		}
	}
}

func (c *Client) processRequest(req *models.Request) *models.Response {
	taskID := generateTaskID()
	log.Printf("Processing request for scanner_service: %s", req.ScannerService)
	log.Printf("Request options: %+v", req.Options)

	switch req.ScannerService {
	case "arp_service":
		return c.processARPRequest(req.Options, taskID)
	case "icmp_service", "ping_service":
		return c.processICMPRequest(req.Options, taskID)
	case "nmap_service":
		log.Printf("Calling processNmapRequest")
		return c.processNmapRequest(req.Options, taskID)
	case "tcp_service":
		return c.processTCPRequest(req.Options, taskID)
	default:
		log.Printf("Unsupported scanner_service: %s", req.ScannerService)
		return &models.Response{
			TaskID: taskID,
			Result: map[string]string{
				"error": "unsupported scanner_service: " + req.ScannerService,
			},
		}
	}
}

func (c *Client) processARPRequest(options any, taskID string) *models.Response {
	var arpOpts struct {
		InterfaceName string `json:"interface_name"`
		IPRange       string `json:"ip_range"`
		Command       string `json:"command,omitempty"`
	}

	if err := parseOptions(options, &arpOpts); err != nil {
		log.Printf("Failed to parse ARP options: %v", err)
		return &models.Response{
			TaskID: taskID,
			Result: map[string]string{"error": "invalid ARP options: " + err.Error()},
		}
	}

	// Handle auto-scan control commands
	if arpOpts.Command == "start" || arpOpts.Command == "stop" {
		log.Printf("ARP auto scan command: %s", arpOpts.Command)
		arpRequest := models.ARPRequest{
			TaskID:  taskID,
			Command: arpOpts.Command,
		}
		return c.app.ProcessRequest(&models.Request{
			ScannerService: "arp_service",
			Options:        arpRequest,
		})
	}

	// Regular ARP scan request
	if arpOpts.InterfaceName == "" {
		return &models.Response{
			TaskID: taskID,
			Result: map[string]string{"error": "interface_name is required for ARP scan"},
		}
	}
	if arpOpts.IPRange == "" {
		return &models.Response{
			TaskID: taskID,
			Result: map[string]string{"error": "ip_range is required for ARP scan"},
		}
	}

	arpRequest := models.ARPRequest{
		TaskID:        taskID,
		InterfaceName: arpOpts.InterfaceName,
		IPRange:       arpOpts.IPRange,
	}

	return c.app.ProcessRequest(&models.Request{
		ScannerService: "arp_service",
		Options:        arpRequest,
	})
}

func (c *Client) processICMPRequest(options any, taskID string) *models.Response {
	var icmpOpts struct {
		Targets   []string `json:"targets"`
		PingCount int      `json:"ping_count"`
		Command   string   `json:"command,omitempty"`
	}

	if err := parseOptions(options, &icmpOpts); err != nil {
		return &models.Response{
			TaskID: taskID,
			Result: map[string]string{"error": "invalid ICMP options: " + err.Error()},
		}
	}

	// Handle auto-scan control commands
	if icmpOpts.Command == "start" || icmpOpts.Command == "stop" {
		log.Printf("ICMP auto scan command: %s", icmpOpts.Command)
		icmpRequest := models.ICMPRequest{
			TaskID:  taskID,
			Command: icmpOpts.Command,
		}
		return c.app.ProcessRequest(&models.Request{
			ScannerService: "icmp_service",
			Options:        icmpRequest,
		})
	}

	// Regular ICMP scan request
	if len(icmpOpts.Targets) == 0 {
		return &models.Response{
			TaskID: taskID,
			Result: map[string]string{"error": "targets are required for ICMP ping"},
		}
	}
	if icmpOpts.PingCount <= 0 {
		icmpOpts.PingCount = 4
	}

	icmpRequest := models.ICMPRequest{
		TaskID:    taskID,
		Targets:   icmpOpts.Targets,
		PingCount: icmpOpts.PingCount,
	}

	return c.app.ProcessRequest(&models.Request{
		ScannerService: "icmp_service",
		Options:        icmpRequest,
	})
}

func (c *Client) processNmapRequest(options any, taskID string) *models.Response {
	log.Printf("Processing Nmap request with options: %+v", options)

	var nmapOpts struct {
		ScanMethod  string `json:"scan_method"`
		Command     string `json:"command,omitempty"`
		IP          string `json:"ip"`
		Input       string `json:"input"`
		Ports       string `json:"ports"`
		ScannerType string `json:"scanner_type"`
	}

	if err := parseOptions(options, &nmapOpts); err != nil {
		log.Printf("Failed to parse Nmap options: %v", err)
		return &models.Response{
			TaskID: taskID,
			Result: map[string]string{"error": "invalid Nmap options: " + err.Error()},
		}
	}

	log.Printf("Parsed Nmap options: ScanMethod=%s, IP=%s, Ports=%s, ScannerType=%s",
		nmapOpts.ScanMethod, nmapOpts.IP, nmapOpts.Ports, nmapOpts.ScannerType)

	if nmapOpts.Command == "start" || nmapOpts.Command == "stop" {
		log.Printf("Nmap auto scan command: %s", nmapOpts.Command)
		nmapRequest := models.NmapComprehensiveRequest{
			TaskID:  taskID,
			Command: nmapOpts.Command,
		}

		return c.app.ProcessRequest(&models.Request{
			ScannerService: "nmap_service",
			Options:        nmapRequest,
		})
	}

	switch nmapOpts.ScanMethod {
	case "comprehensive_scan":
		if nmapOpts.Input == "" {
			return &models.Response{
				TaskID: taskID,
				Result: map[string]string{"error": "input is required for comprehensive scan"},
			}
		}

		nmapRequest := models.NmapComprehensiveRequest{
			TaskID:     taskID,
			Input:      nmapOpts.Input,
			ScanMethod: "comprehensive_scan",
			Command:    nmapOpts.Command,
		}

		log.Printf("Launching async NmapComprehensiveRequest: task=%s input=%s", taskID, nmapOpts.Input)

		// Fire the scan in a goroutine so readPump is not blocked.
		// When the scan finishes, ProcessResponse broadcasts via Hub.
		go func() {
			resp := c.app.ProcessRequest(&models.Request{
				ScannerService: "nmap_service",
				Options:        nmapRequest,
			})
			// On error (timeout / network failure) notify only this client.
			if resp != nil {
				if errRes, ok := resp.Result.(map[string]string); ok && errRes["error"] != "" {
					select {
					case c.send <- Message{Type: "response", Resp: resp, ScannerService: "nmap_service"}:
					default:
					}
				}
			}
		}()

		// Return "started" immediately so the UI shows scanning state.
		return &models.Response{
			TaskID: taskID,
			Result: map[string]string{"status": "started"},
		}

	case "tcp_udp_scan":
		if nmapOpts.IP == "" {
			return &models.Response{
				TaskID: taskID,
				Result: map[string]string{"error": "IP is required for TCP/UDP scan"},
			}
		}

		nmapRequest := models.NmapTcpUdpRequest{
			TaskID:      taskID,
			IP:          nmapOpts.IP,
			ScannerType: nmapOpts.ScannerType,
			Ports:       nmapOpts.Ports,
			ScanMethod:  "tcp_udp_scan",
		}

		log.Printf("Created NmapTcpUdpRequest: %+v", nmapRequest)
		return c.app.ProcessRequest(&models.Request{
			ScannerService: "nmap_service",
			Options:        nmapRequest,
		})

	case "os_detection":
		if nmapOpts.IP == "" {
			return &models.Response{
				TaskID: taskID,
				Result: map[string]string{"error": "IP is required for OS detection"},
			}
		}

		nmapRequest := models.NmapOsDetectionRequest{
			TaskID:     taskID,
			IP:         nmapOpts.IP,
			ScanMethod: "os_detection",
		}

		log.Printf("Created NmapOsDetectionRequest: %+v", nmapRequest)
		return c.app.ProcessRequest(&models.Request{
			ScannerService: "nmap_service",
			Options:        nmapRequest,
		})

	case "host_discovery":
		if nmapOpts.IP == "" {
			return &models.Response{
				TaskID: taskID,
				Result: map[string]string{"error": "IP is required for host discovery"},
			}
		}

		nmapRequest := models.NmapHostDiscoveryRequest{
			TaskID:     taskID,
			IP:         nmapOpts.IP,
			ScanMethod: "host_discovery",
		}

		log.Printf("Created NmapHostDiscoveryRequest: %+v", nmapRequest)
		return c.app.ProcessRequest(&models.Request{
			ScannerService: "nmap_service",
			Options:        nmapRequest,
		})

	default:
		return &models.Response{
			TaskID: taskID,
			Result: map[string]string{"error": "unsupported nmap scan method: " + nmapOpts.ScanMethod},
		}
	}
}

func (c *Client) processTCPRequest(options any, taskID string) *models.Response {
	var tcpOpts struct {
		Host string `json:"host"`
		Port string `json:"port"`
	}

	if err := parseOptions(options, &tcpOpts); err != nil {
		return &models.Response{
			TaskID: taskID,
			Result: map[string]string{"error": "invalid TCP options: " + err.Error()},
		}
	}

	if tcpOpts.Host == "" {
		return &models.Response{
			TaskID: taskID,
			Result: map[string]string{"error": "host is required for TCP scan"},
		}
	}
	if tcpOpts.Port == "" {
		return &models.Response{
			TaskID: taskID,
			Result: map[string]string{"error": "port is required for TCP scan"},
		}
	}

	tcpRequest := models.TCPRequest{
		TaskID: taskID,
		Host:   tcpOpts.Host,
		Port:   tcpOpts.Port,
	}

	return c.app.ProcessRequest(&models.Request{
		ScannerService: "tcp_service",
		Options:        tcpRequest,
	})
}

func generateTaskID() string {
	return uuid.New().String()
}

func parseOptions(options any, target interface{}) error {
	optionsJSON, err := json.Marshal(options)
	if err != nil {
		return err
	}
	return json.Unmarshal(optionsJSON, target)
}

func (c *Client) writePump() {
	defer c.conn.Close()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
			log.Printf("Sent response: %+v", message.Resp)
		}
	}
}
