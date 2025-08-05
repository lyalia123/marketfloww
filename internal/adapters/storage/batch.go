package storage

import (
	"database/sql"
	"log/slog"
	"time"

	"marketflow/internal/domain"
)

func SaveBatchToPostgres(toPG <-chan domain.PriceUpdate, db *sql.DB, logger *slog.Logger) {
	const batchSize = 100
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var rawBatch []domain.PriceUpdate
	var aggBatch []domain.PriceUpdate

	for {
		select {
		case update := <-toPG:
			switch update.Type {
			case "raw", "":
				rawBatch = append(rawBatch, update)
			case "aggregated", "min", "max", "avg":
				aggBatch = append(aggBatch, update)
			}

			if len(rawBatch) >= batchSize {
				insertRawBatch(db, rawBatch, logger)
				rawBatch = nil
			}
			if len(aggBatch) >= batchSize {
				insertAggBatch(db, aggBatch, logger)
				aggBatch = nil
			}

		case <-ticker.C:
			if len(rawBatch) > 0 {
				insertRawBatch(db, rawBatch, logger)
				rawBatch = nil
			}
			if len(aggBatch) > 0 {
				insertAggBatch(db, aggBatch, logger)
				aggBatch = nil
			}
		}
	}
}
