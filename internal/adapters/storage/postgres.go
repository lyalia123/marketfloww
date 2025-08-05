package storage

import (
	"database/sql"
	"log"
	"log/slog"
	"strconv"
	"time"

	"marketflow/internal/domain"
)

type PostgresClient struct {
	DB *sql.DB
}

func NewPostgresClient(dsn string) (*PostgresClient, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	return &PostgresClient{DB: db}, nil
}

func insertAggBatch(db *sql.DB, batch []domain.PriceUpdate, logger *slog.Logger) {
	if len(batch) == 0 {
		return
	}

	tx, err := db.Begin()
	if err != nil {
		logger.Error("Failed to begin transaction", "error", err)
		return
	}

	stmt, err := tx.Prepare(`
		INSERT INTO aggregated_prices (symbol, exchange, timestamp, average_price, min_price, max_price)
		VALUES ($1, $2, $3, $4, $5, $6)
	`)
	if err != nil {
		logger.Error("Failed to prepare statement", "error", err)
		_ = tx.Rollback()
		return
	}
	defer stmt.Close()

	for _, update := range batch {
		_, err := stmt.Exec(update.Symbol, update.Exchange, update.ReceivedAt, update.AvgPrice, update.MinPrice, update.MaxPrice)
		if err != nil {
			logger.Error("Insert failed", "error", err)
			_ = tx.Rollback()
			return
		}
	}

	if err := tx.Commit(); err != nil {
		logger.Error("Failed to commit transaction", "error", err)
	} else {
		logger.Debug("Inserted aggregated batch", "count", len(batch))
	}
}

func insertRawBatch(db *sql.DB, batch []domain.PriceUpdate, logger *slog.Logger) {
	if len(batch) == 0 {
		return
	}

	tx, err := db.Begin()
	if err != nil {
		logger.Error("Failed to begin transaction", "error", err)
		return
	}
	defer tx.Commit()

	stmt, err := tx.Prepare(`
		INSERT INTO price_raw (symbol, exchange, price, timestamp)
		VALUES ($1, $2, $3, $4)
	`)
	if err != nil {
		logger.Error("Failed to prepare statement", "error", err)
		return
	}
	defer stmt.Close()

	for _, update := range batch {
		_, err := stmt.Exec(update.Symbol, update.Exchange, update.Price, update.ReceivedAt)
		if err != nil {
			logger.Error("Insert failed", "error", err)
		}
	}

	logger.Debug("Inserted raw batch", "count", len(batch))
}

func (pc *PostgresClient) GetHighest(symbol string, exchange *string, duration *time.Duration) (*domain.PriceUpdate, error) {
	query := `
	SELECT exchange, symbol, timestamp, max_price  
	FROM aggregated_prices
	WHERE symbol = $1
	`
	args := []interface{}{symbol}
	argIndex := 2

	if exchange != nil {
		query += ` AND exchange = $` + strconv.Itoa(argIndex)
		args = append(args, *exchange)
		argIndex++
	}
	if duration != nil {
		query += ` AND timestamp >= NOW() - INTERVAL '` + duration.String() + `'`
	}
	query += ` ORDER BY max_price DESC LIMIT 1`

	row := pc.DB.QueryRow(query, args)

	var result domain.PriceUpdate
	var ts time.Time
	var maxPrice float64

	err := row.Scan(&result.Exchange, &result.Symbol, &ts, &maxPrice)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Printf(" Failed to scan highest price: %v", err)
		return nil, err
	}
	result.ReceivedAt = ts
	result.MaxPrice = maxPrice
	result.Type = "max"
	return &result, nil
}
