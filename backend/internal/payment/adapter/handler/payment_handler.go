package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authdomain "github.com/parking-violation-portal/backend/internal/auth/domain"
	"github.com/parking-violation-portal/backend/internal/gateway/middleware"
	"github.com/parking-violation-portal/backend/internal/payment/usecase"
	apperrors "github.com/parking-violation-portal/backend/pkg/errors"
)

type PaymentHandler struct {
	payments *usecase.PaymentService
}

func NewPaymentHandler(payments *usecase.PaymentService) *PaymentHandler {
	return &PaymentHandler{payments: payments}
}

type payRequest struct {
	ViolationID string `json:"violation_id" binding:"required"`
	CardNumber  string `json:"card_number" binding:"required"`
	CVV         string `json:"cvv" binding:"required"`
	ExpiryMonth int    `json:"expiry_month" binding:"required"`
	ExpiryYear  int    `json:"expiry_year" binding:"required"`
}

func (h *PaymentHandler) Pay(c *gin.Context) {
	user, ok := middleware.UserFromContext(c)
	if !ok {
		writeError(c, apperrors.New(apperrors.CodeUnauthorized, "unauthorized", http.StatusUnauthorized))
		return
	}
	if user.Role != authdomain.RoleMember {
		writeError(c, apperrors.New(apperrors.CodeForbidden, "only members can make payments", http.StatusForbidden))
		return
	}

	var req payRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, apperrors.New(apperrors.CodeInvalidRequest, "invalid request body", http.StatusBadRequest))
		return
	}

	violationID, err := uuid.Parse(req.ViolationID)
	if err != nil {
		writeError(c, apperrors.New(apperrors.CodeInvalidRequest, "invalid violation ID format", http.StatusBadRequest))
		return
	}

	out, err := h.payments.Pay(c.Request.Context(), user.UserID, user.Role, usecase.PayInput{
		ViolationID: violationID,
		CardNumber:  req.CardNumber,
		CVV:         req.CVV,
		ExpiryMonth: req.ExpiryMonth,
		ExpiryYear:  req.ExpiryYear,
	})
	if err != nil {
		appErr, isAppErr := apperrors.IsAppError(err)
		if isAppErr && appErr.Code == apperrors.CodePaymentFailed {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error": gin.H{
					"code":    appErr.Code,
					"message": appErr.Message,
				},
				"payment": out,
			})
			return
		}
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, out)
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
