create or replace function array_to_string_immutable (
    arg text[], 
    separator text,
    null_string text default null) 
returns text immutable parallel safe language sql as $$
select array_to_string(arg,separator,null_string) $$;

CREATE TABLE people (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nickname VARCHAR(32) NOT NULL,
    name VARCHAR(100) NOT NULL,
    birthday TIMESTAMPTZ NOT NULL,
    stack TEXT[],
    search tsvector GENERATED ALWAYS AS (
        to_tsvector('simple'::regconfig, name || ' ' || nickname || ' ' || coalesce(array_to_string_immutable(stack, ' '::text), ''))
    ) STORED
);

CREATE UNIQUE INDEX idx_people_nickname ON people (nickname);
CREATE INDEX idx_people_search ON people USING GIN (search);