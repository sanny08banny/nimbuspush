-- +goose Up
CREATE TABLE devices (
    id SERIAL PRIMARY KEY,
    device_id TEXT UNIQUE NOT NULL,
    token TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ -- this line added for soft delete support
);

-- +goose Down
DROP TABLE devices;
