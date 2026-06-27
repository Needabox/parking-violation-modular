package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/parking-violation-portal/backend/internal/violation/domain"
)

type ViolationRepository interface {
	CreateViolation(ctx context.Context, v domain.Violation) (domain.Violation, error)
	GetViolationByID(ctx context.Context, id uuid.UUID) (domain.ViolationDetail, error)
	ListAllViolations(ctx context.Context, limit, offset int32) ([]domain.ViolationDetail, error)
	CountAllViolations(ctx context.Context) (int64, error)
	ListViolationsByPlates(ctx context.Context, plates []string, limit, offset int32) ([]domain.ViolationDetail, error)
	CountViolationsByPlates(ctx context.Context, plates []string) (int64, error)
	UpdateViolationStatus(ctx context.Context, id uuid.UUID, status string) (domain.Violation, error)
}
