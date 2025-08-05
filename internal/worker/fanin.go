package worker

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"marketflow/internal/adapters/cache"
	"marketflow/internal/adapters/exchange"
	"marketflow/internal/adapters/storage"
	"marketflow/internal/domain"
)

func StartIngestion(logger *slog.Logger, redisClient *cache.RedisClient, db *sql.DB, toPG chan domain.PriceUpdate, modeManager *domain.Manager) {
	fanIn := make(chan domain.PriceUpdate, 10000)
	toRedis := make(chan domain.PriceUpdate, 10000)

	// Start mode manager
	go func() {
		for {
			mode := modeManager.GetMode()
			logger.Info("Current mode", "mode", mode)

			switch mode {
			case domain.ModeLive:
				logger.Info("Starting live mode listeners")
				go exchange.ListenToExchange("exchange1:40101", "binance", fanIn, logger)
				go exchange.ListenToExchange("exchange2:40102", "coinbase", fanIn, logger)
				go exchange.ListenToExchange("exchange3:40103", "kucoin", fanIn, logger)
			case domain.ModeTest:
				logger.Info("Starting test mode generators")
				ctx := context.Background()
				go exchange.StartTestGenerators(ctx, fanIn)
			}

			time.Sleep(10 * time.Second) // Check mode less frequently
		}
	}()

	// Start Redis workers
	for i := 0; i < 20; i++ { // Increased number of workers
		go redisWorker(i, toRedis, redisClient, logger)
	}

	// Start processing workers
	startWorkerPool("binance", fanIn, toRedis, toPG, redisClient, logger)
	startWorkerPool("coinbase", fanIn, toRedis, toPG, redisClient, logger)
	startWorkerPool("kucoin", fanIn, toRedis, toPG, redisClient, logger)

	// Start PostgreSQL saver
	go storage.SaveBatchToPostgres(toPG, db, logger)
}

func redisWorker(workerID int, toRedis <-chan domain.PriceUpdate, redisClient *cache.RedisClient, logger *slog.Logger) {
	for update := range toRedis {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		err := redisClient.AddPrice(ctx, update.Exchange, update.Symbol, update.Price)
		cancel()

		if err != nil {
			logger.Error("Failed to save price to Redis",
				"worker", workerID,
				"exchange", update.Exchange,
				"symbol", update.Symbol,
				"error", err)

			// Implement simple backoff
			time.Sleep(100 * time.Millisecond)
		}
	}
}
