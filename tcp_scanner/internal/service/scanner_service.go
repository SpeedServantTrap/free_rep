package service

import (
	"context"
	"fmt"
	"net"
	"strings"
	"regexp"
	"time"

	"test_tcp/internal/config"
	"test_tcp/pkg/logger"
	"test_tcp/pkg/queue"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Service struct {
	cfg *config.Config
	log logger.Logger
	mq  *queue.RabbitMQ
	mio *minio.Client
}

func Run(ctx context.Context, cfg *config.Config, log logger.Logger) error {
	mq, err := queue.New(cfg.RabbitMQURL, cfg.ScannerName)
	if err != nil {
		return fmt.Errorf("rabbitmq connect: %w", err)
	}
	defer mq.Close()
	log.Infof("Connected RabbitMQ, queue=%s", cfg.ScannerName)

	mio, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOAccess, cfg.MinIOSecret, ""),
		Secure: false,
	})
	if err != nil {
		return fmt.Errorf("minio connect: %w", err)
	}
	exists, err := mio.BucketExists(ctx, cfg.MinIOBucket)
	if err != nil {
		return fmt.Errorf("minio bucket check: %w", err)
	}
	if !exists {
		if err := mio.MakeBucket(ctx, cfg.MinIOBucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("minio make bucket: %w", err)
		}
	}
	log.Infof("Connected MinIO bucket=%s", cfg.MinIOBucket)

	s := &Service{cfg: cfg, log: log, mq: mq, mio: mio}
	msgs, err := mq.Consume(ctx)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}
	log.Infof("Waiting for TCP tasks...")

	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-msgs:
			if !ok {
				return fmt.Errorf("rabbit channel closed")
			}
			s.handle(ctx, d)
		}
	}
}

func (s *Service) handle(ctx context.Context, d queue.Delivery) {
	req, err := queue.ParseTCPRequest(d.Body)
	if err != nil {
		s.log.Errorf("bad msg: %v", err)
		s.mq.Nack(d)
		return
	}

	s.log.Infof("TCP read: %s:%s (task=%s)", req.Host, req.Port, req.TaskID)
	hexData, decoded, readErr := s.readTCP(ctx, req.Host, req.Port)
	var objKey string
	if readErr == nil {
		objKey = fmt.Sprintf("tcp/%s/%s/%d.txt", sanitizeObjectPart(req.Host), sanitizeObjectPart(req.Port), time.Now().UnixNano())
		payload := decoded
		if payload == "" {
			payload = humanString(hexData)
		}
		reader := strings.NewReader(payload)
		_, err := s.mio.PutObject(ctx, s.cfg.MinIOBucket, objKey, reader, int64(reader.Len()), minio.PutObjectOptions{ContentType: "text/plain; charset=utf-8"})
		if err != nil {
			s.log.Errorf("minio put: %v", err)
		}
		// Removed MongoDB insertion - raw data goes to MinIO, decoded data goes to L3 via backend
	}

	if d.ReplyTo != "" {
		resp := queue.TCPResponse{TaskID: req.TaskID, Host: req.Host, Port: req.Port, HexObjectKey: objKey, DecodedText: decoded, Status: "completed"}
		if readErr != nil {
			resp.Status = "failed"
			resp.Error = readErr.Error()
		}
		if err := s.mq.Reply(d.ReplyTo, d.CorrelationId, resp); err != nil {
			s.log.Errorf("reply: %v", err)
		}
	}
	s.mq.Ack(d)
}

func (s *Service) readTCP(ctx context.Context, host, port string) (raw []byte, decoded string, err error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		raw, decoded, err = s.readTCPOnce(ctx, host, port)
		if err == nil && len(raw) > 0 {
			return raw, decoded, nil
		}
		if err != nil {
			lastErr = err
		}
		if attempt == 0 {
			time.Sleep(300 * time.Millisecond)
		}
	}

	if len(raw) > 0 {
		return raw, decoded, nil
	}
	if lastErr != nil {
		return nil, "", lastErr
	}
	return nil, "", fmt.Errorf("empty banner")
}

func (s *Service) readTCPOnce(ctx context.Context, host, port string) (raw []byte, decoded string, err error) {
	addr := net.JoinHostPort(host, port)
	conn, err := (&net.Dialer{Timeout: s.cfg.ConnTimeout}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, "", err
	}
	defer conn.Close()

	// Stage 1: passive greeting read (some services send banner immediately).
	buf := make([]byte, 0, 8192)
	_ = conn.SetDeadline(time.Now().Add(1500 * time.Millisecond))
	buf = append(buf, readAll(conn)...)

	// Stage 2: if nothing received, send protocol-specific probe and read again.
	if len(buf) == 0 {
		probe := protocolProbe(port, host)
		if len(probe) == 0 {
			probe = []byte("\r\n")
		}
		if len(probe) > 0 {
			_, _ = conn.Write(probe)
		}
		_ = conn.SetDeadline(time.Now().Add(s.cfg.ReadTimeout))
		buf = append(buf, readAll(conn)...)
	}

	banner := humanString(buf)
	if banner == "" && len(buf) > 0 {
		banner = bytesToHexLine(buf)
	}
	return buf, banner, nil
}

func readAll(conn net.Conn) []byte {
	buf := make([]byte, 0, 8192)
	tmp := make([]byte, 4096)
	for {
		n, er := conn.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if er != nil {
			break
		}
	}
	return buf
}

func bytesToHexLine(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	if len(b) > 256 {
		b = b[:256]
	}
	var sb strings.Builder
	for i, v := range b {
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(fmt.Sprintf("%02X", v))
	}
	return sb.String()
}

func humanString(b []byte) string {
	out := make([]byte, 0, len(b))
	for _, x := range b {
		switch x {
		case '\r':
			out = append(out, '\\', 'r')
		case '\n':
			out = append(out, '\\', 'n')
		case '\t':
			out = append(out, '\\', 't')
		default:
			if x >= 32 && x <= 126 {
				out = append(out, x)
			}
		}
	}
	return string(out)
}

func sanitizeObjectPart(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return "unknown"
	}
	re := regexp.MustCompile(`[^a-z0-9._-]+`)
	v = re.ReplaceAllString(v, "_")
	if v == "" {
		return "unknown"
	}
	return v
}

func protocolProbe(port, host string) []byte {
	switch port {
	case "22":
		// Some SSH daemons send identification after client speaks first.
		return []byte("SSH-2.0-scanner\r\n")
	case "80", "8080", "8000", "8008", "8013", "9000", "9001":
		return []byte(fmt.Sprintf("HEAD / HTTP/1.0\r\nHost: %s\r\n\r\n", host))
	case "5432":
		// PostgreSQL SSLRequest packet (8 bytes) often triggers an immediate single-byte reply: 'S' or 'N'.
		return []byte{0x00, 0x00, 0x00, 0x08, 0x04, 0xD2, 0x16, 0x2F}
	case "25", "587", "110", "143":
		return []byte("\r\n")
	default:
		return nil
	}
}
