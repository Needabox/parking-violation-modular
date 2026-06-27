-- +goose Up
CREATE TABLE fine_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    fine_amount DOUBLE PRECISION NOT NULL,
    version INT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_fine_rules_active_code ON fine_rules (code) WHERE is_active = TRUE;
CREATE UNIQUE INDEX idx_fine_rules_code_version ON fine_rules (code, version);
CREATE INDEX idx_fine_rules_code ON fine_rules (code);

-- +goose Down
DROP INDEX IF EXISTS idx_fine_rules_code;
DROP INDEX IF EXISTS idx_fine_rules_code_version;
DROP INDEX IF EXISTS idx_fine_rules_active_code;
DROP TABLE IF EXISTS fine_rules;
