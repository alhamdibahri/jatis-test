CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE messages (
  id UUID,
  tenant_id UUID NOT NULL,
  payload JSONB,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  PRIMARY KEY (tenant_id, id)
) PARTITION BY LIST (tenant_id);
