package usecase

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	authdomain "github.com/parking-violation-portal/backend/internal/auth/domain"
	authport "github.com/parking-violation-portal/backend/internal/auth/port"
	rulesport "github.com/parking-violation-portal/backend/internal/rules/port"
	"github.com/parking-violation-portal/backend/internal/violation/domain"
	"github.com/parking-violation-portal/backend/internal/violation/port"
	apperrors "github.com/parking-violation-portal/backend/pkg/errors"
)

type ViolationService struct {
	repo     port.ViolationRepository
	vehicles authport.VehicleRepository
	rules    rulesport.RuleRepository
}

func NewViolationService(repo port.ViolationRepository, vehicles authport.VehicleRepository, rules rulesport.RuleRepository) *ViolationService {
	return &ViolationService{
		repo:     repo,
		vehicles: vehicles,
		rules:    rules,
	}
}

type ReportInput struct {
	RuleCode    string
	PlateNumber string
	Description string
}

type ViolationOutput struct {
	ID              uuid.UUID `json:"id"`
	FineRuleID      uuid.UUID `json:"fine_rule_id"`
	PlateNumber     string    `json:"plate_number"`
	OfficerID       uuid.UUID `json:"officer_id"`
	Status          string    `json:"status"`
	Description     string    `json:"description"`
	RuleCode        string    `json:"rule_code"`
	RuleName        string    `json:"rule_name"`
	RuleDescription string    `json:"rule_description"`
	FineAmount      float64   `json:"fine_amount"`
	RuleVersion     int32     `json:"rule_version"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ListMetadata struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalCount int64 `json:"total_count"`
	PageCount  int   `json:"page_count"`
}

type ListViolationsOutput struct {
	Data []ViolationOutput `json:"data"`
	Meta ListMetadata      `json:"meta"`
}

func toViolationOutput(d domain.ViolationDetail) ViolationOutput {
	return ViolationOutput{
		ID:              d.ID,
		FineRuleID:      d.FineRuleID,
		PlateNumber:     d.PlateNumber,
		OfficerID:       d.OfficerID,
		Status:          d.Status,
		Description:     d.Description,
		RuleCode:        d.RuleCode,
		RuleName:        d.RuleName,
		RuleDescription: d.RuleDescription,
		FineAmount:      d.FineAmount,
		RuleVersion:     d.RuleVersion,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}
}

func toViolationOutputs(details []domain.ViolationDetail) []ViolationOutput {
	outs := make([]ViolationOutput, len(details))
	for i, d := range details {
		outs[i] = toViolationOutput(d)
	}
	return outs
}

func (s *ViolationService) Report(ctx context.Context, officerID uuid.UUID, input ReportInput) (ViolationOutput, error) {
	plate := domain.NormalizePlate(input.PlateNumber)

	if !domain.IsValidPlate(plate) {
		return ViolationOutput{}, apperrors.Validation("validation failed", apperrors.FieldError{
			Field: "plate_number", Message: "plate number must be between 2 and 20 characters",
		})
	}

	if input.RuleCode == "" {
		return ViolationOutput{}, apperrors.Validation("validation failed", apperrors.FieldError{
			Field: "rule_code", Message: "rule code is required",
		})
	}

	// Resolve the active fine rule
	rule, err := s.rules.GetActiveRuleByCode(ctx, input.RuleCode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ViolationOutput{}, apperrors.Validation("validation failed", apperrors.FieldError{
				Field: "rule_code", Message: "specified active rule code does not exist",
			})
		}
		return ViolationOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to resolve rule", 500, err)
	}

	v := domain.Violation{
		FineRuleID:  rule.ID,
		PlateNumber: plate,
		OfficerID:   officerID,
		Status:      domain.StatusPending,
		Description: input.Description,
	}

	created, err := s.repo.CreateViolation(ctx, v)
	if err != nil {
		return ViolationOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to create violation", 500, err)
	}

	// Fetch detail representation (join with rule)
	detail, err := s.repo.GetViolationByID(ctx, created.ID)
	if err != nil {
		return ViolationOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to fetch violation details", 500, err)
	}

	return toViolationOutput(detail), nil
}

func (s *ViolationService) GetViolation(ctx context.Context, requesterID uuid.UUID, requesterRole string, id uuid.UUID) (ViolationOutput, error) {
	detail, err := s.repo.GetViolationByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ViolationOutput{}, apperrors.New(apperrors.CodeNotFound, "violation not found", 404)
		}
		return ViolationOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to fetch violation", 500, err)
	}

	// Authorization Check
	if requesterRole == authdomain.RoleMember {
		// A member can only retrieve the violation if the plate matches one of their registered vehicles
		vehicle, err := s.vehicles.GetVehicleByPlateNumber(ctx, detail.PlateNumber)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ViolationOutput{}, apperrors.New(apperrors.CodeForbidden, "access forbidden: vehicle not registered to you", 403)
			}
			return ViolationOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to verify vehicle", 500, err)
		}

		if vehicle.UserID != requesterID {
			return ViolationOutput{}, apperrors.New(apperrors.CodeForbidden, "access forbidden: vehicle not registered to you", 403)
		}
	}

	return toViolationOutput(detail), nil
}

func (s *ViolationService) List(ctx context.Context, requesterID uuid.UUID, requesterRole string, page, pageSize int) (ListViolationsOutput, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	limit := int32(pageSize)
	offset := int32((page - 1) * pageSize)

	var details []domain.ViolationDetail
	var totalCount int64
	var err error

	if requesterRole == authdomain.RoleOfficer {
		// Officers see all violations
		details, err = s.repo.ListAllViolations(ctx, limit, offset)
		if err != nil {
			return ListViolationsOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to list violations", 500, err)
		}

		totalCount, err = s.repo.CountAllViolations(ctx)
		if err != nil {
			return ListViolationsOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to count violations", 500, err)
		}
	} else if requesterRole == authdomain.RoleMember {
		// Members only see violations for their registered vehicles
		vehicles, err := s.vehicles.ListVehiclesByUserID(ctx, requesterID, 1000, 0)
		if err != nil {
			return ListViolationsOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to retrieve registered vehicles", 500, err)
		}

		// If no vehicles registered, return empty list
		if len(vehicles) == 0 {
			return ListViolationsOutput{
				Data: []ViolationOutput{},
				Meta: ListMetadata{
					Page:       page,
					PageSize:   pageSize,
					TotalCount: 0,
					PageCount:  0,
				},
			}, nil
		}

		// Extract plate numbers
		plates := make([]string, len(vehicles))
		for i, v := range vehicles {
			plates[i] = v.PlateNumber
		}

		details, err = s.repo.ListViolationsByPlates(ctx, plates, limit, offset)
		if err != nil {
			return ListViolationsOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to list violations for plates", 500, err)
		}

		totalCount, err = s.repo.CountViolationsByPlates(ctx, plates)
		if err != nil {
			return ListViolationsOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to count violations for plates", 500, err)
		}
	} else {
		return ListViolationsOutput{}, apperrors.New(apperrors.CodeForbidden, "forbidden role", 403)
	}

	pageCount := int(math.Ceil(float64(totalCount) / float64(pageSize)))

	return ListViolationsOutput{
		Data: toViolationOutputs(details),
		Meta: ListMetadata{
			Page:       page,
			PageSize:   pageSize,
			TotalCount: totalCount,
			PageCount:  pageCount,
		},
	}, nil
}
