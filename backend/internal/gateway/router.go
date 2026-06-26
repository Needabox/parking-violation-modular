package gateway

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	authhandler "github.com/parking-violation-portal/backend/internal/auth/adapter/handler"
	"github.com/parking-violation-portal/backend/internal/gateway/middleware"
	jwtutil "github.com/parking-violation-portal/backend/pkg/jwt"
	pkgdb "github.com/parking-violation-portal/backend/pkg/db"
)

type Dependencies struct {
	AuthHandler    *authhandler.AuthHandler
	VehicleHandler *authhandler.VehicleHandler
	JWTManager     *jwtutil.Manager
	AllowedOrigins []string
	DBPool         *pgxpool.Pool
}

func NewRouter(deps Dependencies) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS(deps.AllowedOrigins))

	router.GET("/health", func(c *gin.Context) {
		dbStatus := "ok"
		if err := pkgdb.HealthCheck(context.Background(), deps.DBPool); err != nil {
			dbStatus = "down"
		}

		status := http.StatusOK
		body := gin.H{
			"status":   "ok",
			"database": dbStatus,
		}
		if dbStatus != "ok" {
			status = http.StatusServiceUnavailable
			body["status"] = "degraded"
		}

		c.JSON(status, body)
	})

	api := router.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", deps.AuthHandler.Register)
			auth.POST("/login", deps.AuthHandler.Login)
			auth.GET("/me", middleware.Auth(deps.JWTManager), deps.AuthHandler.Me)
		}

		vehicles := api.Group("/vehicles")
		vehicles.Use(middleware.Auth(deps.JWTManager))
		{
			vehicles.POST("", deps.VehicleHandler.Create)
			vehicles.GET("", deps.VehicleHandler.List)
			vehicles.DELETE("/:id", deps.VehicleHandler.Delete)
		}
	}

	return router
}
