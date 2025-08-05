package worker

import (
	"log/slog"
	"sync"
	"time"

	"marketflow/internal/adapters/cache"
	"marketflow/internal/domain"
)

func startWorkerPool(
	exchangeName string,
	in <-chan domain.PriceUpdate,
	toRedis chan<- domain.PriceUpdate,
	toPG chan<- domain.PriceUpdate,
	redisClient *cache.RedisClient,
	logger *slog.Logger,
) {
	type symbolStats struct {
		prices     []float64
		lastUpdate domain.PriceUpdate
		mu         sync.Mutex
	}

	stats := make(map[string]*symbolStats)
	var statsMu sync.Mutex

	// Увеличиваем количество воркеров до 10
	for i := 0; i < 10; i++ {
		go func(workerID int) {
			for update := range in {
				if update.Exchange != exchangeName {
					continue
				}

				// Неблокирующая отправка в Redis
				select {
				case toRedis <- update:
				default:
					logger.Warn("Redis channel full, dropping update",
						"worker", workerID,
						"symbol", update.Symbol)
				}

				// Неблокирующая отправка в PostgreSQL
				select {
				case toPG <- update:
				default:
					logger.Warn("PG channel full, dropping update",
						"worker", workerID,
						"symbol", update.Symbol)
				}

				// Агрегация данных
				statsMu.Lock()
				stat, ok := stats[update.Symbol]
				if !ok {
					stat = &symbolStats{}
					stats[update.Symbol] = stat
				}
				stat.mu.Lock()
				stat.prices = append(stat.prices, update.Price)
				stat.lastUpdate = update

				// Агрегируем каждые 5 сообщений
				if len(stat.prices) >= 5 {
					aggregateAndSend(stat.prices, stat.lastUpdate, toPG, logger, workerID)
					stat.prices = nil
				}
				stat.mu.Unlock()
				statsMu.Unlock()
			}
		}(i)
	}

	// Фоновая горутина для агрегации оставшихся данных
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond) // Уменьшаем интервал
		defer ticker.Stop()

		for range ticker.C {
			statsMu.Lock()
			for _, stat := range stats {
				stat.mu.Lock()
				if len(stat.prices) > 0 {
					aggregateAndSend(stat.prices, stat.lastUpdate, toPG, logger, -1)
					stat.prices = nil
				}
				stat.mu.Unlock()
			}
			statsMu.Unlock()
		}
	}()
}

func aggregateAndSend(prices []float64, update domain.PriceUpdate, toPG chan<- domain.PriceUpdate, logger *slog.Logger, workerID int) {
	if len(prices) == 0 {
		return
	}

	min, max, sum := prices[0], prices[0], 0.0
	for _, p := range prices {
		if p < min {
			min = p
		}
		if p > max {
			max = p
		}
		sum += p
	}
	avg := sum / float64(len(prices))

	agg := domain.PriceUpdate{
		Exchange:   update.Exchange,
		Symbol:     update.Symbol,
		ReceivedAt: time.Now().UTC(),
		Type:       "aggregated",
		AvgPrice:   avg,
		MinPrice:   min,
		MaxPrice:   max,
	}

	if workerID >= 0 {
		logger.Debug("AGGREGATE READY",
			"worker", workerID,
			"exchange", agg.Exchange,
			"symbol", agg.Symbol,
			"avg", avg)
	}

	select {
	case toPG <- agg:
	default:
		logger.Warn("PG channel full, dropping aggregate")
	}
}
