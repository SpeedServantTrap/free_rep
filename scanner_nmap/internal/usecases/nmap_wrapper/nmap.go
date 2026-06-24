package nmap_wrapper

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Ullaakut/nmap/v3"
)

func UDPScan(ctx context.Context, target string, ports string) (*nmap.Run, error) {
	scanCtx, cancel := context.WithTimeout(ctx, 6*time.Minute)
	defer cancel()

	// Try with privileged mode first
	udpScanner, err := nmap.NewScanner(
		scanCtx,
		nmap.WithTargets(target),
		nmap.WithPorts(ports),
		nmap.WithUDPScan(),
		nmap.WithPrivileged(),
		nmap.WithSkipHostDiscovery(),
		nmap.WithTimingTemplate(5),
		nmap.WithMaxRetries(0),
		nmap.WithHostTimeout(5*time.Minute),
		nmap.WithOpenOnly(), // Only report open ports
	)

	if err != nil {
		log.Printf("Failed to create privileged UDP scanner, trying unprivileged: %v", err)
		// Fallback to unprivileged mode
		udpScanner, err = nmap.NewScanner(
			scanCtx,
			nmap.WithTargets(target),
			nmap.WithPorts(ports),
			nmap.WithUDPScan(),
			nmap.WithSkipHostDiscovery(),
			nmap.WithTimingTemplate(5),
			nmap.WithMaxRetries(0),
			nmap.WithHostTimeout(5*time.Minute),
			nmap.WithOpenOnly(), // Only report open ports
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create unprivileged UDP scanner: %w", err)
		}
	}

	result, warnings, err := udpScanner.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run UDP scanner: %w", err)
	}

	if len(*warnings) > 0 {
		log.Printf("UDP scan warning: %v\n", *warnings)
	}

	return result, nil
}

func TCPScan(ctx context.Context, target string, ports string) (*nmap.Run, error) {
	scanCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	tcpScanner, err := nmap.NewScanner(
		scanCtx,
		nmap.WithTargets(target),
		nmap.WithPorts(ports),
		nmap.WithConnectScan(),
		nmap.WithSkipHostDiscovery(),
		nmap.WithTimingTemplate(3),
		nmap.WithMaxRetries(1),
		nmap.WithOpenOnly(), // Only report open ports
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create TCP scanner: %w", err)
	}

	result, warnings, err := tcpScanner.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run TCP scanner: %w", err)
	}

	if len(*warnings) > 0 {
		log.Printf("TCP scan warning: %v\n", *warnings)
	}

	return result, nil
}

func OSDetectionScan(ctx context.Context, target string) (*nmap.Run, error) {
	scanCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	scanner, err := nmap.NewScanner(
		scanCtx,
		nmap.WithTargets(target),
		nmap.WithOSDetection(),
		nmap.WithPrivileged(),
		nmap.WithTimingTemplate(5),
		nmap.WithSkipHostDiscovery(),
		nmap.WithMaxRetries(0),
		nmap.WithOSScanGuess(),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create OS detection scanner: %w", err)
	}

	result, warnings, err := scanner.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run OS detection scanner: %w", err)
	}

	if len(*warnings) > 0 {
		log.Printf("OS detection warnings: %v\n", *warnings)
	}

	return result, nil
}

func HostDiscovery(ctx context.Context, target string) (*nmap.Run, error) {
	scanCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	scanner, err := nmap.NewScanner(
		scanCtx,
		nmap.WithTargets(target),
		nmap.WithPorts("22,80,443"),
		nmap.WithConnectScan(),
		nmap.WithSkipHostDiscovery(),
		nmap.WithTimingTemplate(4),
		nmap.WithMaxRetries(1),
		nmap.WithHostTimeout(10*time.Second),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create host discovery scanner: %w", err)
	}

	result, warnings, err := scanner.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run host discovery scanner: %w", err)
	}

	if len(*warnings) > 0 {
		log.Printf("Host discovery warnings: %v\n", *warnings)
	}

	return result, nil
}
