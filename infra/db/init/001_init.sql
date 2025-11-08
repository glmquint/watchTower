-- Schema: users and incidents, plus seed data

CREATE TABLE IF NOT EXISTS users (
  id          BIGSERIAL PRIMARY KEY,
  email       TEXT NOT NULL UNIQUE,
  name        TEXT NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS incidents (
  id          BIGSERIAL PRIMARY KEY,
  title       TEXT NOT NULL,
  description TEXT,
  user_id     BIGINT REFERENCES users(id) ON DELETE SET NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed data (idempotent-ish): ensure one demo user and a couple of incidents
INSERT INTO users (email, name)
VALUES ('demo@example.com', 'Demo User')
ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO incidents (title, description, user_id)
SELECT 'Server down', 'web-1 is not responding', u.id FROM users u WHERE u.email='demo@example.com'
ON CONFLICT DO NOTHING;

INSERT INTO incidents (title, description, user_id)
SELECT 'High latency', 'API latency > 2s', u.id FROM users u WHERE u.email='demo@example.com'
ON CONFLICT DO NOTHING;
