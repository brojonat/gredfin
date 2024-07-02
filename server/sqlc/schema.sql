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
  UNIQUE (event_id),
  PRIMARY KEY (property_id, listing_id, event_description, event_ts)
);

-- NOTE: Annoyingly, atlas isn't applying the following views. I need to debug
-- why, but for now, note that these were manually added to the DB.

-- This view returns the most recent price event for every property listing.
CREATE OR REPLACE VIEW last_property_price_event AS
SELECT DISTINCT ON (property_id, listing_id) event_id, property_id, listing_id, price, event_description, source, source_id , event_ts
FROM property_events
WHERE price != 0
ORDER BY property_id, listing_id, event_ts DESC, price DESC;

-- This view returns properties with the price extracted from its most recent event.
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
INNER JOIN last_property_price_event pe ON
	p.property_id = pe.property_id AND
	p.listing_id = pe.listing_id;
