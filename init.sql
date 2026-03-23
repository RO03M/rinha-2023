CREATE TABLE people (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nickname VARCHAR(32) NOT NULL,
    name VARCHAR(100) NOT NULL,
    birthday TIMESTAMPTZ NOT NULL,
    stack TEXT[],
    search tsvector
);

CREATE UNIQUE INDEX idx_people_nickname ON people (nickname);
CREATE INDEX idx_people_search ON people USING GIN (search);