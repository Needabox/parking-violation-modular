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
	authpostgres "github.com/parking-violation-portal/backend/internal/auth/adapter/postgres"
	"github.com/parking-violation-portal/backend/internal/auth/usecase"
	"github.com/parking-violation-portal/backend/internal/gateway"
	rulehandler "github.com/parking-violation-portal/backend/internal/rules/adapter/handler"
	rulepostgres "github.com/parking-violation-portal/backend/internal/rules/adapter/postgres"
	ruleusecase "github.com/parking-violation-portal/backend/internal/rules/usecase"
	violationhandler "github.com/parking-violation-portal/backend/internal/violation/adapter/handler"
	violationpostgres "github.com/parking-violation-portal/backend/internal/violation/adapter/postgres"
	violationusecase "github.com/parking-violation-portal/backend/internal/violation/usecase"
	paymenthandler "github.com/parking-violation-portal/backend/internal/payment/adapter/handler"
	paymentpostgres "github.com/parking-violation-portal/backend/internal/payment/adapter/postgres"
	paymentusecase "github.com/parking-violation-portal/backend/internal/payment/usecase"
	paymentport "github.com/parking-violation-portal/backend/internal/payment/port"
	"github.com/google/uuid"
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
	userRepo := authpostgres.NewUserRepository(pool)
	vehicleRepo := authpostgres.NewVehicleRepository(pool)
	tokenGen := jwtadapter.NewTokenGenerator(jwtManager)
	hasher := usecase.NewBcryptHasher()

	authService := usecase.NewAuthService(userRepo, hasher, tokenGen)
	vehicleService := usecase.NewVehicleService(vehicleRepo)

	ruleRepo := rulepostgres.NewRuleRepository(pool)
	ruleService := ruleusecase.NewRuleService(ruleRepo)

	violationRepo := violationpostgres.NewViolationRepository(pool)
	violationService := violationusecase.NewViolationService(violationRepo, vehicleRepo, ruleRepo)

	violationPayment := &paymentViolationAdapter{violationRepo: violationRepo}
	stubTxRec := &stubTransactionRecorder{}

	paymentRepo := paymentpostgres.NewPaymentRepository(pool)
	paymentService := paymentusecase.NewPaymentService(paymentRepo, violationPayment, vehicleRepo, stubTxRec)

	router := gateway.NewRouter(gateway.Dependencies{
		AuthHandler:      authhandler.NewAuthHandler(authService),
		VehicleHandler:   authhandler.NewVehicleHandler(vehicleService),
		RuleHandler:      rulehandler.NewRuleHandler(ruleService),
		ViolationHandler: violationhandler.NewViolationHandler(violationService),
		PaymentHandler:   paymenthandler.NewPaymentHandler(paymentService),
		JWTManager:       jwtManager,
		AllowedOrigins:   cfg.AllowedOrigins,
		DBPool:           pool,
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

type paymentViolationAdapter struct {
	violationRepo *violationpostgres.ViolationRepository
}

func (a *paymentViolationAdapter) GetViolationForPayment(ctx context.Context, id uuid.UUID) (paymentport.ViolationDetail, error) {
	detail, err := a.violationRepo.GetViolationByID(ctx, id)
	if err != nil {
		return paymentport.ViolationDetail{}, err
	}
	return paymentport.ViolationDetail{
		ID:          detail.ID,
		PlateNumber: detail.PlateNumber,
		FineAmount:  detail.FineAmount,
		Status:      detail.Status,
		FineRuleID:  detail.FineRuleID,
	}, nil
}

func (a *paymentViolationAdapter) UpdateViolationStatusToPaid(ctx context.Context, id uuid.UUID) error {
	_, err := a.violationRepo.UpdateViolationStatus(ctx, id, "PAID")
	return err
}

type stubTransactionRecorder struct{}

func (stubTransactionRecorder) RecordTransaction(ctx context.Context, violationID uuid.UUID, amount float64, ruleID uuid.UUID) error {
	log.Printf("[STUB] RecordTransaction: violation %s, amount %f, rule %s", violationID, amount, ruleID)
	return nil
}
