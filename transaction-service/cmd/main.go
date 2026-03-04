package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"transaction-service/config"
	_ "transaction-service/docs"
	"transaction-service/internal/handler"
	"transaction-service/internal/messaging"
	"transaction-service/internal/middleware"
	"transaction-service/internal/repository"
	"transaction-service/internal/service"
	"transaction-service/migrations"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// @title           Minibank Transaction Service API
// @version         1.0
// @description     Transaction management service for Minibank
// @host            localhost:8083
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your token with the Bearer prefix, e.g. "Bearer eyJhbGciOiJIUzI1NiIs..."
func main() {
	cfg := config.Load()

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		direction := "up"
		if len(os.Args) > 2 {
			direction = os.Args[2]
		}
		if err := runMigration(cfg.DatabaseURL, direction); err != nil {
			log.Fatalf("migration failed: %v", err)
		}
		log.Printf("Migration %s completed successfully", direction)
		return
	}

	if err := runMigration(cfg.DatabaseURL, "up"); err != nil {
		log.Fatalf("auto-migration failed: %v", err)
	}
	log.Println("Database migrations applied successfully")

	db := connectDB(cfg.DatabaseURL)

	// RabbitMQ publisher
	var mq *messaging.RabbitMQ
	if cfg.RabbitMQURL != "" {
		var err error
		mq, err = messaging.NewRabbitMQ(cfg.RabbitMQURL)
		if err != nil {
			log.Printf("Warning: RabbitMQ connection failed: %v (events will not be published)", err)
		}
	}

	txRepo := repository.NewTransactionRepository(db)
	txSvc := service.NewTransactionService(txRepo, mq)
	txHandler := handler.NewTransactionHandler(txSvc)

	r := gin.Default()

	r.GET("/health", handler.HealthCheck)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	{
		api.GET("/health", handler.HealthCheck)

		transactions := api.Group("/transactions", middleware.JWTAuth(cfg.JWTSecret))
		{
			transactions.POST("/transfer", txHandler.Transfer)
			transactions.POST("/topup", txHandler.TopUp)
			transactions.GET("/history/:account_id", txHandler.GetHistory)
		}
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: r,
	}

	go func() {
		log.Printf("Transaction service starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if mq != nil {
		mq.Close()
	}
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("Server exited gracefully")
}

const migrationsTable = "schema_migrations_transactions"

func appendMigrationsTable(databaseURL, table string) string {
	if strings.Contains(databaseURL, "?") {
		return databaseURL + "&x-migrations-table=" + table
	}
	return databaseURL + "?x-migrations-table=" + table
}

func runMigration(databaseURL, direction string) error {
	dsn := appendMigrationsTable(databaseURL, migrationsTable)

	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, dsn)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	switch direction {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
	case "down":
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
	default:
		return fmt.Errorf("unknown migration direction: %s (use 'up' or 'down')", direction)
	}
	return nil
}

func connectDB(dsn string) *gorm.DB {
	var db *gorm.DB
	var err error

	for i := 0; i < 30; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			sqlDB, _ := db.DB()
			if sqlDB.Ping() == nil {
				log.Println("Connected to database successfully")
				sqlDB.SetMaxOpenConns(25)
				sqlDB.SetMaxIdleConns(5)
				sqlDB.SetConnMaxLifetime(5 * time.Minute)
				return db
			}
		}
		log.Printf("Waiting for database connection... attempt %d/30", i+1)
		time.Sleep(2 * time.Second)
	}

	log.Fatalf("Failed to connect to database after 30 attempts: %v", err)
	return nil
}
