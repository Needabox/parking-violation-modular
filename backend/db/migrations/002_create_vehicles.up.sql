-- +goose Up
CREATE TABLE vehicles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    plate_number VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT vehicles_plate_number_unique UNIQUE (plate_number)
);

CREATE INDEX idx_vehicles_user_id ON vehicles (user_id);
CREATE INDEX idx_vehicles_plate_number ON vehicles (plate_number);

-- +goose Down
DROP TABLE IF EXISTS vehicles;
