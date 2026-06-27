package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authdomain "github.com/parking-violation-portal/backend/internal/auth/domain"
	"github.com/parking-violation-portal/backend/internal/gateway/middleware"
	"github.com/parking-violation-portal/backend/internal/violation/usecase"
	apperrors "github.com/parking-violation-portal/backend/pkg/errors"
)

type ViolationHandler struct {
	violations *usecase.ViolationService
}

func NewViolationHandler(violations *usecase.ViolationService) *ViolationHandler {
	return &ViolationHandler{violations: violations}
}

type reportRequest struct {
	RuleCode    string `json:"rule_code" binding:"required"`
	PlateNumber string `json:"plate_number" binding:"required"`
	Description string `json:"description"`
}

func (h *ViolationHandler) Report(c *gin.Context) {
	user, ok := middleware.UserFromContext(c)
	if !ok {
		writeError(c, apperrors.New(apperrors.CodeUnauthorized, "unauthorized", http.StatusUnauthorized))
		return
	}
	if user.Role != authdomain.RoleOfficer {
		writeError(c, apperrors.New(apperrors.CodeForbidden, "only officers can report violations", http.StatusForbidden))
		return
	}

	var req reportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, apperrors.New(apperrors.CodeInvalidRequest, "invalid request body", http.StatusBadRequest))
		return
	}

	out, err := h.violations.Report(c.Request.Context(), user.UserID, usecase.ReportInput{
		RuleCode:    req.RuleCode,
		PlateNumber: req.PlateNumber,
		Description: req.Description,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, out)
}

func (h *ViolationHandler) Get(c *gin.Context) {
	user, ok := middleware.UserFromContext(c)
	if !ok {
		writeError(c, apperrors.New(apperrors.CodeUnauthorized, "unauthorized", http.StatusUnauthorized))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, apperrors.New(apperrors.CodeInvalidRequest, "invalid violation id", http.StatusBadRequest))
		return
	}

	out, err := h.violations.GetViolation(c.Request.Context(), user.UserID, user.Role, id)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, out)
}

func (h *ViolationHandler) List(c *gin.Context) {
	user, ok := middleware.UserFromContext(c)
	if !ok {
		writeError(c, apperrors.New(apperrors.CodeUnauthorized, "unauthorized", http.StatusUnauthorized))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	out, err := h.violations.List(c.Request.Context(), user.UserID, user.Role, page, pageSize)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, out)
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
