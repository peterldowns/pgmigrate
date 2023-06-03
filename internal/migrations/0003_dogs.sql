CREATE TABLE dogs (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    age int,
    name TEXT,
    enemy_id BIGINT REFERENCES cats (id)
);

INSERT INTO dogs (name, enemy_id)
VALUES ('shep', 1);
