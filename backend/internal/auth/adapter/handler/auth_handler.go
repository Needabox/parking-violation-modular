package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apperrors "github.com/parking-violation-portal/backend/pkg/errors"
	"github.com/parking-violation-portal/backend/internal/auth/domain"
	"github.com/parking-violation-portal/backend/internal/auth/usecase"
	"github.com/parking-violation-portal/backend/internal/gateway/middleware"
)

type AuthHandler struct {
	auth *usecase.AuthService
}

func NewAuthHandler(auth *usecase.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, apperrors.New(apperrors.CodeInvalidRequest, "invalid request body", http.StatusBadRequest))
		return
	}

	out, err := h.auth.Register(c.Request.Context(), usecase.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, out)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, apperrors.New(apperrors.CodeInvalidRequest, "invalid request body", http.StatusBadRequest))
		return
	}

	out, err := h.auth.Login(c.Request.Context(), usecase.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, out)
}

func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := middleware.UserFromContext(c)
	if !ok {
		writeError(c, apperrors.New(apperrors.CodeUnauthorized, "unauthorized", http.StatusUnauthorized))
		return
	}

	out, err := h.auth.Me(c.Request.Context(), user.UserID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, out)
}

type VehicleHandler struct {
	vehicles *usecase.VehicleService
}

func NewVehicleHandler(vehicles *usecase.VehicleService) *VehicleHandler {
	return &VehicleHandler{vehicles: vehicles}
}

type createVehicleRequest struct {
	PlateNumber string `json:"plate_number" binding:"required"`
}

func (h *VehicleHandler) Create(c *gin.Context) {
	user, ok := middleware.UserFromContext(c)
	if !ok {
		writeError(c, apperrors.New(apperrors.CodeUnauthorized, "unauthorized", http.StatusUnauthorized))
		return
	}
	if user.Role != domain.RoleMember {
		writeError(c, apperrors.New(apperrors.CodeForbidden, "forbidden", http.StatusForbidden))
		return
	}

	var req createVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, apperrors.New(apperrors.CodeInvalidRequest, "invalid request body", http.StatusBadRequest))
		return
	}

	out, err := h.vehicles.Create(c.Request.Context(), usecase.CreateVehicleInput{
		UserID:      user.UserID,
		PlateNumber: req.PlateNumber,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, out)
}

func (h *VehicleHandler) List(c *gin.Context) {
	user, ok := middleware.UserFromContext(c)
	if !ok {
		writeError(c, apperrors.New(apperrors.CodeUnauthorized, "unauthorized", http.StatusUnauthorized))
		return
	}
	if user.Role != domain.RoleMember {
		writeError(c, apperrors.New(apperrors.CodeForbidden, "forbidden", http.StatusForbidden))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	out, err := h.vehicles.List(c.Request.Context(), usecase.ListVehiclesInput{
		UserID:   user.UserID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, out)
}

func (h *VehicleHandler) Delete(c *gin.Context) {
	user, ok := middleware.UserFromContext(c)
	if !ok {
		writeError(c, apperrors.New(apperrors.CodeUnauthorized, "unauthorized", http.StatusUnauthorized))
		return
	}
	if user.Role != domain.RoleMember {
		writeError(c, apperrors.New(apperrors.CodeForbidden, "forbidden", http.StatusForbidden))
		return
	}

	vehicleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, apperrors.New(apperrors.CodeInvalidRequest, "invalid vehicle id", http.StatusBadRequest))
		return
	}

	if err := h.vehicles.Delete(c.Request.Context(), user.UserID, vehicleID); err != nil {
		writeError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    apperrors.Code         `json:"code"`
	Message string                 `json:"message"`
	Details []apperrors.FieldError `json:"details,omitempty"`
}

func writeError(c *gin.Context, err error) {
	appErr, ok := apperrors.IsAppError(err)
	if !ok {
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error: errorBody{
				Code:    apperrors.CodeInternal,
				Message: "internal server error",
			},
		})
		return
	}

	c.JSON(appErr.HTTPStatus, errorResponse{
		Error: errorBody{
			Code:    appErr.Code,
			Message: appErr.Message,
			Details: appErr.Details,
		},
	})
}
