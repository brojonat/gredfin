
-- These views are necessary because atlas isn't running the corresponding migration.

-- This view returns the most recent price event for every property.
CREATE OR REPLACE VIEW last_property_price_event AS
SELECT DISTINCT ON (property_id, listing_id) event_id, property_id, listing_id, price, event_description, source, source_id , event_ts
FROM property_events
WHERE price != 0
ORDER BY property_id, listing_id, event_ts DESC, price DESC;

-- This view returns property rows with an added price field extracted from the most recent price event.
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
  p.last_scrape_metadata
FROM property p
INNER JOIN last_property_price_event pe ON
	p.property_id = pe.property_id AND 
	p.listing_id = pe.listing_id;

SELECT SUM(m.value::NUMERIC) 
FROM "search" AS s, jsonb_each(last_scrape_metadata) AS m
WHERE m.key = 'success_count';

SELECT * FROM "search" s WHERE last_scrape_status = 'pending';

SELECT * FROM realtor WHERE name LIKE '%Mat%' AND company LIKE '%Keller%';

SELECT last_scrape_status, COUNT(*) AS count 
FROM property
WHERE last_scrape_ts > $1
GROUP BY last_scrape_status; 

SELECT * FROM property WHERE last_scrape_status ='good';

SELECT COUNT(*) FROM "search" WHERE last_scrape_metadata = '{}'::JSONB;

UPDATE "search" SET last_scrape_status = 'good' WHERE search_id =312;
UPDATE "search" SET last_scrape_metadata = '{}'::JSONB WHERE TRUE;
UPDATE "search" SET last_scrape_ts = '1970-01-01 00:00:00.000'::TIMESTAMP WHERE TRUE;


SELECT * FROM property ORDER BY property_id;
DELETE FROM property;
SELECT property_id , listing_id, zipcode, location::geometry AS location FROM property;
DELETE FROM property WHERE zipcode IS NULL;
SELECT count(*) AS "Property Count" FROM property;
DELETE FROM property WHERE property_id != 91325281;
UPDATE property SET last_scrape_status = 'good' WHERE last_scrape_status != 'good';

SELECT last_scrape_metadata -> 'image_urls' FROM property_price pp;

SELECT '[1, 2, "foo", null]'::json;

SELECT * FROM pg_catalog.pg_type WHERE oid = 18199;

SELECT sum("Property Count") FROM (
SELECT r.name, r.company, count(*) AS "Property Count" FROM realtor r 
LEFT JOIN realtor_property_through rpt ON r.realtor_id = rpt.realtor_id 
LEFT JOIN property_price pp ON rpt.property_id  = pp.property_id AND rpt.listing_id = pp.listing_id
GROUP BY r.realtor_id, r.name, r.company
ORDER BY "Property Count" DESC
);

-- List realtors with some useful aggregate data. This is like the "realtor stats" handler. This
-- lets us do more aggregation on the backend and reduce bandwidth.
SELECT name, company, property_count, avg_price, median_price, zipcodes
FROM (
	SELECT
		rp.name, rp.company,
		COUNT(*) AS "property_count",
		AVG(rp.price) AS "avg_price",
		PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY rp.price) AS "median_price",
		STRING_AGG(DISTINCT rp.zipcode, ',')::TEXT AS "zipcodes"
	FROM (
		SELECT pp.property_id, pp.listing_id, price, url, zipcode, city, state, location, last_scrape_ts, last_scrape_status, last_scrape_metadata, rpt.realtor_id, rpt.property_id, rpt.listing_id, r.realtor_id, name, company
		FROM property_price pp
		LEFT JOIN realtor_property_through rpt ON pp.property_id = rpt.property_id AND pp.listing_id = rpt.listing_id
		LEFT JOIN realtor r ON rpt.realtor_id = r.realtor_id
		WHERE r.name IS NOT NULL AND r.company IS NOT NULL
	) rp
	GROUP BY rp.name, rp.company
) AS rs
WHERE FALSE
ORDER BY rs.property_count DESC
LIMIT 100;



SELECT * FROM property p 
WHERE property_id =123
LIMIT 10;

UPDATE property SET last_scrape_status = 'good' WHERE property_id  = 123;







