package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"marketflow/internal/adapters/cache"
	"marketflow/internal/adapters/storage"
	"marketflow/internal/adapters/web"
	"marketflow/internal/config"
	"marketflow/internal/domain"
	"marketflow/internal/worker"
	"marketflow/pkg/logger"

	_ "github.com/lib/pq"
)

func main() {
	logger, cleanup := logger.SetupLogger()
	defer cleanup()

	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		logger.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	logger.Info("Starting application", "mode", cfg.Mode)

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Postgres.Host,
		cfg.Postgres.Port,
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.DBName,
		cfg.Postgres.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logger.Error("Failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Wait for PostgreSQL
	for i := 0; i < 10; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		logger.Info("Waiting for PostgreSQL...", "attempt", i+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		logger.Error("PostgreSQL is not responding", "error", err)
		os.Exit(1)
	}

	if err := storage.InitDB(db); err != nil {
		logger.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)

	poolSize := 50
	if ps := os.Getenv("REDIS_POOL_SIZE"); ps != "" {
		if n, err := strconv.Atoi(ps); err == nil {
			poolSize = n
		}
	}

	redisClient, err := cache.NewRedisClient(redisAddr, logger, poolSize) // Increased pool size
	if err != nil {
		logger.Error("Failed to connect to Redis", "error", err)
		os.Exit(1)
	}

	toPG := make(chan domain.PriceUpdate, 20000) // Increased buffer size
	modeManager := domain.NewModeManager()

	go worker.StartIngestion(logger, redisClient, db, toPG, modeManager)

	go worker.StartIngestion(logger, redisClient, db, toPG, modeManager)

	router := web.NewRouter(db, redisClient, modeManager)
	server := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("Starting server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	close(toPG)

	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", "error", err)
	}
	logger.Info("Server stopped")
}
