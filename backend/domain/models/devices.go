package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// L2Device represents a unique device at the Data-Link (MAC) layer.
// Key: MAC address (used as _id).
// One MAC address can have multiple associated IP addresses
// (e.g. a device with several interfaces or DHCP lease changes).
type L2Device struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MAC         string             `bson:"mac"           json:"mac"`
	Vendor      string             `bson:"vendor"        json:"vendor"`
	IPAddresses []string           `bson:"ip_addresses"  json:"ip_addresses"`
	FirstSeen   time.Time          `bson:"first_seen"    json:"first_seen"`
	LastSeen    time.Time          `bson:"last_seen"     json:"last_seen"`
}

// IPAddressInfo contains information about a specific IP address associated with a MAC
type IPAddressInfo struct {
	IP        string    `bson:"ip" json:"ip"`                   // IP address
	FirstSeen time.Time `bson:"first_seen" json:"first_seen"` // First time this IP was seen with this MAC
	LastSeen  time.Time `bson:"last_seen" json:"last_seen"`   // Last time this IP was seen with this MAC
}

// L2DeviceNew represents a unique device at the Data-Link (MAC) layer with new structure.
// Key: MAC address (used as _id).
// Information accumulates from different scans (ARP, NMAP, etc.).
type L2DeviceNew struct {
	ID          string         `bson:"_id" json:"id"`             // MAC address as ID
	Vendor      string         `bson:"vendor" json:"vendor"`      // Device vendor from MAC
	ScannerTypes []string      `bson:"scanner_types" json:"scanner_types"` // Scanner types (e.g., "arp")
	IPAddresses []IPAddressInfo `bson:"ip_addresses" json:"ip_addresses"` // IP addresses with their own timestamps
	FirstSeen   time.Time      `bson:"first_seen" json:"first_seen"` // Overall first seen (across all IPs)
	LastSeen    time.Time      `bson:"last_seen" json:"last_seen"`   // Overall last seen (across all IPs)
}

// L3Device represents a unique device at the Network (IP) layer.
// Key: IP address.
// References its L2 parent via the MAC field.
type L3Device struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	IP        string             `bson:"ip"            json:"ip"`
	MAC       string             `bson:"mac"           json:"mac"`
	Vendor    string             `bson:"vendor"        json:"vendor"`
	FirstSeen time.Time          `bson:"first_seen"    json:"first_seen"`
	LastSeen  time.Time          `bson:"last_seen"     json:"last_seen"`
}

// L3DeviceNew represents a unique device at the Network (IP) layer with new structure.
// Key: IP address (used as _id).
// Information accumulates from different scans (ARP, NMAP, ICMP, TCP).
type L3DeviceNew struct {
	ID             string    `bson:"_id" json:"id"` // IP address as ID
	MAC            string    `bson:"mac" json:"mac"`                             // MAC address or "-" if not available
	TCPOpenPorts   []string  `bson:"tcp_open_ports" json:"tcp_open_ports"`       // Comma-separated TCP open ports
	UDPOpenPorts   []string  `bson:"udp_open_ports" json:"udp_open_ports"`       // Comma-separated UDP open ports
	OS             string    `bson:"os" json:"os"`                               // OS detection result or "-"
	DNS            string    `bson:"dns" json:"dns"`                              // DNS result or "-"
	PacketsReached []string  `bson:"packets_reached" json:"packets_reached"`     // Comma-separated packets reached (ICMP)
	TCPBanners     map[string]string `bson:"tcp_banners,omitempty" json:"tcp_banners,omitempty"` // port -> banner
	ScannerTypes   []string  `bson:"scanner_types" json:"scanner_types"`        // Comma-separated scanner types (4 types: arp, nmap, icmp, tcp)
	FirstSeen      time.Time `bson:"first_seen" json:"first_seen"`
	LastSeen       time.Time `bson:"last_seen" json:"last_seen"`
}

