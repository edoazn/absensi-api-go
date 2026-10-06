package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edoazn/absensi-go/config"
	"github.com/edoazn/absensi-go/internal/database"
	"github.com/edoazn/absensi-go/internal/seeders"
	"github.com/edoazn/absensi-go/internal/server"
	"github.com/gin-gonic/gin"
)

// @title Absensi Mahasiswa API
// @version 1.0
// @description REST API sistem absensi mahasiswa berbasis geolocation. Semua respons memakai envelope {success, data, message, errors}.
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Format: "Bearer {access_token}"
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if err := database.EnsureDatabase(cfg); err != nil {
		log.Fatalf("ensure database: %v", err)
	}

	if err := database.Migrate(cfg); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Printf("migrations applied")

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}

	if cfg.SeedOnStart {
		if err := seeders.Seed(db, cfg.BcryptCost, cfg.Timezone); err != nil {
			log.Fatalf("seed: %v", err)
		}
		log.Printf("seeder done")
	}

	if cfg.IsProd() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := server.BuildRouter(server.Deps{Cfg: cfg, DB: db})

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		fmt.Printf("%s listening on :%s\n", cfg.AppName, cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
	log.Println("server stopped")
}
