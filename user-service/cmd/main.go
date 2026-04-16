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

	"user-service/config"
	_ "user-service/docs"
	"user-service/internal/handler"
	"user-service/internal/middleware"
	"user-service/internal/repository"
	"user-service/internal/service"
	"user-service/migrations"

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

// @title           Minibank User Service API
// @version         1.0
// @description     User authentication and management service for Minibank
// @host            localhost:8081
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your token with the Bearer prefix, e.g. "Bearer eyJhbGciOiJIUzI1NiIs..."
func main() {
	cfg := config.Load()

	// --- 1. Inisialisasi Tracer Datadog ---
    tracer.Start(
        tracer.WithService("user-service"),
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

	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo)
	authHandler := handler.NewAuthHandler(authSvc, cfg.JWTSecret)

	r := gin.Default()

	// --- 2. Tambahkan Middleware Tracing ke Gin ---
    // Ini akan merekam semua request ke /register, /login, dll.
    r.Use(gintrace.Middleware("user-service"))
    // -----------------------------------------
	r.GET("/health", handler.HealthCheck)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	{
		api.GET("/health", handler.HealthCheck)

		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.GET("/profile", middleware.JWTAuth(cfg.JWTSecret), authHandler.Profile)
			auth.GET("/profile/detail", middleware.JWTAuth(cfg.JWTSecret), authHandler.ProfileDetail)
		}
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: r,
	}

	go func() {
		log.Printf("User service starting on port %s", cfg.Port)
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
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("Server exited gracefully")
}

const migrationsTable = "schema_migrations_users"

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
			// --- 3. Tambahkan Plugin Tracing ke GORM ---// --- 3. Tambahkan Plugin Tracing ke GORM ---
            // Memantau query ke tabel users
            if err := db.Use(gormtrace.NewPlugin(gormtrace.WithServiceName("user-db"))); err != nil {
                log.Printf("failed to setup gorm tracing: %v", err)
            }
            // ----------------------------------------------
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
