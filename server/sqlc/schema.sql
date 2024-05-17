CREATE TABLE property (
  property_id VARCHAR(32),
  listing_id VARCHAR(32),
  address VARCHAR(128),
  zipcode VARCHAR(10),
  state CHAR(2),
  last_scrape_ts TIMESTAMP DEFAULT '19700101 00:00:00',
  last_scrape_status VARCHAR(16),
  PRIMARY KEY (property_id, listing_id)
);

CREATE TABLE search (
  search_id SERIAL PRIMARY KEY,
  query VARCHAR(128),
  last_scrape_ts TIMESTAMP DEFAULT '19700101 00:00:00',
  last_scrape_status VARCHAR(16)
);

CREATE TABLE realtor (
  realtor_id SERIAL PRIMARY KEY,
  realtor_name VARCHAR(64),
  realtor_region VARCHAR(32),
  property_id VARCHAR(32),
  listing_id VARCHAR(32),
  list_price INT,
  FOREIGN KEY (property_id, listing_id) REFERENCES property (property_id, listing_id),
  PRIMARY KEY (realtor_id, property_id, listing_id)
);