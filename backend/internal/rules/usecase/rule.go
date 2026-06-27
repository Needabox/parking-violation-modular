package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/parking-violation-portal/backend/internal/rules/domain"
	"github.com/parking-violation-portal/backend/internal/rules/port"
	apperrors "github.com/parking-violation-portal/backend/pkg/errors"
)

type RuleService struct {
	repo port.RuleRepository
}

func NewRuleService(repo port.RuleRepository) *RuleService {
	return &RuleService{repo: repo}
}

type CreateRuleInput struct {
	Code        string
	Name        string
	Description string
	FineAmount  float64
}

type UpdateRuleInput struct {
	Name        string
	Description string
	FineAmount  float64
}

type RuleOutput struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	FineAmount  float64   `json:"fine_amount"`
	Version     int32     `json:"version"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toRuleOutput(r domain.FineRule) RuleOutput {
	return RuleOutput{
		ID:          r.ID,
		Code:        r.Code,
		Name:        r.Name,
		Description: r.Description,
		FineAmount:  r.FineAmount,
		Version:     r.Version,
		IsActive:    r.IsActive,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func toRuleOutputs(rules []domain.FineRule) []RuleOutput {
	outs := make([]RuleOutput, len(rules))
	for i, r := range rules {
		outs[i] = toRuleOutput(r)
	}
	return outs
}

func (s *RuleService) Create(ctx context.Context, input CreateRuleInput) (RuleOutput, error) {
	code := domain.NormalizeCode(input.Code)

	if !domain.IsValidCode(code) {
		return RuleOutput{}, apperrors.Validation("validation failed", apperrors.FieldError{
			Field: "code", Message: "code must be between 3 and 50 characters",
		})
	}
	if !domain.IsValidName(input.Name) {
		return RuleOutput{}, apperrors.Validation("validation failed", apperrors.FieldError{
			Field: "name", Message: "name must be between 3 and 255 characters",
		})
	}
	if !domain.IsValidAmount(input.FineAmount) {
		return RuleOutput{}, apperrors.Validation("validation failed", apperrors.FieldError{
			Field: "fine_amount", Message: "fine_amount must be greater than or equal to 0",
		})
	}

	// Check if active rule already exists
	_, err := s.repo.GetActiveRuleByCode(ctx, code)
	if err == nil {
		return RuleOutput{}, apperrors.New(apperrors.CodeConflict, "rule already exists; update it instead", 409)
	} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return RuleOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to check existing rule", 500, err)
	}

	newRule := domain.FineRule{
		Code:        code,
		Name:        input.Name,
		Description: input.Description,
		FineAmount:  input.FineAmount,
		Version:     1,
		IsActive:    true,
	}

	created, err := s.repo.CreateRule(ctx, newRule)
	if err != nil {
		return RuleOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to create rule", 500, err)
	}

	return toRuleOutput(created), nil
}

func (s *RuleService) Update(ctx context.Context, rawCode string, input UpdateRuleInput) (RuleOutput, error) {
	code := domain.NormalizeCode(rawCode)

	if !domain.IsValidCode(code) {
		return RuleOutput{}, apperrors.Validation("validation failed", apperrors.FieldError{
			Field: "code", Message: "code must be between 3 and 50 characters",
		})
	}
	if !domain.IsValidName(input.Name) {
		return RuleOutput{}, apperrors.Validation("validation failed", apperrors.FieldError{
			Field: "name", Message: "name must be between 3 and 255 characters",
		})
	}
	if !domain.IsValidAmount(input.FineAmount) {
		return RuleOutput{}, apperrors.Validation("validation failed", apperrors.FieldError{
			Field: "fine_amount", Message: "fine_amount must be greater than or equal to 0",
		})
	}

	// Fetch active version
	active, err := s.repo.GetActiveRuleByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RuleOutput{}, apperrors.New(apperrors.CodeNotFound, "active rule not found", 404)
		}
		return RuleOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to check existing rule", 500, err)
	}

	newRule := domain.FineRule{
		Code:        code,
		Name:        input.Name,
		Description: input.Description,
		FineAmount:  input.FineAmount,
		Version:     active.Version + 1,
		IsActive:    true,
	}

	var updated domain.FineRule
	err = s.repo.WithTx(ctx, func(txRepo port.RuleRepository) error {
		// Deactivate active version
		if err := txRepo.DeactivateActiveRule(ctx, code); err != nil {
			return err
		}

		// Insert new rule version
		res, err := txRepo.CreateRule(ctx, newRule)
		if err != nil {
			return err
		}
		updated = res
		return nil
	})

	if err != nil {
		return RuleOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to update rule", 500, err)
	}

	return toRuleOutput(updated), nil
}

func (s *RuleService) GetActiveByCode(ctx context.Context, rawCode string) (RuleOutput, error) {
	code := domain.NormalizeCode(rawCode)
	rule, err := s.repo.GetActiveRuleByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RuleOutput{}, apperrors.New(apperrors.CodeNotFound, "rule not found", 404)
		}
		return RuleOutput{}, apperrors.Wrap(apperrors.CodeInternal, "failed to fetch rule", 500, err)
	}
	return toRuleOutput(rule), nil
}

func (s *RuleService) ListActive(ctx context.Context) ([]RuleOutput, error) {
	rules, err := s.repo.ListActiveRules(ctx)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternal, "failed to list rules", 500, err)
	}
	return toRuleOutputs(rules), nil
}

func (s *RuleService) ListVersions(ctx context.Context, rawCode string) ([]RuleOutput, error) {
	code := domain.NormalizeCode(rawCode)
	
	// Check if rule code exists first by trying to find at least one version
	rules, err := s.repo.ListRuleVersions(ctx, code)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternal, "failed to list rule versions", 500, err)
	}
	if len(rules) == 0 {
		return nil, apperrors.New(apperrors.CodeNotFound, "rule not found", 404)
	}

	return toRuleOutputs(rules), nil
}
