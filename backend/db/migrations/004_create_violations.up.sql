-- +goose Up
CREATE TABLE violations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fine_rule_id UUID NOT NULL REFERENCES fine_rules (id),
    plate_number VARCHAR(20) NOT NULL,
    officer_id UUID NOT NULL REFERENCES users (id),
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT violations_status_check CHECK (status IN ('PENDING', 'PAID'))
);

CREATE INDEX idx_violations_plate_number ON violations (plate_number);
CREATE INDEX idx_violations_fine_rule_id ON violations (fine_rule_id);
CREATE INDEX idx_violations_officer_id ON violations (officer_id);

-- +goose Down
DROP INDEX IF EXISTS idx_violations_officer_id;
DROP INDEX IF EXISTS idx_violations_fine_rule_id;
DROP INDEX IF EXISTS idx_violations_plate_number;
DROP TABLE IF EXISTS violations;
