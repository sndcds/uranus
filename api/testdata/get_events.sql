-- Minimal projections for exercising the public handlers with their real SQL.
CREATE TABLE uranus.event_projection (
    event_uuid uuid PRIMARY KEY,
    org_uuid uuid NOT NULL,
    release_status text NOT NULL DEFAULT 'released',
    title text NOT NULL,
    subtitle text,
    summary text,
    categories integer[] DEFAULT '{1}',
    types jsonb DEFAULT '[[1,10]]',
    languages text[],
    tags text[],
    org_name text DEFAULT 'Organizer',
    org_postal_code text,
    org_city text,
    org_country text,
    image_uuid uuid,
    image_ai_label text,
    venue_uuid uuid DEFAULT '01994126-6680-7000-8000-000000099999',
    space_uuid uuid,
    venue_name text DEFAULT 'Hall',
    venue_city text,
    venue_street text,
    venue_house_number text,
    venue_postal_code text,
    venue_state text,
    venue_country text,
    venue_point geometry(Point,4326) DEFAULT ST_SetSRID(ST_MakePoint(9.4,54.8),4326),
    space_accessibility_flags bigint,
    min_age integer,
    max_age integer,
    price_type text,
    min_price double precision,
    max_price double precision,
    currency text,
    visitor_info_flags bigint
);

CREATE TABLE uranus.event_date_projection (
    event_date_uuid uuid PRIMARY KEY,
    event_uuid uuid NOT NULL REFERENCES uranus.event_projection(event_uuid),
    start_date date NOT NULL,
    start_time time,
    end_date date,
    end_time time,
    entry_time time,
    duration integer,
    all_day boolean,
    release_status text DEFAULT 'inherited',
    ticket_link text,
    venue_uuid uuid,
    space_uuid uuid,
    venue_name text,
    venue_city text,
    venue_street text,
    venue_house_number text,
    venue_postal_code text,
    venue_state text,
    venue_country text,
    venue_point geometry(Point,4326),
    space_accessibility_flags bigint,
    event_start_at timestamp GENERATED ALWAYS AS (
        start_date + COALESCE(start_time, TIME '00:00')
    ) STORED,
    event_end_at timestamp GENERATED ALWAYS AS (
        end_date + COALESCE(end_time, TIME '23:59')
    ) STORED
);

CREATE VIEW uranus.event_date AS
SELECT event_date_uuid AS uuid, event_uuid, start_date, start_time
FROM uranus.event_date_projection;
