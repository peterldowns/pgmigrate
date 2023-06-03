CREATE TABLE cats (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT
);

INSERT INTO cats (name)
VALUES ('daisy'), ('sunny');
