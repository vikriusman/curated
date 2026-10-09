CREATE TABLE notes (
    id         BIGSERIAL PRIMARY KEY,
    body       TEXT        NOT NULL CHECK (length(body) BETWEEN 1 AND 500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
