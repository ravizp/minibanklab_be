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

	"account-service/config"
	_ "account-service/docs"
	"account-service/internal/handler"
	"account-service/internal/messaging"
	"account-service/internal/middleware"
	"account-service/internal/repository"
	"account-service/internal/service"
	"account-service/migrations"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
    gintrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gin-gonic/gin"
    gormtrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gorm.io/gorm.io/gorm.v2"
)

// @title           Minibank Account Service API
// @version         1.0
// @description     Bank account management service for Minibank
// @host            localhost:8082
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your token with the Bearer prefix, e.g. "Bearer eyJhbGciOiJIUzI1NiIs..."
func main() {
	cfg := config.Load()

	// --- TAMBAHKAN INI: Start Tracer ---
    // Datadog akan otomatis mengambil konfig dari env var DD_AGENT_HOST
    tracer.Start(
        tracer.WithService("account-service"),
        tracer.WithEnv("development"),
		tracer.WithRuntimeMetrics(), // Opsional: untuk melihat performa Go Garbage Collector
    )
    defer tracer.Stop()
    // -----------------------------------------

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

	// RabbitMQ consumer for transaction events
	var mq *messaging.RabbitMQ
	if cfg.RabbitMQURL != "" {
		var err error
		mq, err = messaging.NewRabbitMQ(cfg.RabbitMQURL)
		if err != nil {
			log.Printf("Warning: RabbitMQ connection failed: %v", err)
		} else {
			if err := mq.Consume(
				"account-service.transaction.created",
				"transaction.created",
				func(body []byte) error {
					log.Printf("[Consumer] Received transaction event: %s", string(body))
					return nil
				},
			); err != nil {
				log.Printf("Warning: Failed to start consumer: %v", err)
			}
		}
	}

	accountRepo := repository.NewAccountRepository(db)
	accountSvc := service.NewAccountService(accountRepo)
	accountHandler := handler.NewAccountHandler(accountSvc)

	r := gin.Default()

	// --- TAMBAHKAN INI: Middleware Tracing ---
    r.Use(gintrace.Middleware("account-service"))
    // -----------------------------------------
	r.GET("/health", handler.HealthCheck)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	{
		api.GET("/health", handler.HealthCheck)

		accounts := api.Group("/accounts", middleware.JWTAuth(cfg.JWTSecret))
		{
			accounts.POST("", accountHandler.CreateAccount)
			accounts.GET("", accountHandler.GetMyAccounts)
			accounts.GET("/:id", accountHandler.GetAccount)
			accounts.GET("/:id/balance", accountHandler.GetBalance)
		}
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: r,
	}

	go func() {
		log.Printf("Account service starting on port %s", cfg.Port)
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

const migrationsTable = "schema_migrations_accounts"

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
			// --- TAMBAHKAN INI: Tracing untuk Database ---
            if err := db.Use(gormtrace.NewPlugin(gormtrace.WithServiceName("account-service-db"))); err != nil {
                log.Printf("failed to use gormtrace plugin: %v", err)
            }
            // ----------------------------------------------
			sqlDB, _ := db.DB()
			if sqlDB.Ping() == nil {
				log.Println("Connected to database successfully")
				sqlDB.SetMaxOpenConns(25)
				sqlDB.SetMaxIdleConns(5)
				sqlDB.SetConnMaxLifetime(5 * time.Minute)x
				return db
			}
		}
		log.Printf("Waiting for database connection... attempt %d/30", i+1)
		time.Sleep(2 * time.Second)
	}

	log.Fatalf("Failed to connect to database after 30 attempts: %v", err)
	return nil
}
