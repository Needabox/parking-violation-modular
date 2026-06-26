package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	authhandler "github.com/parking-violation-portal/backend/internal/auth/adapter/handler"
	jwtadapter "github.com/parking-violation-portal/backend/internal/auth/adapter/jwt"
	"github.com/parking-violation-portal/backend/internal/auth/adapter/postgres"
	"github.com/parking-violation-portal/backend/internal/auth/usecase"
	"github.com/parking-violation-portal/backend/internal/gateway"
	"github.com/parking-violation-portal/backend/pkg/config"
	pkgdb "github.com/parking-violation-portal/backend/pkg/db"
	jwtutil "github.com/parking-violation-portal/backend/pkg/jwt"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pkgdb.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	jwtManager := jwtutil.NewManager(cfg.JWTSecret, cfg.JWTExpiration)
	userRepo := postgres.NewUserRepository(pool)
	vehicleRepo := postgres.NewVehicleRepository(pool)
	tokenGen := jwtadapter.NewTokenGenerator(jwtManager)
	hasher := usecase.NewBcryptHasher()

	authService := usecase.NewAuthService(userRepo, hasher, tokenGen)
	vehicleService := usecase.NewVehicleService(vehicleRepo)

	router := gateway.NewRouter(gateway.Dependencies{
		AuthHandler:    authhandler.NewAuthHandler(authService),
		VehicleHandler: authhandler.NewVehicleHandler(vehicleService),
		JWTManager:     jwtManager,
		AllowedOrigins: cfg.AllowedOrigins,
		DBPool:         pool,
	})

	server := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("server listening on :%s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
