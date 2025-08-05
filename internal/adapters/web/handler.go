package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"marketflow/internal/adapters/cache"
	"marketflow/internal/domain"
)

type Handler struct {
	ModeManager *domain.Manager
	DB          *sql.DB
	RedisClient *cache.RedisClient
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewHandler(mm *domain.Manager, db *sql.DB, rc *cache.RedisClient) *Handler {
	return &Handler{
		ModeManager: mm,
		DB:          db,
		RedisClient: rc,
	}
}

func HandleLatest(redisClient *cache.RedisClient, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 100*time.Millisecond) // Reduced timeout
		defer cancel()

		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/prices/latest/"), "/")
		var exchange, symbol string

		if len(parts) == 1 {
			symbol = parts[0]
			price, err := redisClient.GetLatestPrice(ctx, "", symbol)
			if err != nil {
				// Use a separate context with longer timeout for PostgreSQL fallback
				pgCtx, pgCancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
				defer pgCancel()

				row := db.QueryRowContext(pgCtx,
					"SELECT exchange, price FROM price_raw WHERE symbol = $1 ORDER BY timestamp DESC LIMIT 1",
					symbol)

				var lastPrice float64
				if err := row.Scan(&exchange, &lastPrice); err != nil {
					if err == sql.ErrNoRows {
						http.Error(w, "price not available", http.StatusNotFound)
						return
					}
					http.Error(w, "database error", http.StatusInternalServerError)
					return
				}
				price = lastPrice
			}

			response := map[string]interface{}{
				"symbol":    symbol,
				"exchange":  exchange,
				"price":     price,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"source":    "redis",
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(response); err != nil {
				log.Printf("Failed to encode response: %v", err)
			}
			return
		} else if len(parts) == 2 {
			exchange = parts[0]
			symbol = parts[1]
		} else {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}

		price, err := redisClient.GetLatestPrice(ctx, exchange, symbol)
		if err != nil {
			// Use a separate context with longer timeout for PostgreSQL fallback
			pgCtx, pgCancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
			defer pgCancel()

			row := db.QueryRowContext(pgCtx,
				"SELECT price FROM price_raw WHERE exchange = $1 AND symbol = $2 ORDER BY timestamp DESC LIMIT 1",
				exchange, symbol)

			var lastPrice float64
			if err := row.Scan(&lastPrice); err != nil {
				if err == sql.ErrNoRows {
					http.Error(w, "price not available", http.StatusNotFound)
				} else {
					http.Error(w, "database error", http.StatusInternalServerError)
				}
				return
			}
			price = lastPrice
		}

		response := map[string]interface{}{
			"symbol":    symbol,
			"exchange":  exchange,
			"price":     price,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"source":    "redis",
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Failed to encode response: %v", err)
		}
	}
}

func queryAggregatedValue(db *sql.DB, fn string, pair, exchange string, duration string) (float64, string, error) {
	baseQuery := fmt.Sprintf("SELECT %s(average_price), exchange FROM aggregated_prices WHERE symbol = $1", fn)
	args := []interface{}{pair}
	argIdx := 2

	if exchange != "" {
		baseQuery += fmt.Sprintf(" AND exchange = $%d", argIdx)
		args = append(args, exchange)
		argIdx++
	}

	if duration != "" {
		baseQuery += fmt.Sprintf(" AND timestamp > now() - $%d::interval", argIdx)
		args = append(args, duration)
	}

	baseQuery += " GROUP BY exchange"

	var result float64
	var exch string
	err := db.QueryRow(baseQuery, args...).Scan(&result, &exch)
	return result, exch, err
}

func HandleAggregatedValue(db *sql.DB, aggType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		defer func() {
			slog.Info("Request processed",
				"path", r.URL.Path,
				"duration_ms", time.Since(startTime).Milliseconds())
		}()

		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		path := r.URL.Path
		var prefix string

		switch aggType {
		case "MAX":
			prefix = "/prices/highest/"
		case "MIN":
			prefix = "/prices/lowest/"
		case "AVG":
			prefix = "/prices/average/"
		default:
			http.Error(w, "invalid aggregation type", http.StatusBadRequest)
			return
		}

		path = strings.TrimPrefix(path, prefix)
		parts := strings.Split(path, "/")
		if len(parts) < 1 || parts[0] == "" {
			http.Error(w, "missing symbol", http.StatusBadRequest)
			return
		}

		var symbol, exchange string
		if len(parts) == 1 {
			symbol = parts[0]
		} else if len(parts) == 2 {
			exchange = parts[0]
			symbol = parts[1]
		} else {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}

		// Валидация symbol и exchange
		validSymbols := map[string]bool{"BTCUSDT": true, "ETHUSDT": true, "DOGEUSDT": true, "SOLUSDT": true, "TONUSDT": true}
		if !validSymbols[symbol] {
			http.Error(w, "invalid symbol", http.StatusBadRequest)
			return
		}

		if exchange != "" {
			validExchanges := map[string]bool{"binance": true, "coinbase": true, "kucoin": true}
			if !validExchanges[exchange] {
				http.Error(w, "invalid exchange", http.StatusBadRequest)
				return
			}
		}

		duration := r.URL.Query().Get("period")
		if duration != "" {
			if _, err := time.ParseDuration(duration); err != nil {
				http.Error(w, "invalid period format", http.StatusBadRequest)
				return
			}
		}

		result, exch, err := queryAggregatedValue(db, aggType, symbol, exchange, duration)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "no data available", http.StatusNotFound)
			} else {
				slog.Error("Database query failed", "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
			return
		}

		response := map[string]interface{}{
			"symbol":   symbol,
			"exchange": exch,
			"period":   duration,
		}

		switch aggType {
		case "AVG":
			response["average"] = result
		case "MAX":
			response["max"] = result
		case "MIN":
			response["min"] = result
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			slog.Error("Failed to encode response", "error", err)
		}
	}
}

type MessageResponse struct {
	Message string `json:"message"`
}

func (h *Handler) SwitchToTestMode(w http.ResponseWriter, r *http.Request) {
	if err := h.ModeManager.SetMode(r.Context(), domain.ModeTest); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSONResponse(w, http.StatusOK, MessageResponse{Message: "Switched to Test Mode"})
}

func (h *Handler) SwitchToLiveMode(w http.ResponseWriter, r *http.Request) {
	if err := h.ModeManager.SetMode(r.Context(), domain.ModeLive); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSONResponse(w, http.StatusOK, MessageResponse{Message: "Switched to Live Mode"})
}

func writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}

func HealthHandler(db *sql.DB, redis *cache.RedisClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := map[string]string{}

		if err := db.Ping(); err != nil {
			status["postgres"] = "unhealthy"
		} else {
			status["postgres"] = "healthy"
		}

		// if err := cache.Ping(); err != nil {
		// 	status["redis"] = "unhealthy"
		// } else {
		// 	status["redis"] = "healthy"
		// }

		status["workers"] = "running" // можно доработать позже

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	}
}

func HandleHealthCheck(db *sql.DB, rc *cache.RedisClient, modeManager *domain.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := map[string]string{
			"postgres": "ok",
			"redis":    "ok",
			"workers":  "running",
			"mode":     modeManager.GetMode().String(),
		}

		if err := db.Ping(); err != nil {
			status["postgres"] = "down"
		}
		if err := rc.Ping(); err != nil {
			status["redis"] = "down"
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(status); err != nil {
			log.Printf("Failed to encode health check response: %v", err)
		}
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	slog.Warn("Returning error", "status", status, "message", message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}
