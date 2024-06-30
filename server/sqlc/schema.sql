CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE property (
  property_id INT,
  listing_id INT,
  url VARCHAR(128),
  zipcode VARCHAR(5),
  city VARCHAR(128),
  state VARCHAR(32),
  location GEOMETRY(Point, 4326) NOT NULL,
  last_scrape_ts TIMESTAMP DEFAULT '19700101 00:00:00'::TIMESTAMP,
  last_scrape_status VARCHAR(16) DEFAULT 'good',
  last_scrape_checksums JSONB NOT NULL DEFAULT '{}'::JSONB,
  PRIMARY KEY (property_id, listing_id)
);

CREATE TABLE search (
  search_id SERIAL PRIMARY KEY,
  query VARCHAR(128),
  last_scrape_ts TIMESTAMP NOT NULL DEFAULT '19700101 00:00:00'::TIMESTAMP,
  last_scrape_status VARCHAR(16) NOT NULL DEFAULT 'good',
  UNIQUE (query)
);

CREATE TABLE realtor (
  realtor_id SERIAL,
  name VARCHAR(128),
  company VARCHAR(128),
  property_id INT,
  listing_id INT,
  created_ts TIMESTAMP NOT NULL DEFAULT '19700101 00:00:00'::TIMESTAMP,
  FOREIGN KEY (property_id, listing_id) REFERENCES property (property_id, listing_id) ON DELETE CASCADE,
  PRIMARY KEY (name, property_id, listing_id),
  UNIQUE (realtor_id)
);

CREATE TABLE property_blocklist (
  url VARCHAR(128),
  expl TEXT,
  PRIMARY KEY (url)
);

CREATE TABLE property_events (
  event_id SERIAL,
  property_id INT,
  listing_id INT,
  price INT,
  event_description VARCHAR(64),
  source VARCHAR(64),
  source_id VARCHAR(32),
  event_ts TIMESTAMP DEFAULT '19700101 00:00:00'::TIMESTAMP,
  FOREIGN KEY (property_id, listing_id) REFERENCES property (property_id, listing_id) ON DELETE CASCADE,
  PRIMARY KEY (event_id)
);

-- NOTE: Annoyingly, atlas isn't applying the following views. I need to debug
-- why, but for now, note that these were manually added to the DB.

-- returns the most recent row for every property_id
CREATE OR REPLACE VIEW last_property_price_event AS
SELECT p.event_id, p.property_id, p.listing_id, p.price, p.event_description, p.source, p.source_id , p.event_ts
FROM property_events p
INNER JOIN (
	SELECT property_id, MAX(ipe.event_ts) AS max_event_ts
	FROM property_events ipe
	WHERE price != 0
	GROUP BY property_id
) AS pe ON p.event_ts = pe.max_event_ts;

-- returns the property with the price of its most recent event
CREATE OR REPLACE VIEW property_price AS
SELECT
  p.property_id,
  p.listing_id,
  pe.price,
  p.url,
  p.zipcode,
  p.city,
  p.state,
  p.location,
  p.last_scrape_ts,
  p.last_scrape_status,
  p.last_scrape_checksums
FROM property p
INNER JOIN last_property_price_event pe ON p.property_id = pe.property_id;
