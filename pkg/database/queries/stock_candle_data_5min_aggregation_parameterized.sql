-- 5-Minute Aggregation Query for Stock Candle Data (Parameterized Version)
-- This query aggregates 1-minute data into 5-minute candles with technical indicators
-- Compatible with MySQL ONLY_FULL_GROUP_BY mode
-- Parameters: ? = instrument_key, ? = start_timestamp, ? = end_timestamp

-- Step 1: Create 5-minute buckets and get basic aggregations
WITH
    five_min_basic AS (
        SELECT
            instrument_key,
            FLOOR(
                UNIX_TIMESTAMP(timestamp) / 300
            ) * 300 AS bucket_timestamp,
            MIN(timestamp) AS period_start,
            MAX(timestamp) AS period_end,
            MAX(high) AS high,
            MIN(low) AS low,
            SUM(volume) AS volume,
            SUM(open_interest) AS open_interest,
            SUM(close * volume) / SUM(volume) AS vwap_5min,
            COUNT(*) AS candle_count
        FROM stock_candle_data
        WHERE
            instrument_key = ?
            AND timestamp >= ?
            AND timestamp <= ?
            AND time_interval = '1minute'
        GROUP BY
            instrument_key,
            FLOOR(
                UNIX_TIMESTAMP(timestamp) / 300
            ) * 300
    ),
    -- Step 2: Get the first open price for each bucket
    five_min_open AS (
        SELECT f.instrument_key, f.bucket_timestamp, f.period_start, f.period_end, f.high, f.low, f.volume, f.open_interest, f.vwap_5min, f.candle_count, s.open
        FROM
            five_min_basic f
            JOIN stock_candle_data s ON s.instrument_key = f.instrument_key
            AND s.timestamp = f.period_start
            AND s.time_interval = '1minute'
    ),
    -- Step 3: Get the last close price and technical indicators for each bucket
    five_min_complete AS (
        SELECT
            f.instrument_key,
            f.bucket_timestamp,
            f.period_start,
            f.period_end,
            f.open,
            f.high,
            f.low,
            f.volume,
            f.open_interest,
            f.vwap_5min,
            f.candle_count,
            s.close,
            s.ma_9,
            s.bb_upper,
            s.bb_middle,
            s.bb_lower,
            s.vwap,
            s.ema_5,
            s.ema_9,
            s.ema_50,
            s.atr,
            s.rsi,
            s.bb_width,
            s.lowest_bb_width
        FROM
            five_min_open f
            JOIN stock_candle_data s ON s.instrument_key = f.instrument_key
            AND s.timestamp = f.period_end
            AND s.time_interval = '1minute'
    )
SELECT
    instrument_key,
    period_start AS timestamp,
    '5minute' AS time_interval,
    open,
    high,
    low,
    close,
    volume,
    open_interest,
    vwap_5min,
    ma_9,
    bb_upper,
    bb_middle,
    bb_lower,
    vwap,
    ema_5,
    ema_9,
    ema_50,
    atr,
    rsi,
    bb_width,
    lowest_bb_width,
    candle_count,
    period_start,
    period_end,
    -- Calculate additional derived metrics
    (high - low) AS range_,
    (close - open) AS change_,
    CASE
        WHEN open > 0 THEN ((close - open) / open) * 100
        ELSE 0
    END AS change_percent,
    -- Volume analysis
    CASE
        WHEN volume > 0 THEN (close * volume) / volume
        ELSE 0
    END AS volume_weighted_price,
    -- Bollinger Band analysis
    CASE
        WHEN bb_middle > 0 THEN (
            (close - bb_middle) / bb_middle
        ) * 100
        ELSE 0
    END AS bb_position_percent,
    CASE
        WHEN bb_upper > bb_lower THEN (
            (close - bb_lower) / (bb_upper - bb_lower)
        ) * 100
        ELSE 50
    END AS bb_percentile
FROM five_min_complete
ORDER BY period_start;