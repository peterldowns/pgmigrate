CREATE TABLE companies (
  id bigint PRIMARY KEY NOT NULL GENERATED ALWAYS AS IDENTITY,
  created_at timestamptz NOT NULL DEFAULT now(),
  domain text UNIQUE NOT NULL,
  name text NOT NULL
);

CREATE TABLE users (
  id bigint PRIMARY KEY NOT NULL GENERATED ALWAYS AS IDENTITY,
  created_at timestamptz NOT NULL DEFAULT now(),
  name text NOT NULL,
  email text UNIQUE NOT NULL,
  company_id bigint REFERENCES companies (id)
);

CREATE TABLE blobs (
  id bigint PRIMARY KEY NOT NULL GENERATED ALWAYS AS IDENTITY,
  created_at timestamptz NOT NULL DEFAULT now(),
  company_id bigint NOT NULL REFERENCES companies (id),
  owner_id bigint REFERENCES users (id),
  name text NOT NULL,
  data jsonb NOT NULL default '{}'::jsonb
);

