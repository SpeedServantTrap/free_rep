package domain

type RawRequest struct {
	ScanMethod string `json:"scan_method"`
}
type ScanTcpUdpRequest struct {
	TaskID      string `json:"task_id"`
	IP          string `json:"ip"`
	ScannerType string `json:"scanner_type"`
	Ports       string `json:"ports"`
}

type ScanTcpUdpResponse struct {
	TaskID   string           `json:"task_id"`
	Host     string           `json:"host"`
	PortInfo []PortTcpUdpInfo `json:"port_info"`
	Status   string           `json:"status"`
}

type PortTcpUdpInfo struct {
	Status      string   `json:"status"`
	AllPorts    []uint16 `json:"all_ports"`
	Protocols   []string `json:"protocols"`
	State       []string `json:"state"`
	ServiceName []string `json:"service_name"`
}

type OsDetectionRequest struct {
	TaskID string `json:"task_id"`
	IP     string `json:"ip"`
}

type OsDetectionResponse struct {
	TaskID   string `json:"task_id"`
	Host     string `json:"host"`
	Name     string `json:"name"`
	Accuracy int    `json:"accuracy"`
	Vendor   string `json:"vendor"`
	Family   string `json:"family"`
	Type     string `json:"type"`
	Status   string `json:"status"`
}

type HostDiscoveryRequest struct {
	TaskID string `json:"task_id"`
	IP     string `json:"ip"`
}

type ComprehensiveScanRequest struct {
	TaskID     string `json:"task_id"`
	Input      string `json:"input"`
	ScanMethod string `json:"scan_method"`
}

type ComprehensiveTargetResult struct {
	Host            string           `json:"host"`
	TCPPortInfo     []PortTcpUdpInfo `json:"tcp_port_info,omitempty"`
	UDPPortInfo     []PortTcpUdpInfo `json:"udp_port_info,omitempty"`
	OSName          string           `json:"os_name,omitempty"`
	OSAccuracy      int              `json:"os_accuracy,omitempty"`
	OSVendor        string           `json:"os_vendor,omitempty"`
	OSFamily        string           `json:"os_family,omitempty"`
	OSType          string           `json:"os_type,omitempty"`
	DNS             string           `json:"dns,omitempty"`
	DiscoveryStatus string           `json:"discovery_status,omitempty"`
	DiscoveryReason string           `json:"discovery_reason,omitempty"`
	TCPError        string           `json:"tcp_error,omitempty"`
	UDPError        string           `json:"udp_error,omitempty"`
	OSError         string           `json:"os_error,omitempty"`
	DNSError        string           `json:"dns_error,omitempty"`
}

type ComprehensiveScanResponse struct {
	TaskID  string                    `json:"task_id"`
	Results []ComprehensiveTargetResult `json:"results"`
	Status  string                    `json:"status"`
	Error   string                    `json:"error,omitempty"`
}

type HostDiscoveryResponse struct {
	TaskID    string `json:"task_id"`
	Host      string `json:"host"`
	HostUP    int    `json:"host_up"`
	HostTotal int    `json:"host_total"`
	Status    string `json:"status"`
	DNS       string `json:"dns"`
	Reason    string `json:"reason"`
}
