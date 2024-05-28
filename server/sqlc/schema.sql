CREATE TABLE property (
  property_id INT,
  listing_id INT,
  url VARCHAR(128),
  last_scrape_ts TIMESTAMP DEFAULT '19700101 00:00:00'::TIMESTAMP,
  last_scrape_status VARCHAR(16) DEFAULT 'good',
  last_scrape_checksums JSONB NOT NULL DEFAULT '{}'::JSONB,
  PRIMARY KEY (property_id, listing_id)
);

-- TODO: add property sale status to Property?

CREATE TABLE search (
  search_id SERIAL PRIMARY KEY,
  query VARCHAR(128),
  last_scrape_ts TIMESTAMP NOT NULL DEFAULT '19700101 00:00:00'::TIMESTAMP,
  last_scrape_status VARCHAR(16) NOT NULL DEFAULT 'good',
  UNIQUE (query)
);

CREATE TABLE realtor (
  realtor_id SERIAL,
  realtor_name VARCHAR(64),
  realtor_company VARCHAR(64),
  property_id INT,
  listing_id INT,
  list_price INT,
  FOREIGN KEY (property_id, listing_id) REFERENCES property (property_id, listing_id),
  PRIMARY KEY (realtor_id, property_id, listing_id)
);