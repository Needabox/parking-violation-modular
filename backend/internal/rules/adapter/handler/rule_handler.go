package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/parking-violation-portal/backend/internal/auth/domain"
	"github.com/parking-violation-portal/backend/internal/gateway/middleware"
	"github.com/parking-violation-portal/backend/internal/rules/usecase"
	apperrors "github.com/parking-violation-portal/backend/pkg/errors"
)

type RuleHandler struct {
	rules *usecase.RuleService
}

func NewRuleHandler(rules *usecase.RuleService) *RuleHandler {
	return &RuleHandler{rules: rules}
}

type createRequest struct {
	Code        string   `json:"code" binding:"required"`
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	FineAmount  *float64 `json:"fine_amount" binding:"required"`
}

type updateRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	FineAmount  *float64 `json:"fine_amount" binding:"required"`
}

func (h *RuleHandler) Create(c *gin.Context) {
	user, ok := middleware.UserFromContext(c)
	if !ok {
		writeError(c, apperrors.New(apperrors.CodeUnauthorized, "unauthorized", http.StatusUnauthorized))
		return
	}
	if user.Role != domain.RoleOfficer {
		writeError(c, apperrors.New(apperrors.CodeForbidden, "only officers can create rules", http.StatusForbidden))
		return
	}

	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, apperrors.New(apperrors.CodeInvalidRequest, "invalid request body", http.StatusBadRequest))
		return
	}

	out, err := h.rules.Create(c.Request.Context(), usecase.CreateRuleInput{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		FineAmount:  *req.FineAmount,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, out)
}

func (h *RuleHandler) Update(c *gin.Context) {
	user, ok := middleware.UserFromContext(c)
	if !ok {
		writeError(c, apperrors.New(apperrors.CodeUnauthorized, "unauthorized", http.StatusUnauthorized))
		return
	}
	if user.Role != domain.RoleOfficer {
		writeError(c, apperrors.New(apperrors.CodeForbidden, "only officers can update rules", http.StatusForbidden))
		return
	}

	code := c.Param("code")
	if code == "" {
		writeError(c, apperrors.New(apperrors.CodeInvalidRequest, "missing rule code", http.StatusBadRequest))
		return
	}

	var req updateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, apperrors.New(apperrors.CodeInvalidRequest, "invalid request body", http.StatusBadRequest))
		return
	}

	out, err := h.rules.Update(c.Request.Context(), code, usecase.UpdateRuleInput{
		Name:        req.Name,
		Description: req.Description,
		FineAmount:  *req.FineAmount,
	})
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, out)
}

func (h *RuleHandler) GetActive(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		writeError(c, apperrors.New(apperrors.CodeInvalidRequest, "missing rule code", http.StatusBadRequest))
		return
	}

	out, err := h.rules.GetActiveByCode(c.Request.Context(), code)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, out)
}

func (h *RuleHandler) ListActive(c *gin.Context) {
	out, err := h.rules.ListActive(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, out)
}

func (h *RuleHandler) ListVersions(c *gin.Context) {
	user, ok := middleware.UserFromContext(c)
	if !ok {
		writeError(c, apperrors.New(apperrors.CodeUnauthorized, "unauthorized", http.StatusUnauthorized))
		return
	}
	if user.Role != domain.RoleOfficer {
		writeError(c, apperrors.New(apperrors.CodeForbidden, "only officers can view version history", http.StatusForbidden))
		return
	}

	code := c.Param("code")
	if code == "" {
		writeError(c, apperrors.New(apperrors.CodeInvalidRequest, "missing rule code", http.StatusBadRequest))
		return
	}

	out, err := h.rules.ListVersions(c.Request.Context(), code)
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
