-- Create a temporary table with all the candles grouped by day
CREATE TEMPORARY TABLE temp_daily_candles AS
SELECT 
    instrument_key,
    DATE(timestamp) AS interval_date,
    MIN(timestamp) AS first_timestamp,
    MAX(timestamp) AS last_timestamp,
    MAX(high) AS high_price,
    MIN(low) AS low_price,
    SUM(volume) AS total_volume
FROM 
    stock_candle_data
WHERE 
    time_interval = '1minute'
GROUP BY 
    instrument_key, DATE(timestamp);

-- Get the open price from the first candle of each day
CREATE TEMPORARY TABLE temp_open_prices AS
SELECT 
    t.instrument_key,
    t.interval_date,
    scd.open AS open_price
FROM 
    temp_daily_candles t
JOIN 
    stock_candle_data scd ON scd.instrument_key = t.instrument_key 
        AND scd.timestamp = t.first_timestamp;

-- Get the close price from the last candle of each day
CREATE TEMPORARY TABLE temp_close_prices AS
SELECT 
    t.instrument_key,
    t.interval_date,
    scd.close AS close_price,
    scd.open_interest AS open_interest
FROM 
    temp_daily_candles t
JOIN 
    stock_candle_data scd ON scd.instrument_key = t.instrument_key 
        AND scd.timestamp = t.last_timestamp;

-- Join all temporary tables to get the final result
SELECT 
    t.instrument_key,
    t.interval_date,
    o.open_price,
    t.high_price,
    t.low_price,
    c.close_price,
    t.total_volume,
    c.open_interest,
    'day' AS time_interval
FROM 
    temp_daily_candles t
JOIN 
    temp_open_prices o ON t.instrument_key = o.instrument_key 
        AND t.interval_date = o.interval_date
JOIN 
    temp_close_prices c ON t.instrument_key = c.instrument_key 
        AND t.interval_date = c.interval_date
ORDER BY 
    t.instrument_key, t.interval_date;

-- Clean up temporary tables
DROP TEMPORARY TABLE IF EXISTS temp_daily_candles;
DROP TEMPORARY TABLE IF EXISTS temp_open_prices;
DROP TEMPORARY TABLE IF EXISTS temp_close_prices;