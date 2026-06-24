package usecases

import (
	"context"
	"fmt"
	"scanner_nmap/internal/domain"
	"scanner_nmap/internal/usecases/nmap_wrapper"

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
