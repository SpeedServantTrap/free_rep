package rabbitmq

import (
	"context"
	"log"
	"time"

	"backend/domain/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// removeDuplicates removes duplicate strings from a slice
func removeDuplicates(slice []string) []string {
	seen := make(map[string]struct{})
	result := []string{}

	for _, item := range slice {
		if _, exists := seen[item]; !exists {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}

type Repository struct {
	db *Database
}

func NewRepository(db *Database) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SaveICMPHistory(record *models.ICMPHistoryRecord) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	record.ScanType = "icmp"
	record.CreatedAt = time.Now()
	_, err := r.db.ICMPCollection().UpdateOne(
		ctx,
		bson.M{"task_id": record.TaskID, "scan_type": "icmp"},
		bson.M{"$setOnInsert": record},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		log.Printf("Error saving ICMP history: %v", err)
		return err
	}

	log.Printf("ICMP history saved successfully for task: %s", record.TaskID)
	return nil
}

func (r *Repository) GetICMPHistory(limit int) ([]models.ICMPHistoryRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := r.db.ICMPCollection().Find(ctx, bson.M{"scan_type": "icmp"}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []models.ICMPHistoryRecord
	if err = cursor.All(ctx, &records); err != nil {
		return nil, err
	}

	return records, nil
}

func (r *Repository) DeleteICMPHistory() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.db.ICMPCollection().DeleteMany(ctx, bson.M{"scan_type": "icmp"})
	if err != nil {
		log.Printf("Error deleting ICMP history: %v", err)
		return err
	}

	log.Printf("Deleted %d ICMP history records", result.DeletedCount)
	return nil
}

func (r *Repository) GetICMPHistoryByTargets(targets []string, limit int) ([]models.ICMPHistoryRecord, error) {
	if len(targets) == 0 {
		return r.GetICMPHistory(limit)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"scan_type": "icmp", "targets": bson.M{"$in": targets}}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	cursor, err := r.db.ICMPCollection().Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []models.ICMPHistoryRecord
	if err = cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (r *Repository) GetICMPHistoryByID(id string) (*models.ICMPHistoryRecord, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var rec models.ICMPHistoryRecord
	err = r.db.ICMPCollection().FindOne(ctx, bson.M{"_id": objID, "scan_type": "icmp"}).Decode(&rec)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *Repository) SaveNmapTcpUdpHistory(record *models.NmapTcpUdpHistoryRecord) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	record.ScanType = "nmap_tcp_udp"
	record.CreatedAt = time.Now()
	_, err := r.db.NmapTcpUdpCollection().UpdateOne(
		ctx,
		bson.M{"task_id": record.TaskID, "scan_type": "nmap_tcp_udp"},
		bson.M{"$setOnInsert": record},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		log.Printf("Error saving Nmap TCP/UDP history: %v", err)
		return err
	}

	log.Printf("Nmap TCP/UDP history saved successfully for task: %s", record.TaskID)
	return nil
}

func (r *Repository) GetNmapTcpUdpHistory(limit int) ([]models.NmapTcpUdpHistoryRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := r.db.NmapTcpUdpCollection().Find(ctx, bson.M{"scan_type": "nmap_tcp_udp"}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []models.NmapTcpUdpHistoryRecord
	if err = cursor.All(ctx, &records); err != nil {
		return nil, err
	}

	return records, nil
}

func (r *Repository) DeleteNmapTcpUdpHistory() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.db.NmapTcpUdpCollection().DeleteMany(ctx, bson.M{"scan_type": "nmap_tcp_udp"})
	if err != nil {
		log.Printf("Error deleting Nmap TCP/UDP history: %v", err)
		return err
	}

	log.Printf("Deleted %d Nmap TCP/UDP history records", result.DeletedCount)
	return nil
}

func (r *Repository) GetNmapTcpUdpHistoryByIP(ip string, limit int) ([]models.NmapTcpUdpHistoryRecord, error) {
	if ip == "" {
		return r.GetNmapTcpUdpHistory(limit)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"scan_type": "nmap_tcp_udp", "ip": ip}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	cursor, err := r.db.NmapTcpUdpCollection().Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []models.NmapTcpUdpHistoryRecord
	if err = cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (r *Repository) GetNmapTcpUdpHistoryByID(id string) (*models.NmapTcpUdpHistoryRecord, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var rec models.NmapTcpUdpHistoryRecord
	err = r.db.NmapTcpUdpCollection().FindOne(ctx, bson.M{"_id": objID, "scan_type": "nmap_tcp_udp"}).Decode(&rec)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *Repository) SaveNmapOsDetectionHistory(record *models.NmapOsDetectionHistoryRecord) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	record.ScanType = "nmap_os_detection"
	record.CreatedAt = time.Now()
	_, err := r.db.NmapOsDetectionCollection().UpdateOne(
		ctx,
		bson.M{"task_id": record.TaskID, "scan_type": "nmap_os_detection"},
		bson.M{"$setOnInsert": record},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		log.Printf("Error saving Nmap OS Detection history: %v", err)
		return err
	}

	log.Printf("Nmap OS Detection history saved successfully for task: %s", record.TaskID)
	return nil
}

func (r *Repository) GetNmapOsDetectionHistory(limit int) ([]models.NmapOsDetectionHistoryRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := r.db.NmapOsDetectionCollection().Find(ctx, bson.M{"scan_type": "nmap_os_detection"}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []models.NmapOsDetectionHistoryRecord
	if err = cursor.All(ctx, &records); err != nil {
		return nil, err
	}

	return records, nil
}

func (r *Repository) DeleteNmapOsDetectionHistory() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.db.NmapOsDetectionCollection().DeleteMany(ctx, bson.M{"scan_type": "nmap_os_detection"})
	if err != nil {
		log.Printf("Error deleting Nmap OS Detection history: %v", err)
		return err
	}

	log.Printf("Deleted %d Nmap OS Detection history records", result.DeletedCount)
	return nil
}

func (r *Repository) GetNmapOsDetectionHistoryByIP(ip string, limit int) ([]models.NmapOsDetectionHistoryRecord, error) {
	if ip == "" {
		return r.GetNmapOsDetectionHistory(limit)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"scan_type": "nmap_os_detection", "ip": ip}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	cursor, err := r.db.NmapOsDetectionCollection().Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []models.NmapOsDetectionHistoryRecord
	if err = cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (r *Repository) GetNmapOsDetectionHistoryByID(id string) (*models.NmapOsDetectionHistoryRecord, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var rec models.NmapOsDetectionHistoryRecord
	err = r.db.NmapOsDetectionCollection().FindOne(ctx, bson.M{"_id": objID, "scan_type": "nmap_os_detection"}).Decode(&rec)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *Repository) SaveNmapHostDiscoveryHistory(record *models.NmapHostDiscoveryHistoryRecord) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	record.ScanType = "nmap_host_discovery"
	record.CreatedAt = time.Now()
	_, err := r.db.NmapHostDiscoveryCollection().UpdateOne(
		ctx,
		bson.M{"task_id": record.TaskID, "scan_type": "nmap_host_discovery"},
		bson.M{"$setOnInsert": record},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		log.Printf("Error saving Nmap Host Discovery history: %v", err)
		return err
	}

	log.Printf("Nmap Host Discovery history saved successfully for task: %s", record.TaskID)
	return nil
}

func (r *Repository) GetNmapHostDiscoveryHistory(limit int) ([]models.NmapHostDiscoveryHistoryRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := r.db.NmapHostDiscoveryCollection().Find(ctx, bson.M{"scan_type": "nmap_host_discovery"}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []models.NmapHostDiscoveryHistoryRecord
	if err = cursor.All(ctx, &records); err != nil {
		return nil, err
	}

	return records, nil
}

func (r *Repository) DeleteNmapHostDiscoveryHistory() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.db.NmapHostDiscoveryCollection().DeleteMany(ctx, bson.M{"scan_type": "nmap_host_discovery"})
	if err != nil {
		log.Printf("Error deleting Nmap Host Discovery history: %v", err)
		return err
	}

	log.Printf("Deleted %d Nmap Host Discovery history records", result.DeletedCount)
	return nil
}

func (r *Repository) GetNmapHostDiscoveryHistoryByIP(ip string, limit int) ([]models.NmapHostDiscoveryHistoryRecord, error) {
	if ip == "" {
		return r.GetNmapHostDiscoveryHistory(limit)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"scan_type": "nmap_host_discovery", "ip": ip}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	cursor, err := r.db.NmapHostDiscoveryCollection().Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []models.NmapHostDiscoveryHistoryRecord
	if err = cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (r *Repository) GetNmapHostDiscoveryHistoryByID(id string) (*models.NmapHostDiscoveryHistoryRecord, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var rec models.NmapHostDiscoveryHistoryRecord
	err = r.db.NmapHostDiscoveryCollection().FindOne(ctx, bson.M{"_id": objID, "scan_type": "nmap_host_discovery"}).Decode(&rec)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *Repository) SaveTCPHistory(record *models.TCPHistoryRecord) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	record.ScanType = "tcp"
	record.CreatedAt = time.Now()
	_, err := r.db.TCPCollection().UpdateOne(
		ctx,
		bson.M{"task_id": record.TaskID, "scan_type": "tcp"},
		bson.M{"$setOnInsert": record},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		log.Printf("Error saving TCP history: %v", err)
		return err
	}

	log.Printf("TCP history saved successfully for task: %s", record.TaskID)
	return nil
}

func (r *Repository) GetTCPHistory(limit int) ([]models.TCPHistoryRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := r.db.TCPCollection().Find(ctx, bson.M{"scan_type": "tcp"}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []models.TCPHistoryRecord
	if err = cursor.All(ctx, &records); err != nil {
		return nil, err
	}

	return records, nil
}

func (r *Repository) GetTCPHistoryByHostPort(host, port string, limit int) ([]models.TCPHistoryRecord, error) {
	if host == "" && port == "" {
		return r.GetTCPHistory(limit)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"scan_type": "tcp"}
	if host != "" {
		filter["host"] = host
	}
	if port != "" {
		filter["port"] = port
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	cursor, err := r.db.TCPCollection().Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []models.TCPHistoryRecord
	if err = cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (r *Repository) GetTCPHistoryByID(id string) (*models.TCPHistoryRecord, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var rec models.TCPHistoryRecord
	err = r.db.TCPCollection().FindOne(ctx, bson.M{"_id": objID, "scan_type": "tcp"}).Decode(&rec)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *Repository) DeleteTCPHistory() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.db.TCPCollection().DeleteMany(ctx, bson.M{"scan_type": "tcp"})
	if err != nil {
		log.Printf("Error deleting TCP history: %v", err)
		return err
	}

	log.Printf("Deleted %d TCP history records", result.DeletedCount)
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Change Events  (scan_type = "change_event" inside l3_devices)
// ──────────────────────────────────────────────────────────────────────────────

// GetChangeEvents returns the most recent change events, optionally filtered by severity.
func (r *Repository) GetChangeEvents(limit int, severity string) ([]models.ChangeEvent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"scan_type": "change_event"}
	if severity != "" && severity != "ALL" {
		filter["severity"] = severity
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := r.db.ChangesCollection().Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []models.ChangeEvent
	if err = cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	return records, nil
}

// GetChangeEventsSince returns events created strictly after `since`, ordered ascending.
func (r *Repository) GetChangeEventsSince(since time.Time) ([]models.ChangeEvent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"scan_type": "change_event", "created_at": bson.M{"$gt": since}}
	opts   := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})

	cursor, err := r.db.ChangesCollection().Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var records []models.ChangeEvent
	if err = cursor.All(ctx, &records); err != nil {
		return nil, err
	}
	return records, nil
}

// DeleteChangeEvents removes all change events from the collection.
func (r *Repository) DeleteChangeEvents() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.db.ChangesCollection().DeleteMany(ctx, bson.M{"scan_type": "change_event"})
	if err != nil {
		log.Printf("Error deleting change events: %v", err)
		return err
	}
	log.Printf("Deleted %d change event records", result.DeletedCount)
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// New L2/L3 Device Repository Methods
// ──────────────────────────────────────────────────────────────────────────────

// SaveOrUpdateL2Device saves or updates an L2 device with data accumulation
func (r *Repository) SaveOrUpdateL2Device(device *models.L2DeviceNew) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	device.LastSeen = now

	// Try to find existing device
	var existing models.L2DeviceNew
	err := r.db.L2DevicesCollection().FindOne(ctx, bson.M{"_id": device.ID}).Decode(&existing)

	if err == nil {
		// Device exists, update it with accumulated data
		update := bson.M{
			"$set": bson.M{
				"last_seen": now,
			},
			"$unset": bson.M{
				"scan_times": "",
				"tcp_banner": "",
			},
		}

		// Update vendor if provided
		if device.Vendor != "" && device.Vendor != existing.Vendor {
			update["$set"].(bson.M)["vendor"] = device.Vendor
		}

		// Append new scanner types if not already present (removing duplicates)
		if len(device.ScannerTypes) > 0 {
			uniqueScannerTypes := append(existing.ScannerTypes, device.ScannerTypes...)
			uniqueScannerTypes = removeDuplicates(uniqueScannerTypes)
			update["$set"].(bson.M)["scanner_types"] = uniqueScannerTypes
		}

		// Append new IP addresses if not already present (removing duplicates)
		if len(device.IPAddresses) > 0 {
			uniqueIPs := append(existing.IPAddresses, device.IPAddresses...)
			uniqueIPs = removeDuplicates(uniqueIPs)
			update["$set"].(bson.M)["ip_addresses"] = uniqueIPs
		}

		_, err = r.db.L2DevicesCollection().UpdateOne(ctx, bson.M{"_id": device.ID}, update)
		if err != nil {
			return err
		}
	} else {
		// Device doesn't exist, insert new one
		device.FirstSeen = now
		_, err = r.db.L2DevicesCollection().InsertOne(ctx, device)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetL2Device retrieves a single L2 device by MAC address
func (r *Repository) GetL2Device(mac string) (*models.L2DeviceNew, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var device models.L2DeviceNew
	err := r.db.L2DevicesCollection().FindOne(ctx, bson.M{"_id": mac}).Decode(&device)
	if err != nil {
		return nil, err
	}
	return &device, nil
}

// GetAllL2Devices retrieves all L2 devices
func (r *Repository) GetAllL2Devices() ([]models.L2DeviceNew, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := r.db.L2DevicesCollection().Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var devices []models.L2DeviceNew
	if err = cursor.All(ctx, &devices); err != nil {
		return nil, err
	}

	return devices, nil
}

// DeleteAllL2Devices removes all L2 devices
func (r *Repository) DeleteAllL2Devices() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.db.L2DevicesCollection().DeleteMany(ctx, bson.D{})
	if err != nil {
		log.Printf("Error deleting L2 devices: %v", err)
		return err
	}
	log.Printf("Deleted %d L2 device records", result.DeletedCount)
	return nil
}

// SaveOrUpdateL3Device saves or updates an L3 device with data accumulation
func (r *Repository) SaveOrUpdateL3Device(device *models.L3DeviceNew) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	device.LastSeen = now

	// Try to find existing device
	var existing models.L3DeviceNew
	err := r.db.ICMPCollection().FindOne(ctx, bson.M{"_id": device.ID}).Decode(&existing)

	if err == nil {
		// Device exists, update it with accumulated data
		update := bson.M{
			"$set": bson.M{
				"last_seen": now,
			},
			"$unset": bson.M{
				"scan_times": "",
			},
		}

		// Update MAC if provided (and not just "-")
		if device.MAC != "" && device.MAC != "-" {
			if existing.MAC == "-" || existing.MAC == "" {
				update["$set"].(bson.M)["mac"] = device.MAC
			}
		}

		// Append TCP ports if provided (removing duplicates)
		if len(device.TCPOpenPorts) > 0 {
			uniquePorts := append(existing.TCPOpenPorts, device.TCPOpenPorts...)
			uniquePorts = removeDuplicates(uniquePorts)
			update["$set"].(bson.M)["tcp_open_ports"] = uniquePorts
		}

		// Append UDP ports if provided (removing duplicates)
		if len(device.UDPOpenPorts) > 0 {
			uniquePorts := append(existing.UDPOpenPorts, device.UDPOpenPorts...)
			uniquePorts = removeDuplicates(uniquePorts)
			update["$set"].(bson.M)["udp_open_ports"] = uniquePorts
		}

		// Update OS if provided (and not just "-")
		if device.OS != "" && device.OS != "-" {
			if existing.OS == "-" || existing.OS == "" {
				update["$set"].(bson.M)["os"] = device.OS
			}
		}

		// Update DNS if provided (and not just "-")
		if device.DNS != "" && device.DNS != "-" {
			if existing.DNS == "-" || existing.DNS == "" {
				update["$set"].(bson.M)["dns"] = device.DNS
			}
		}

		// Append packets reached if provided (removing duplicates)
		if len(device.PacketsReached) > 0 {
			uniquePackets := append(existing.PacketsReached, device.PacketsReached...)
			uniquePackets = removeDuplicates(uniquePackets)
			update["$set"].(bson.M)["packets_reached"] = uniquePackets
		}

		// Merge TCP banners by port
		if len(device.TCPBanners) > 0 {
			merged := make(map[string]string)
			for port, banner := range existing.TCPBanners {
				merged[port] = banner
			}
			for port, banner := range device.TCPBanners {
				if port == "" || banner == "" {
					continue
				}
				merged[port] = banner
			}
			if len(merged) > 0 {
				update["$set"].(bson.M)["tcp_banners"] = merged
			}
		}

		// Append scanner types if provided (removing duplicates)
		if len(device.ScannerTypes) > 0 {
			uniqueScannerTypes := append(existing.ScannerTypes, device.ScannerTypes...)
			uniqueScannerTypes = removeDuplicates(uniqueScannerTypes)
			update["$set"].(bson.M)["scanner_types"] = uniqueScannerTypes
		}

		_, err = r.db.ICMPCollection().UpdateOne(ctx, bson.M{"_id": device.ID}, update)
		if err != nil {
			log.Printf("Error updating L3 device: %v", err)
			return err
		}
		log.Printf("L3 device updated successfully: %s", device.ID)
	} else {
		// Device doesn't exist, insert new one
		device.FirstSeen = now
		// Set default values for fields that might be "-"
		if device.MAC == "" {
			device.MAC = "-"
		}
		if device.OS == "" {
			device.OS = "-"
		}
		if device.DNS == "" {
			device.DNS = "-"
		}
		_, err = r.db.ICMPCollection().InsertOne(ctx, device)
		if err != nil {
			log.Printf("Error inserting L3 device: %v", err)
			return err
		}
		log.Printf("L3 device inserted successfully: %s", device.ID)
	}

	return nil
}

// GetL3Device retrieves a single L3 device by IP address
func (r *Repository) GetL3Device(ip string) (*models.L3DeviceNew, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("[GetL3Device] Searching for device with IP: '%s'", ip)
	var device models.L3DeviceNew
	err := r.db.L3DevicesCollection().FindOne(ctx, bson.M{"_id": ip}).Decode(&device)
	if err != nil {
		log.Printf("[GetL3Device] Device not found for IP: '%s', error: %v", ip, err)
		return nil, err
	}
	log.Printf("[GetL3Device] Found device for IP: '%s', MAC: '%s'", ip, device.MAC)
	return &device, nil
}

// GetAllL3Devices retrieves all L3 devices
func (r *Repository) GetAllL3Devices() ([]models.L3DeviceNew, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Printf("[GetAllL3Devices] Retrieving all L3 devices")
	cursor, err := r.db.L3DevicesCollection().Find(ctx, bson.D{})
	if err != nil {
		log.Printf("[GetAllL3Devices] Error finding devices: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var devices []models.L3DeviceNew
	if err = cursor.All(ctx, &devices); err != nil {
		log.Printf("[GetAllL3Devices] Error decoding devices: %v", err)
		return nil, err
	}
	log.Printf("[GetAllL3Devices] Retrieved %d L3 devices", len(devices))
	return devices, nil
}

// DeleteAllL3Devices removes all L3 devices
func (r *Repository) DeleteAllL3Devices() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := r.db.ICMPCollection().DeleteMany(ctx, bson.D{})
	if err != nil {
		log.Printf("Error deleting L3 devices: %v", err)
		return err
	}
	log.Printf("Deleted %d L3 device records", result.DeletedCount)
	return nil
}

// DropL2Collection drops the entire L2 collection
func (r *Repository) DropL2Collection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := r.db.L2DevicesCollection().Drop(ctx)
	if err != nil {
		log.Printf("Error dropping L2 collection: %v", err)
		return err
	}
	log.Printf("L2 collection dropped successfully")
	return nil
}

// DropL3Collection drops the entire L3 collection
func (r *Repository) DropL3Collection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := r.db.ICMPCollection().Drop(ctx)
	if err != nil {
		log.Printf("Error dropping L3 collection: %v", err)
		return err
	}
	log.Printf("L3 collection dropped successfully")
	return nil
}

