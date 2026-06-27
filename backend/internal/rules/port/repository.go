package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/parking-violation-portal/backend/internal/rules/domain"
)

type RuleRepository interface {
	CreateRule(ctx context.Context, rule domain.FineRule) (domain.FineRule, error)
	DeactivateActiveRule(ctx context.Context, code string) error
	GetActiveRuleByCode(ctx context.Context, code string) (domain.FineRule, error)
	GetRuleByID(ctx context.Context, id uuid.UUID) (domain.FineRule, error)
	GetRuleByCodeAndVersion(ctx context.Context, code string, version int32) (domain.FineRule, error)
	ListActiveRules(ctx context.Context) ([]domain.FineRule, error)
	ListRuleVersions(ctx context.Context, code string) ([]domain.FineRule, error)
	WithTx(ctx context.Context, fn func(repo RuleRepository) error) error
}
