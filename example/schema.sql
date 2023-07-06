CREATE TABLE public.blob_type_enum (
  value text PRIMARY KEY NOT NULL
);

CREATE TABLE public.blobs (
  id bigint PRIMARY KEY NOT NULL GENERATED ALWAYS AS IDENTITY,
  created_at timestamp with time zone NOT NULL DEFAULT now(),
  company_id bigint NOT NULL,
  owner_id bigint,
  name text NOT NULL,
  data jsonb NOT NULL DEFAULT '{}'::jsonb,
  status text NOT NULL DEFAULT 'pending_review'::text
);

CREATE TABLE public.companies (
  id bigint PRIMARY KEY NOT NULL GENERATED ALWAYS AS IDENTITY,
  created_at timestamp with time zone NOT NULL DEFAULT now(),
  domain text UNIQUE NOT NULL,
  name text NOT NULL
);

ALTER TABLE public.blobs
ADD CONSTRAINT blobs_company_id_fkey
FOREIGN KEY (company_id) REFERENCES companies(id);

CREATE TABLE public.users (
  id bigint PRIMARY KEY NOT NULL GENERATED ALWAYS AS IDENTITY,
  created_at timestamp with time zone NOT NULL DEFAULT now(),
  name text NOT NULL,
  email text UNIQUE NOT NULL,
  company_id bigint
);

ALTER TABLE public.blobs
ADD CONSTRAINT blobs_owner_id_fkey
FOREIGN KEY (owner_id) REFERENCES users(id);

ALTER TABLE public.blobs
ADD CONSTRAINT blobs_status_fkey
FOREIGN KEY (status) REFERENCES blob_type_enum(value);

CREATE TABLE public.cats (
  id bigint PRIMARY KEY NOT NULL GENERATED ALWAYS AS IDENTITY,
  name text UNIQUE NOT NULL,
  so_pretty_and_elegant boolean NOT NULL DEFAULT true
);

CREATE TABLE public.dogs (
  id bigint PRIMARY KEY NOT NULL GENERATED ALWAYS AS IDENTITY,
  name text UNIQUE NOT NULL,
  very_good boolean NOT NULL DEFAULT true
);

CREATE TABLE public.pgmigrate_migrations (
  id text PRIMARY KEY NOT NULL,
  checksum text NOT NULL,
  execution_time_in_millis bigint NOT NULL,
  applied_at timestamp with time zone NOT NULL
);

CREATE VIEW public.reviewable_blobs AS
   SELECT blobs.id,
    blobs.created_at,
    blobs.company_id,
    blobs.owner_id,
    blobs.name,
    blobs.data,
    blobs.status
   FROM blobs
  WHERE (blobs.status = 'pending_review'::text);

ALTER TABLE public.users
ADD CONSTRAINT users_company_id_fkey
FOREIGN KEY (company_id) REFERENCES companies(id);

INSERT INTO public.blob_type_enum (value) VALUES
('pending_review'),
('approved'),
('rejected')
;
