package usecases

import (
	"context"
	"fmt"
	"scanner_nmap/internal/domain"
	"scanner_nmap/internal/usecases/nmap_wrapper"

	"github.com/Ullaakut/nmap/v3"
)

func UdpTcpScanner(ctx context.Context, request domain.ScanTcpUdpRequest) (response domain.ScanTcpUdpResponse, err error) {
	var scanResult *nmap.Run

	scanType := "TCP"
	if request.ScannerType == "UDP" || request.ScannerType == "udp_scan" {
		scanType = "UDP"
	}

	fmt.Printf("Starting %s scan for %s on ports %s\n", scanType, request.IP, request.Ports)

	if request.ScannerType == "UDP" || request.ScannerType == "udp_scan" {
		scanResult, err = nmap_wrapper.UDPScan(ctx, request.IP, request.Ports)
	} else {
		scanResult, err = nmap_wrapper.TCPScan(ctx, request.IP, request.Ports)
	}

	if err != nil {
		fmt.Printf("%s scan error: %v\n", scanType, err)
		return domain.ScanTcpUdpResponse{}, err
	}

	if scanResult == nil {
		fmt.Printf("%s scanner doesn't have any results\n", scanType)
		return domain.ScanTcpUdpResponse{}, fmt.Errorf("no scan results")
	}

	// Detailed logging of scan results
	fmt.Printf("Scan result stats: %d hosts found\n", len(scanResult.Hosts))
	for i, host := range scanResult.Hosts {
		fmt.Printf("Host %d: Status=%s, %d addresses, %d ports\n",
			i, host.Status.State, len(host.Addresses), len(host.Ports))

		// Log first few ports for debugging
		for j, port := range host.Ports {
			if j < 5 { // Log first 5 ports
				fmt.Printf("  Port %d: ID=%d, Protocol=%s, State=%s, Service=%s\n",
					j, port.ID, port.Protocol, port.State.State, port.Service.Name)
			}
		}
		if len(host.Ports) > 5 {
			fmt.Printf("  ... and %d more ports\n", len(host.Ports)-5)
		}
	}

	var hostResult string
	for _, host := range scanResult.Hosts {
		if len(host.Addresses) > 0 {
			hostResult = host.Addresses[0].String()
			break
		}
	}

	portInfo := domain.PortTcpUdpInfo{}
	totalPorts := 0
	openPorts := 0

	for _, host := range scanResult.Hosts {
		if len(host.Addresses) == 0 {
			continue
		}

		portInfo.Status = host.Status.State

		for _, port := range host.Ports {
			totalPorts++
			fmt.Printf("Processing port %d: Protocol=%s, State=%s\n", port.ID, port.Protocol, port.State.State)

			// Only add ports that are actually open
			if port.State.State != "open" {
				fmt.Printf("  Skipping port %d - not open (state: %s)\n", port.ID, port.State.State)
				continue
			}

			openPorts++
			portInfo.AllPorts = append(portInfo.AllPorts, port.ID)
			portInfo.Protocols = append(portInfo.Protocols, port.Protocol)
			portInfo.State = append(portInfo.State, port.State.State)

			serviceName := port.Service.Name
			if serviceName == "" {
				serviceName = "unknown"
			}
			portInfo.ServiceName = append(portInfo.ServiceName, serviceName)
			fmt.Printf("  Added open port %d (%s)\n", port.ID, port.Protocol)
		}
	}

	fmt.Printf("%s scan completed: Total ports scanned: %d, Open ports found: %d\n", scanType, totalPorts, openPorts)

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
