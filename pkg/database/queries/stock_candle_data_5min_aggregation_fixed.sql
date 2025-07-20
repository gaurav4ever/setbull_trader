-- 5-Minute Aggregation Query with Corrected Technical Indicators
-- This query properly calculates BB indicators for 5-minute periods instead of copying from 1-minute data
-- Compatible with MySQL ONLY_FULL_GROUP_BY mode

WITH
    -- Step 1: Create 5-minute buckets and aggregate OHLCV data
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
            instrument_key = 'NSE_EQ|INE301A01014'
            AND timestamp >= '2025-07-18 09:15:00'
            AND timestamp <= '2025-07-18 15:30:00'
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
    -- Step 3: Get the last close price for each bucket
    five_min_close AS (
        SELECT f.instrument_key, f.bucket_timestamp, f.period_start, f.period_end, f.open, f.high, f.low, f.volume, f.open_interest, f.vwap_5min, f.candle_count, s.close
        FROM
            five_min_open f
            JOIN stock_candle_data s ON s.instrument_key = f.instrument_key
            AND s.timestamp = f.period_end
            AND s.time_interval = '1minute'
    ),
    -- Step 4: Calculate 5-minute Bollinger Bands using window functions
    five_min_bb AS (
        SELECT
            *,
            -- Calculate 20-period moving average for BB middle
            AVG(close) OVER (
                ORDER BY
                    bucket_timestamp ROWS BETWEEN 19 PRECEDING
                    AND CURRENT ROW
            ) AS bb_middle_5min,
            -- Calculate 20-period standard deviation for BB bands
            STDDEV(close) OVER (
                ORDER BY
                    bucket_timestamp ROWS BETWEEN 19 PRECEDING
                    AND CURRENT ROW
            ) AS bb_std_5min
        FROM five_min_close
    ),
    -- Step 5: Calculate final BB indicators
    five_min_final AS (
        SELECT
            instrument_key,
            bucket_timestamp,
            period_start,
            period_end,
            open,
            high,
            low,
            close,
            volume,
            open_interest,
            vwap_5min,
            candle_count,
            bb_middle_5min,
            bb_std_5min,
            -- Calculate BB upper band (middle + 2*std)
            CASE
                WHEN bb_middle_5min IS NOT NULL
                AND bb_std_5min IS NOT NULL THEN bb_middle_5min + (2 * bb_std_5min)
                ELSE NULL
            END AS bb_upper_5min,
            -- Calculate BB lower band (middle - 2*std)
            CASE
                WHEN bb_middle_5min IS NOT NULL
                AND bb_std_5min IS NOT NULL THEN bb_middle_5min - (2 * bb_std_5min)
                ELSE NULL
            END AS bb_lower_5min,
            -- Calculate BB width percentage
            CASE
                WHEN bb_middle_5min IS NOT NULL
                AND bb_std_5min IS NOT NULL
                AND bb_middle_5min > 0 THEN (
                    (
                        bb_middle_5min + (2 * bb_std_5min)
                    ) - (
                        bb_middle_5min - (2 * bb_std_5min)
                    )
                ) / bb_middle_5min * 100
                ELSE NULL
            END AS bb_width_5min
        FROM five_min_bb
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
    vwap_5min AS vwap,
    -- Use calculated 5-minute BB indicators instead of 1-minute ones
    bb_upper_5min AS bb_upper,
    bb_middle_5min AS bb_middle,
    bb_lower_5min AS bb_lower,
    bb_width_5min AS bb_width,
    -- Additional derived metrics
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
        WHEN bb_middle_5min > 0 THEN (
            (close - bb_middle_5min) / bb_middle_5min
        ) * 100
        ELSE 0
    END AS bb_position_percent,
    CASE
        WHEN bb_upper_5min > bb_lower_5min THEN (
            (close - bb_lower_5min) / (bb_upper_5min - bb_lower_5min)
        ) * 100
        ELSE 50
    END AS bb_percentile,
    -- Data quality indicators
    candle_count,
    period_start,
    period_end,
    -- Validation flags
    CASE
        WHEN candle_count < 5 THEN 'INCOMPLETE_PERIOD'
        WHEN candle_count > 5 THEN 'EXTRA_CANDLES'
        ELSE 'COMPLETE_PERIOD'
    END AS period_status,
    CASE
        WHEN bb_width_5min IS NULL THEN 'MISSING_BB_DATA'
        WHEN bb_width_5min < 0 THEN 'INVALID_BB_WIDTH'
        ELSE 'VALID_BB_DATA'
    END AS bb_data_status
FROM five_min_final
WHERE
    -- Filter out incomplete periods at the beginning (need 20 periods for BB calculation)
    bucket_timestamp >= (
        SELECT MIN(bucket_timestamp) + (19 * 300)
        FROM five_min_basic
    )
ORDER BY period_start;