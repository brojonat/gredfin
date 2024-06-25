CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE property (
  property_id INT,
  listing_id INT,
  url VARCHAR(128),
  zipcode VARCHAR(5),
  city VARCHAR(128),
  state VARCHAR(32),
  location GEOGRAPHY(Point, 4326),
  list_price INT,
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
