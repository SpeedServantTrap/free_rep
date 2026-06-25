package usecases

import (
	"context"
	"fmt"
	"net"
	"scanner_nmap/internal/domain"
	"scanner_nmap/internal/usecases/nmap_wrapper"
	"strings"
	"sync"

	"github.com/Ullaakut/nmap/v3"
)

func UdpTcpScanner(ctx context.Context, request domain.ScanTcpUdpRequest) (response domain.ScanTcpUdpResponse, err error) {
	var scanResults []*nmap.Run

	scanType := "TCP"
	if request.ScannerType == "UDP" || request.ScannerType == "udp_scan" {
		scanType = "UDP"
	}

	fmt.Printf("Starting %s scan for %s on ports %s\n", scanType, request.IP, request.Ports)

	if request.ScannerType == "UDP" || request.ScannerType == "udp_scan" {
		scanResult, scanErr := nmap_wrapper.UDPScan(ctx, request.IP, request.Ports)
		if scanErr != nil {
			err = scanErr
		} else if scanResult != nil {
			scanResults = append(scanResults, scanResult)
		}
	} else {
		scanResult, scanErr := nmap_wrapper.TCPScan(ctx, request.IP, request.Ports)
		if scanErr != nil {
			err = scanErr
		} else if scanResult != nil {
			scanResults = append(scanResults, scanResult)
		}
	}

	if err != nil {
		fmt.Printf("%s scan error: %v\n", scanType, err)
		return domain.ScanTcpUdpResponse{}, err
	}

	if len(scanResults) == 0 {
		fmt.Printf("%s scanner doesn't have any results\n", scanType)
		return domain.ScanTcpUdpResponse{}, fmt.Errorf("no scan results")
	}

	// Detailed logging of scan results
	for runIdx, scanResult := range scanResults {
		fmt.Printf("Scan run %d stats: %d hosts found\n", runIdx+1, len(scanResult.Hosts))
		for i, host := range scanResult.Hosts {
			fmt.Printf("Host %d: Status=%s, %d addresses, %d ports\n",
				i, host.Status.State, len(host.Addresses), len(host.Ports))
		}
	}

	var hostResult string
	for _, run := range scanResults {
		for _, host := range run.Hosts {
			if len(host.Addresses) > 0 {
				hostResult = host.Addresses[0].String()
				break
			}
		}
		if hostResult != "" {
			break
		}
	}

	portInfo := domain.PortTcpUdpInfo{}
	totalPorts := 0
	openPorts := 0
	seenOpen := map[string]struct{}{}

	for _, run := range scanResults {
		for _, host := range run.Hosts {
			if len(host.Addresses) == 0 {
				continue
			}

			portInfo.Status = host.Status.State

			for _, port := range host.Ports {
				totalPorts++

				// Only add ports that are actually open
				// Check for both "open" and "open|filtered" states
				isOpen := port.State.State == "open" || port.State.State == "open|filtered"

				if !isOpen {
					continue
				}

				key := fmt.Sprintf("%s/%d", port.Protocol, port.ID)
				if _, exists := seenOpen[key]; exists {
					continue
				}
				seenOpen[key] = struct{}{}

				openPorts++
				portInfo.AllPorts = append(portInfo.AllPorts, port.ID)
				portInfo.Protocols = append(portInfo.Protocols, port.Protocol)
				portInfo.State = append(portInfo.State, port.State.State)

				serviceName := port.Service.Name
				if serviceName == "" {
					serviceName = "unknown"
				}
				portInfo.ServiceName = append(portInfo.ServiceName, serviceName)
			}
		}
	}

	fmt.Printf("%s scan completed: Total ports in results: %d, Open ports found: %d\n", scanType, totalPorts, openPorts)

	responseResult := domain.ScanTcpUdpResponse{
		TaskID:   request.TaskID,
		Host:     hostResult,
		PortInfo: []domain.PortTcpUdpInfo{portInfo},
	}

	return responseResult, err
}

func ComprehensiveScanner(ctx context.Context, request domain.ComprehensiveScanRequest) (domain.ComprehensiveScanResponse, error) {
	return ComprehensiveScannerStream(ctx, request, nil)
}

func ComprehensiveScannerStream(
	ctx context.Context,
	request domain.ComprehensiveScanRequest,
	onTargetDone func(domain.ComprehensiveTargetResult),
) (domain.ComprehensiveScanResponse, error) {
	targets, err := parseTargetsInput(request.Input)
	if err != nil {
		return domain.ComprehensiveScanResponse{}, err
	}
	if len(targets) == 0 {
		return domain.ComprehensiveScanResponse{}, fmt.Errorf("no valid targets to scan")
	}

	results := make([]domain.ComprehensiveTargetResult, len(targets))
	type indexedResult struct {
		idx    int
		result domain.ComprehensiveTargetResult
	}
	resultsCh := make(chan indexedResult, len(targets))

	for i, target := range targets {
		host := target
		idx := i
		go func(idx int, host string) {
			r := runComprehensiveTarget(ctx, request.TaskID, host)
			resultsCh <- indexedResult{idx: idx, result: r}
		}(idx, host)
	}

	for done := 0; done < len(targets); done++ {
		item := <-resultsCh
		results[item.idx] = item.result
		if onTargetDone != nil {
			onTargetDone(item.result)
		}
	}
	close(resultsCh)

	return domain.ComprehensiveScanResponse{
		TaskID:  request.TaskID,
		Results: results,
		Status:  "completed",
	}, nil
}

func runComprehensiveTarget(ctx context.Context, taskID, host string) domain.ComprehensiveTargetResult {
	result := domain.ComprehensiveTargetResult{Host: host}
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(4)

	go func() {
		defer wg.Done()
		tcpResp, err := UdpTcpScanner(ctx, domain.ScanTcpUdpRequest{
			TaskID:      taskID,
			IP:          host,
			ScannerType: "TCP",
			Ports:       "1-65535",
		})
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			result.TCPError = err.Error()
			return
		}
		result.Host = firstNonEmpty(tcpResp.Host, result.Host)
		result.TCPPortInfo = tcpResp.PortInfo
	}()

	go func() {
		defer wg.Done()
		udpResp, err := UdpTcpScanner(ctx, domain.ScanTcpUdpRequest{
			TaskID:      taskID,
			IP:          host,
			ScannerType: "UDP",
			Ports:       "1-65535",
		})
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			result.UDPError = err.Error()
			return
		}
		result.Host = firstNonEmpty(udpResp.Host, result.Host)
		result.UDPPortInfo = udpResp.PortInfo
	}()

	go func() {
		defer wg.Done()
		osResp, err := OSDetectionScanner(ctx, domain.OsDetectionRequest{TaskID: taskID, IP: host})
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			result.OSError = err.Error()
			return
		}
		result.Host = firstNonEmpty(osResp.Host, result.Host)
		result.OSName = osResp.Name
		result.OSAccuracy = osResp.Accuracy
		result.OSVendor = osResp.Vendor
		result.OSFamily = osResp.Family
		result.OSType = osResp.Type
	}()

	go func() {
		defer wg.Done()
		dnsResp, err := HostDiscoveryScanner(ctx, domain.HostDiscoveryRequest{TaskID: taskID, IP: host})
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			result.DNSError = err.Error()
			return
		}
		result.Host = firstNonEmpty(dnsResp.Host, result.Host)
		result.DNS = dnsResp.DNS
		result.DiscoveryStatus = dnsResp.Status
		result.DiscoveryReason = dnsResp.Reason
	}()

	wg.Wait()
	return result
}

func parseTargetsInput(input string) ([]string, error) {
	parts := strings.Split(input, ",")
	targets := make([]string, 0)
	seen := make(map[string]struct{})

	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}

		var expanded []string
		if strings.Contains(token, "/") {
			cidrTargets, err := expandCIDR(token)
			if err != nil {
				return nil, err
			}
			expanded = cidrTargets
		} else {
			expanded = []string{token}
		}

		for _, target := range expanded {
			if net.ParseIP(target) == nil {
				return nil, fmt.Errorf("invalid target: %s", target)
			}
			if _, exists := seen[target]; exists {
				continue
			}
			seen[target] = struct{}{}
			targets = append(targets, target)
		}
	}

	return targets, nil
}

func expandCIDR(cidr string) ([]string, error) {
	ip, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR %s: %w", cidr, err)
	}
	ip = ip.To4()
	if ip == nil {
		return nil, fmt.Errorf("only IPv4 CIDR is supported: %s", cidr)
	}
	maskSize, bits := network.Mask.Size()
	if bits != 32 {
		return nil, fmt.Errorf("only IPv4 CIDR is supported: %s", cidr)
	}

	start := ipv4ToUint32(network.IP.Mask(network.Mask))
	hostCount := uint32(1) << uint32(32-maskSize)
	end := start + hostCount - 1

	first := start
	last := end
	if hostCount > 2 {
		first = start + 1
		last = end - 1
	}

	results := make([]string, 0, last-first+1)
	for current := first; current <= last; current++ {
		results = append(results, uint32ToIPv4(current).String())
	}
	return results, nil
}

func ipv4ToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func uint32ToIPv4(value uint32) net.IP {
	return net.IPv4(byte(value>>24), byte(value>>16), byte(value>>8), byte(value))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func OSDetectionScanner(ctx context.Context, request domain.OsDetectionRequest) (response domain.OsDetectionResponse, err error) {
	scanResult, err := nmap_wrapper.OSDetectionScan(ctx, request.IP)

	if scanResult == nil {
		fmt.Println("OS detection scanner doesn't have any results")
		return domain.OsDetectionResponse{}, err
	}

	var hostResult string
	for _, hostItem := range scanResult.Hosts {
		if len(hostItem.Addresses) > 0 {
			hostResult = hostItem.Addresses[0].String()
			break
		}
	}

	responseResult := domain.OsDetectionResponse{
		TaskID:   request.TaskID,
		Host:     hostResult,
		Name:     "unknown",
		Accuracy: 0,
		Vendor:   "unknown",
		Family:   "unknown",
		Type:     "unknown",
	}

	for _, hostItem := range scanResult.Hosts {
		if len(hostItem.Addresses) == 0 {
			continue
		}

		if len(hostItem.OS.Matches) > 0 {
			osMatch := hostItem.OS.Matches[0]
			responseResult.Name = osMatch.Name
			responseResult.Accuracy = osMatch.Accuracy

			if len(osMatch.Classes) > 0 {
				osClass := osMatch.Classes[0]
				responseResult.Vendor = osClass.Vendor
				responseResult.Family = osClass.Family
				responseResult.Type = osClass.Type
			}
			break
		}
	}

	return responseResult, err
}

func HostDiscoveryScanner(ctx context.Context, request domain.HostDiscoveryRequest) (response domain.HostDiscoveryResponse, err error) {
	fmt.Printf("Starting host discovery for %s\n", request.IP)
	scanResult, err := nmap_wrapper.HostDiscovery(ctx, request.IP)

	if scanResult == nil {
		fmt.Println("Host discovery scanner doesn't have any results")
		return domain.HostDiscoveryResponse{}, err
	}

	fmt.Printf("Host discovery found %d hosts\n", len(scanResult.Hosts))

	var hostResult string
	for _, host := range scanResult.Hosts {
		if len(host.Addresses) > 0 {
			hostResult = host.Addresses[0].String()
			break
		}
	}

	if hostResult == "" {
		hostResult = request.IP
	}

	responseResult := domain.HostDiscoveryResponse{
		TaskID:    request.TaskID,
		Host:      hostResult,
		HostUP:    0,
		HostTotal: len(scanResult.Hosts),
		Status:    "unknown",
		DNS:       "unknown",
		Reason:    "unknown",
	}

	for i, host := range scanResult.Hosts {
		fmt.Printf("Host %d: Addresses=%d, Status=%s, Reason=%s\n", i, len(host.Addresses), host.Status.State, host.Status.Reason)

		if len(host.Addresses) == 0 {
			continue
		}

		if host.Status.State == "up" {
			responseResult.HostUP++
		}

		if hostResult == host.Addresses[0].String() {
			responseResult.Status = host.Status.State
			responseResult.Reason = host.Status.Reason

			if len(host.Hostnames) > 0 {
				responseResult.DNS = host.Hostnames[0].Name
			}
		}
	}

	fmt.Printf("Final result: Host=%s, UP=%d, Total=%d, Status=%s\n", responseResult.Host, responseResult.HostUP, responseResult.HostTotal, responseResult.Status)

	return responseResult, err
}
