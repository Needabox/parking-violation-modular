-- +goose Up
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    violation_id UUID NOT NULL REFERENCES violations (id),
    amount DOUBLE PRECISION NOT NULL,
    status VARCHAR(20) NOT NULL,
    payment_method VARCHAR(50) NOT NULL DEFAULT 'CARD',
    reference_id VARCHAR(100) NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT payments_status_check CHECK (status IN ('SUCCESS', 'FAILED'))
);

CREATE INDEX idx_payments_violation_id ON payments (violation_id);

-- +goose Down
DROP INDEX IF EXISTS idx_payments_violation_id;
DROP TABLE IF EXISTS payments;
