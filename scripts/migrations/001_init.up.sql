CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS wrapper_credential (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name        text NOT NULL UNIQUE,
  payload     bytea NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS wrapper_server (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name            text NOT NULL UNIQUE,
  protocol        text NOT NULL CHECK (protocol IN ('http','postgres','mysql','file')),
  options         jsonb NOT NULL DEFAULT '{}',
  credential_ref  uuid REFERENCES wrapper_credential(id) ON DELETE SET NULL,
  enabled         boolean NOT NULL DEFAULT true,
  updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS wrapper_table (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  server_id   uuid NOT NULL REFERENCES wrapper_server(id) ON DELETE CASCADE,
  schema_name text NOT NULL DEFAULT 'public',
  table_name  text NOT NULL,
  remote_name text,
  key_columns text[] NOT NULL DEFAULT '{}',
  options     jsonb NOT NULL DEFAULT '{}',
  UNIQUE (schema_name, table_name)
);

CREATE TABLE IF NOT EXISTS wrapper_column (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  table_id    uuid NOT NULL REFERENCES wrapper_table(id) ON DELETE CASCADE,
  name        text NOT NULL,
  data_type   text NOT NULL DEFAULT 'text',
  remote_name text,
  nullable    boolean NOT NULL DEFAULT true,
  position    int NOT NULL DEFAULT 0,
  UNIQUE (table_id, name)
);

CREATE TABLE IF NOT EXISTS wrapper_function (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  server_id   uuid NOT NULL REFERENCES wrapper_server(id) ON DELETE CASCADE,
  schema_name text NOT NULL DEFAULT 'public',
  name        text NOT NULL,
  operation   text NOT NULL CHECK (operation IN ('select','insert','update','upsert','delete','invoke')),
  remote_path text,
  method      text,
  options     jsonb NOT NULL DEFAULT '{}',
  UNIQUE (schema_name, name)
);
