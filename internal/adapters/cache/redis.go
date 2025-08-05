package cache

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RedisClient struct {
	Addr       string
	logger     *slog.Logger
	mu         sync.Mutex
	poolSize   int
	connPool   chan net.Conn
	done       chan struct{}
	reconnTime time.Duration
}

func NewRedisClient(addr string, logger *slog.Logger, poolSize int) (*RedisClient, error) {
	rc := &RedisClient{
		Addr:       addr,
		logger:     logger,
		poolSize:   poolSize,
		connPool:   make(chan net.Conn, poolSize),
		done:       make(chan struct{}),
		reconnTime: 1 * time.Second,
	}

	// Initialize connection pool
	for i := 0; i < poolSize; i++ {
		conn, err := rc.createConnection()
		if err != nil {
			rc.Close() // Clean up any created connections
			return nil, fmt.Errorf("failed to initialize connection pool: %v", err)
		}
		rc.connPool <- conn
	}

	// Start background reconnection goroutine
	go rc.connectionManager()

	return rc, nil
}

func (rc *RedisClient) createConnection() (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", rc.Addr, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
		tcpConn.SetNoDelay(true)
	}

	// Verify connection
	conn.SetDeadline(time.Now().Add(2 * time.Second))
	defer conn.SetDeadline(time.Time{})

	_, err = conn.Write([]byte("*1\r\n$4\r\nPING\r\n"))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("redis ping failed: %v", err)
	}

	buf := make([]byte, 1024)
	_, err = conn.Read(buf)
	if err != nil || !strings.Contains(string(buf), "+PONG") {
		conn.Close()
		return nil, fmt.Errorf("redis ping response invalid: %v", err)
	}

	return conn, nil
}

func (rc *RedisClient) connectionManager() {
	for {
		select {
		case <-rc.done:
			return
		case <-time.After(rc.reconnTime):
			rc.mu.Lock()
			// Check all connections in pool
			for i := 0; i < len(rc.connPool); i++ {
				conn := <-rc.connPool
				if err := rc.checkConnection(conn); err != nil {
					conn.Close()
					newConn, err := rc.createConnection()
					if err != nil {
						rc.logger.Error("Failed to recreate connection", "error", err)
						continue
					}
					rc.connPool <- newConn
				} else {
					rc.connPool <- conn
				}
			}
			rc.mu.Unlock()
		}
	}
}

func (rc *RedisClient) checkConnection(conn net.Conn) error {
	if conn == nil {
		return fmt.Errorf("connection is nil")
	}

	conn.SetDeadline(time.Now().Add(500 * time.Millisecond))
	defer conn.SetDeadline(time.Time{})

	_, err := conn.Write([]byte("*1\r\n$4\r\nPING\r\n"))
	if err != nil {
		return fmt.Errorf("ping failed: %v", err)
	}

	buf := make([]byte, 1024)
	_, err = conn.Read(buf)
	if err != nil || !strings.Contains(string(buf), "+PONG") {
		return fmt.Errorf("invalid ping response: %v", err)
	}

	return nil
}

func (rc *RedisClient) getConn() (net.Conn, error) {
	select {
	case conn := <-rc.connPool:
		return conn, nil
	case <-time.After(500 * time.Millisecond):
		return nil, fmt.Errorf("connection pool timeout")
	}
}

func (rc *RedisClient) putConn(conn net.Conn) {
	select {
	case rc.connPool <- conn:
	default:
		conn.Close()
	}
}

func (rc *RedisClient) execCommand(ctx context.Context, cmd string, args ...string) ([]string, error) {
	conn, err := rc.getConn()
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %v", err)
	}
	defer rc.putConn(conn)

	deadline, ok := ctx.Deadline()
	if ok {
		conn.SetDeadline(deadline)
		defer conn.SetDeadline(time.Time{})
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*%d\r\n", len(args)+1))
	sb.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(cmd), cmd))

	for _, arg := range args {
		sb.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(arg), arg))
	}

	cmdStr := sb.String()

	_, err = conn.Write([]byte(cmdStr))
	if err != nil {
		return nil, fmt.Errorf("failed to write command: %v", err)
	}

	reader := bufio.NewReader(conn)
	return readRESP(reader)
}

func (rc *RedisClient) Close() {
	close(rc.done)
	rc.mu.Lock()
	defer rc.mu.Unlock()

	close(rc.connPool)
	for conn := range rc.connPool {
		conn.Close()
	}
}

func (rc *RedisClient) GetLatestPrice(ctx context.Context, exchange, symbol string) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	var keys []string
	if exchange == "" {
		resp, err := rc.execCommand(ctx, "KEYS", fmt.Sprintf("price:*:%s", symbol))
		if err != nil {
			return 0, err
		}
		keys = resp
	} else {
		keys = []string{fmt.Sprintf("price:%s:%s", exchange, symbol)}
	}

	var maxPrice float64
	var maxTime int64
	var found bool

	for _, key := range keys {
		resp, err := rc.execCommand(ctx, "ZREVRANGE", key, "0", "0", "WITHSCORES")
		if err != nil || len(resp) < 2 {
			continue
		}

		price, err := strconv.ParseFloat(resp[0], 64)
		if err != nil {
			continue
		}

		timestamp, err := strconv.ParseInt(resp[1], 10, 64)
		if err != nil {
			continue
		}

		if !found || timestamp > maxTime {
			maxPrice = price
			maxTime = timestamp
			found = true
		}
	}

	if !found {
		return 0, fmt.Errorf("no prices found")
	}

	return maxPrice, nil
}

func readRESP(reader *bufio.Reader) ([]string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read RESP line: %v", err)
	}

	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return nil, fmt.Errorf("empty RESP line")
	}

	switch line[0] {
	case '*':
		count, err := strconv.Atoi(line[1:])
		if err != nil {
			return nil, fmt.Errorf("invalid array length: %v", err)
		}

		if count == -1 {
			return nil, nil
		}

		if count == 0 {
			return []string{}, nil
		}

		var elements []string
		for i := 0; i < count; i++ {
			typeLine, err := reader.ReadString('\n')
			if err != nil {
				return nil, fmt.Errorf("failed to read element type: %v", err)
			}
			typeLine = strings.TrimSpace(typeLine)

			if len(typeLine) == 0 {
				return nil, fmt.Errorf("empty element type")
			}

			if typeLine[0] != '$' {
				return nil, fmt.Errorf("expected bulk string type ($), got %q", typeLine)
			}

			length, err := strconv.Atoi(typeLine[1:])
			if err != nil {
				return nil, fmt.Errorf("invalid bulk string length: %v", err)
			}

			if length == -1 {
				elements = append(elements, "")
				continue
			}

			data := make([]byte, length)
			_, err = io.ReadFull(reader, data)
			if err != nil {
				return nil, fmt.Errorf("failed to read bulk string: %v", err)
			}

			crlf := make([]byte, 2)
			_, err = io.ReadFull(reader, crlf)
			if err != nil || crlf[0] != '\r' || crlf[1] != '\n' {
				return nil, fmt.Errorf("invalid CRLF terminator")
			}

			elements = append(elements, string(data))
		}
		return elements, nil

	case '+':
		return []string{line[1:]}, nil
	case '-':
		return nil, fmt.Errorf("redis error: %s", line[1:])
	case ':':
		return []string{line[1:]}, nil
	case '$':
		length, err := strconv.Atoi(line[1:])
		if err != nil {
			return nil, fmt.Errorf("invalid bulk string length: %v", err)
		}
		if length == -1 {
			return nil, nil
		}

		data := make([]byte, length)
		_, err = io.ReadFull(reader, data)
		if err != nil {
			return nil, fmt.Errorf("failed to read bulk string: %v", err)
		}

		crlf := make([]byte, 2)
		_, err = io.ReadFull(reader, crlf)
		if err != nil || crlf[0] != '\r' || crlf[1] != '\n' {
			return nil, fmt.Errorf("invalid CRLF terminator")
		}

		return []string{string(data)}, nil
	default:
		return nil, fmt.Errorf("unknown RESP type: %c", line[0])
	}
}

func (rc *RedisClient) AddPrice(ctx context.Context, exchange, symbol string, price float64) error {
	key := "price:" + exchange + ":" + symbol
	now := time.Now().Unix()

	_, err := rc.execCommand(ctx, "ZADD", key, fmt.Sprintf("%d", now), fmt.Sprintf("%f", price))
	if err != nil {
		return fmt.Errorf("failed to add price: %v", err)
	}

	_, err = rc.execCommand(ctx, "EXPIRE", key, "120")
	if err != nil {
		rc.logger.Warn("Failed to set TTL for key", "key", key, "error", err)
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		if err := rc.CleanOldPrices(ctx); err != nil {
			rc.logger.Warn("Background cleanup failed", "error", err)
		}
	}()

	return nil
}

func (rc *RedisClient) CleanOldPrices(ctx context.Context) error {
	resp, err := rc.execCommand(ctx, "KEYS", "price:*")
	if err != nil {
		return fmt.Errorf("failed to fetch keys for cleanup: %v", err)
	}

	expireBefore := time.Now().Add(-2 * time.Minute).Unix()

	for _, key := range resp {
		_, err := rc.execCommand(ctx, "ZREMRANGEBYSCORE", key, "-inf", fmt.Sprintf("%d", expireBefore))
		if err != nil {
			rc.logger.Warn("Failed to clean key", "key", key, "error", err)
		}
	}

	return nil
}

func (rc *RedisClient) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	resp, err := rc.execCommand(ctx, "PING")
	if err != nil {
		return fmt.Errorf("PING failed: %v", err)
	}

	if len(resp) == 0 || resp[0] != "PONG" {
		return fmt.Errorf("invalid PING response: %v", resp)
	}

	return nil
}
