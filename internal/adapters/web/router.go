package web

import (
	"database/sql"
	"net/http"

	"marketflow/internal/adapters/cache"
	"marketflow/internal/domain"

	_ "net/http"
)

func NewRouter(db *sql.DB, redisClient *cache.RedisClient, modeManager *domain.Manager) *http.ServeMux {
	mux := http.NewServeMux()
	handler := &Handler{
		DB:          db,
		RedisClient: redisClient,
		ModeManager: modeManager,
	}

	mux.HandleFunc("/mode/live", handler.SwitchToLiveMode)
	mux.HandleFunc("/mode/test", handler.SwitchToTestMode)

	mux.HandleFunc("GET /prices/latest/{symbol}", HandleLatest(redisClient, db))
	mux.HandleFunc("GET /prices/latest/{exchange}/{symbol}", HandleLatest(redisClient, db))

	mux.HandleFunc("/prices/highest/", HandleAggregatedValue(db, "MAX"))
	mux.HandleFunc("/prices/lowest/", HandleAggregatedValue(db, "MIN"))
	mux.HandleFunc("/prices/average/", HandleAggregatedValue(db, "AVG"))
	mux.HandleFunc("GET /health", HandleHealthCheck(db, redisClient, modeManager))

	return mux
}
