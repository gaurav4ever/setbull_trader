-- Query to find instrument keys with less than 95 daily candle records
-- This query counts daily interval records per instrument and filters those with count < 95

SELECT 
    instrument_key,
    COUNT(*) as daily_record_count
FROM stock_candle_data 
WHERE time_interval = 'day'
GROUP BY instrument_key
HAVING COUNT(*) < 95
ORDER BY daily_record_count ASC, instrument_key ASC;

-- Alternative query with more details (including date range)
-- Uncomment below if you want to see the date range for each instrument

/*
SELECT 
    instrument_key,
    COUNT(*) as daily_record_count,
    MIN(DATE(timestamp)) as earliest_date,
    MAX(DATE(timestamp)) as latest_date,
    DATEDIFF(MAX(DATE(timestamp)), MIN(DATE(timestamp))) + 1 as date_span_days
FROM stock_candle_data 
WHERE time_interval = 'day'
GROUP BY instrument_key
HAVING COUNT(*) < 95
ORDER BY daily_record_count ASC, instrument_key ASC;
*/

-- Query to get just the instrument keys (for easy copying/processing)
-- Uncomment below if you only need the instrument key list

/*
SELECT instrument_key
FROM stock_candle_data 
WHERE time_interval = 'day'
GROUP BY instrument_key
HAVING COUNT(*) < 95
ORDER BY instrument_key ASC;
*/
