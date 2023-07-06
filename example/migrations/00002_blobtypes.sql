CREATE TABLE blob_type_enum (
  value text UNIQUE PRIMARY KEY NOT NULL 
);

INSERT INTO blob_type_enum VALUES
('pending_review'),
('approved'),
('rejected');

ALTER TABLE blobs ADD COLUMN status text NOT NULL DEFAULT 'pending_review' REFERENCES blob_type_enum (value);
