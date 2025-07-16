package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/alhamdibahri/my-echo-app/docs"
	"github.com/alhamdibahri/my-echo-app/internal/api"
	"github.com/alhamdibahri/my-echo-app/internal/config"
	"github.com/alhamdibahri/my-echo-app/internal/db"
	"github.com/alhamdibahri/my-echo-app/internal/rabbitmq"
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// @title Multi-Tenant Messaging API
// @version 1.0
// @description Multi-Tenant Messaging system with RabbitMQ and Postgres
// @alhamdibahri Developer
// @contact.email alhamdiferdiawanbahri@gmail.com
// @host localhost:8080
// @BasePath /
func main() {
	// Load config
	cfg, err := config.LoadConfig("./config/config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Connect to Postgres
	pg, err := db.NewPostgres(cfg.Database.URL)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pg.Close()

	// Connect to RabbitMQ
	rmq, err := rabbitmq.NewManager(cfg.RabbitMQ.URL, pg)
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}
	defer rmq.Close()

	// Echo setup
	e := echo.New()

	// Register API routes
	api.RegisterRoutes(e, rmq, pg, cfg)

	// Swagger docs route
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// Start server
	go func() {
		if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down")
		}
	}()
	log.Println("[Server] Started on :8080")

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[Server] Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}
