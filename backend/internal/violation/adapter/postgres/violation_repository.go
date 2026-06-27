package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/parking-violation-portal/backend/internal/violation/adapter/postgres/violationdb"
	"github.com/parking-violation-portal/backend/internal/violation/domain"
)

type ViolationRepository struct {
	queries *violationdb.Queries
}

func NewViolationRepository(pool *pgxpool.Pool) *ViolationRepository {
	return &ViolationRepository{
		queries: violationdb.New(pool),
	}
}

func (r *ViolationRepository) CreateViolation(ctx context.Context, v domain.Violation) (domain.Violation, error) {
	row, err := r.queries.CreateViolation(ctx, violationdb.CreateViolationParams{
		FineRuleID:  v.FineRuleID,
		PlateNumber: v.PlateNumber,
		OfficerID:   v.OfficerID,
		Status:      v.Status,
		Description: v.Description,
	})
	if err != nil {
		return domain.Violation{}, err
	}
	return mapViolation(row), nil
}

func (r *ViolationRepository) GetViolationByID(ctx context.Context, id uuid.UUID) (domain.ViolationDetail, error) {
	row, err := r.queries.GetViolationByID(ctx, id)
	if err != nil {
		return domain.ViolationDetail{}, err
	}
	return mapGetViolationByIDRow(row), nil
}

func (r *ViolationRepository) ListAllViolations(ctx context.Context, limit, offset int32) ([]domain.ViolationDetail, error) {
	rows, err := r.queries.ListAllViolations(ctx, violationdb.ListAllViolationsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	return mapListAllViolationsRows(rows), nil
}

func (r *ViolationRepository) CountAllViolations(ctx context.Context) (int64, error) {
	return r.queries.CountAllViolations(ctx)
}

func (r *ViolationRepository) ListViolationsByPlates(ctx context.Context, plates []string, limit, offset int32) ([]domain.ViolationDetail, error) {
	rows, err := r.queries.ListViolationsByPlates(ctx, violationdb.ListViolationsByPlatesParams{
		Column1: plates,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, err
	}
	return mapListViolationsByPlatesRows(rows), nil
}

func (r *ViolationRepository) CountViolationsByPlates(ctx context.Context, plates []string) (int64, error) {
	return r.queries.CountViolationsByPlates(ctx, plates)
}

func (r *ViolationRepository) UpdateViolationStatus(ctx context.Context, id uuid.UUID, status string) (domain.Violation, error) {
	row, err := r.queries.UpdateViolationStatus(ctx, violationdb.UpdateViolationStatusParams{
		ID:     id,
		Status: status,
	})
	if err != nil {
		return domain.Violation{}, err
	}
	return mapViolation(row), nil
}

func mapViolation(v violationdb.Violation) domain.Violation {
	return domain.Violation{
		ID:          v.ID,
		FineRuleID:  v.FineRuleID,
		PlateNumber: v.PlateNumber,
		OfficerID:   v.OfficerID,
		Status:      v.Status,
		Description: v.Description,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
	}
}

func mapGetViolationByIDRow(row violationdb.GetViolationByIDRow) domain.ViolationDetail {
	return domain.ViolationDetail{
		Violation: domain.Violation{
			ID:          row.ID,
			FineRuleID:  row.FineRuleID,
			PlateNumber: row.PlateNumber,
			OfficerID:   row.OfficerID,
			Status:      row.Status,
			Description: row.Description,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		},
		RuleCode:        row.RuleCode,
		RuleName:        row.RuleName,
		RuleDescription: row.RuleDescription,
		FineAmount:      row.FineAmount,
		RuleVersion:     row.RuleVersion,
	}
}

func mapListAllViolationsRows(rows []violationdb.ListAllViolationsRow) []domain.ViolationDetail {
	items := make([]domain.ViolationDetail, len(rows))
	for i, r := range rows {
		items[i] = domain.ViolationDetail{
			Violation: domain.Violation{
				ID:          r.ID,
				FineRuleID:  r.FineRuleID,
				PlateNumber: r.PlateNumber,
				OfficerID:   r.OfficerID,
				Status:      r.Status,
				Description: r.Description,
				CreatedAt:   r.CreatedAt,
				UpdatedAt:   r.UpdatedAt,
			},
			RuleCode:        r.RuleCode,
			RuleName:        r.RuleName,
			RuleDescription: r.RuleDescription,
			FineAmount:      r.FineAmount,
			RuleVersion:     r.RuleVersion,
		}
	}
	return items
}

func mapListViolationsByPlatesRows(rows []violationdb.ListViolationsByPlatesRow) []domain.ViolationDetail {
	items := make([]domain.ViolationDetail, len(rows))
	for i, r := range rows {
		items[i] = domain.ViolationDetail{
			Violation: domain.Violation{
				ID:          r.ID,
				FineRuleID:  r.FineRuleID,
				PlateNumber: r.PlateNumber,
				OfficerID:   r.OfficerID,
				Status:      r.Status,
				Description: r.Description,
				CreatedAt:   r.CreatedAt,
				UpdatedAt:   r.UpdatedAt,
			},
			RuleCode:        r.RuleCode,
			RuleName:        r.RuleName,
			RuleDescription: r.RuleDescription,
			FineAmount:      r.FineAmount,
			RuleVersion:     r.RuleVersion,
		}
	}
	return items
}
