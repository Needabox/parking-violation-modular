package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/parking-violation-portal/backend/internal/rules/adapter/postgres/ruledb"
	"github.com/parking-violation-portal/backend/internal/rules/domain"
	"github.com/parking-violation-portal/backend/internal/rules/port"
)

type RuleRepository struct {
	db      ruledb.DBTX
	queries *ruledb.Queries
}

func NewRuleRepository(pool *pgxpool.Pool) *RuleRepository {
	return &RuleRepository{
		db:      pool,
		queries: ruledb.New(pool),
	}
}

func (r *RuleRepository) CreateRule(ctx context.Context, rule domain.FineRule) (domain.FineRule, error) {
	row, err := r.queries.CreateRule(ctx, ruledb.CreateRuleParams{
		Code:        rule.Code,
		Name:        rule.Name,
		Description: rule.Description,
		FineAmount:  rule.FineAmount,
		Version:     rule.Version,
		IsActive:    rule.IsActive,
	})
	if err != nil {
		return domain.FineRule{}, err
	}
	return mapFineRule(row), nil
}

func (r *RuleRepository) DeactivateActiveRule(ctx context.Context, code string) error {
	return r.queries.DeactivateRuleVersions(ctx, code)
}

func (r *RuleRepository) GetActiveRuleByCode(ctx context.Context, code string) (domain.FineRule, error) {
	row, err := r.queries.GetActiveRuleByCode(ctx, code)
	if err != nil {
		return domain.FineRule{}, err
	}
	return mapFineRule(row), nil
}

func (r *RuleRepository) GetRuleByID(ctx context.Context, id uuid.UUID) (domain.FineRule, error) {
	row, err := r.queries.GetRuleByID(ctx, id)
	if err != nil {
		return domain.FineRule{}, err
	}
	return mapFineRule(row), nil
}

func (r *RuleRepository) GetRuleByCodeAndVersion(ctx context.Context, code string, version int32) (domain.FineRule, error) {
	row, err := r.queries.GetRuleByCodeAndVersion(ctx, ruledb.GetRuleByCodeAndVersionParams{
		Code:    code,
		Version: version,
	})
	if err != nil {
		return domain.FineRule{}, err
	}
	return mapFineRule(row), nil
}

func (r *RuleRepository) ListActiveRules(ctx context.Context) ([]domain.FineRule, error) {
	rows, err := r.queries.ListActiveRules(ctx)
	if err != nil {
		return nil, err
	}
	return mapFineRules(rows), nil
}

func (r *RuleRepository) ListRuleVersions(ctx context.Context, code string) ([]domain.FineRule, error) {
	rows, err := r.queries.ListRuleVersions(ctx, code)
	if err != nil {
		return nil, err
	}
	return mapFineRules(rows), nil
}

func (r *RuleRepository) WithTx(ctx context.Context, fn func(repo port.RuleRepository) error) error {
	pool, ok := r.db.(*pgxpool.Pool)
	if !ok {
		// Already in a transaction
		return fn(r)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	txRepo := &RuleRepository{
		db:      tx,
		queries: ruledb.New(tx),
	}

	if err := fn(txRepo); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func mapFineRule(row ruledb.FineRule) domain.FineRule {
	return domain.FineRule{
		ID:          row.ID,
		Code:        row.Code,
		Name:        row.Name,
		Description: row.Description,
		FineAmount:  row.FineAmount,
		Version:     row.Version,
		IsActive:    row.IsActive,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func mapFineRules(rows []ruledb.FineRule) []domain.FineRule {
	items := make([]domain.FineRule, len(rows))
	for i, row := range rows {
		items[i] = mapFineRule(row)
	}
	return items
}
