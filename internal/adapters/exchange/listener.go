package exchange

import (
	"bufio"
	"context"
	"encoding/json"
	"log/slog"
	"math/rand"
	"net"
	"time"

	"marketflow/internal/domain"
)

type PriceMessage struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Timestamp int64   `json:"timestamp"`
}

func ListenToExchange(address, exchangeName string, out chan<- domain.PriceUpdate, logger *slog.Logger) {
	logger = logger.With("exchange", exchangeName)

	for {
		conn, err := net.Dial("tcp", address)
		if err != nil {
			logger.Error("Failed to connect", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		logger.Info("Connected to exchange", "address", address)

		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			var msg PriceMessage
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				logger.Warn("Failed to parse JSON message", "message", line, "error", err)
				continue
			}

			update := domain.PriceUpdate{
				Exchange:   exchangeName,
				Symbol:     msg.Symbol,
				Price:      msg.Price,
				ReceivedAt: time.Now(),
				Type:       "raw",
			}
			out <- update
		}

		if err := scanner.Err(); err != nil {
			logger.Error("Connection error", "error", err)
		}
		conn.Close()
		logger.Info("Connection closed, reconnecting...")
	}
}

var testPairs = []string{"BTCUSDT", "ETHUSDT", "DOGEUSDT", "TONUSDT", "SOLUSDT"}

func StartTestGenerators(ctx context.Context, out chan<- domain.PriceUpdate) {
	exchanges := []string{"binance", "coinbase", "kucoin"}
	for _, exchange := range exchanges {
		go generateTestData(ctx, exchange, out)
	}
}

func generateTestData(ctx context.Context, exchange string, out chan<- domain.PriceUpdate) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, pair := range testPairs {
				price := 30000 + rand.Float64()*20000 // BTC ~30k-50k

				update := domain.PriceUpdate{
					Exchange:   exchange,
					Symbol:     pair,
					Price:      price,
					ReceivedAt: time.Now(),
					Type:       "raw",
				}

				select {
				case out <- update:
				default:
					slog.Warn("Channel full, dropping test data")
				}
			}
		}
	}
}
