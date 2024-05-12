-- +goose Up
-- +goose StatementBegin
CREATE TABLE command (
                          id SERIAL PRIMARY KEY,
                          script TEXT NOT NULL,
                          status VARCHAR(20) NOT NULL,
                          output TEXT,
                          created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE command;
-- +goose StatementEnd
