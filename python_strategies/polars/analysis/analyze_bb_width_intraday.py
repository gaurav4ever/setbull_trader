import polars as pl
import numpy as np
from datetime import datetime, timedelta

# --- CONFIGURABLE PARAMETERS ---
# Path to your CSV with multiple instruments.
# Using the sample file which is assumed to have an 'instrument_key' column.
CSV_PATH = "/Users/gaurav/setbull_projects/setbull_trader/python_strategies/polars/analysis/top_3_bullish/top_3_bullish.csv"
N_DAYS = 20  # Number of days to consider for percentile calculation
MARKET_START = "09:15"
MARKET_END = "15:30"
TIMEZONE = "Asia/Kolkata"  # Not used directly, but for reference

# --- LOAD DATA ---
try:
    df = pl.read_csv(CSV_PATH, try_parse_dates=True)
except FileNotFoundError:
    print(f"Error: The file '{CSV_PATH}' was not found.")
    exit()

print("Columns in loaded CSV:", df.columns)

# --- ASSUMPTIONS ON COLUMN NAMES ---
TIME_COL = "timestamp"
OPEN_COL = "open"
HIGH_COL = "high"
LOW_COL = "low"
CLOSE_COL = "close"
INSTRUMENT_COL = "instrument_key"

# --- VALIDATE REQUIRED COLUMNS EXIST ---
required_cols = [TIME_COL, OPEN_COL, HIGH_COL, LOW_COL, CLOSE_COL, INSTRUMENT_COL]
for col in required_cols:
    if col not in df.columns:
        raise ValueError(f"Required column '{col}' not found in CSV. Available columns: {df.columns}")

# --- PROCESS EACH INSTRUMENT INDIVIDUALLY ---
# The correct way to iterate over groups in Polars is to iterate directly over the group_by object.
for instrument_key, instrument_df in df.group_by(INSTRUMENT_COL):
    print(f"\n{'='*20} Processing Instrument: {instrument_key} {'='*20}")

    # --- FILTER FOR MARKET HOURS (IST 9:15 to 15:30) ---
    market_hours_df = instrument_df.filter(
        pl.col(TIME_COL).dt.time().is_between(
            datetime.strptime(MARKET_START, "%H:%M").time(),
            datetime.strptime(MARKET_END, "%H:%M").time()
        )
    )

    if market_hours_df.is_empty():
        print(f"No data for '{instrument_key}' within market hours. Skipping.")
        continue

    # --- AGGREGATE TO 5-MINUTE CANDLES ---
    # We apply the time aggregation within each instrument's sub-dataframe
    grouped = market_hours_df.group_by(
        pl.col(TIME_COL).dt.truncate("5m"), maintain_order=True
    ).agg(
        pl.col(OPEN_COL).first().alias("open"),
        pl.col(HIGH_COL).max().alias("high"),
        pl.col(LOW_COL).min().alias("low"),
        pl.col(CLOSE_COL).last().alias("close"),
    ).rename({TIME_COL: "dt_5min"})

    # --- ADD DATE COLUMN FOR DAY SPLITTING ---
    grouped = grouped.with_columns(
        pl.col("dt_5min").dt.date().alias("date")
    )

    # --- FILTER LAST N DAYS ---
    all_dates = grouped["date"].unique().sort()
    if all_dates.is_empty():
        print(f"No data for '{instrument_key}' after aggregation. Skipping.")
        continue
    
    # If less than N_DAYS are available, use all available days
    if len(all_dates) < N_DAYS:
        print(f"Warning for '{instrument_key}': Not enough days in data. Found {len(all_dates)}, using all of them.")
        last_n_dates = all_dates
    else:
        last_n_dates = all_dates.tail(N_DAYS)

    analysis_df = grouped.filter(pl.col("date").is_in(last_n_dates))

    # --- CALCULATE BB WIDTH (20-period, 2 std) ---
    WINDOW = 20
    STD_DEV = 2

    if len(analysis_df) < WINDOW:
        print(f"Not enough data points ({len(analysis_df)}) for '{instrument_key}' to calculate rolling window of size {WINDOW}. Skipping.")
        continue

    # Calculate Bollinger Bands and Width
    analysis_df = analysis_df.with_columns(
        bb_mid=pl.col("close").rolling_mean(WINDOW),
        bb_std=pl.col("close").rolling_std(WINDOW),
    ).with_columns(
        bb_upper=pl.col("bb_mid") + STD_DEV * pl.col("bb_std"),
        bb_lower=pl.col("bb_mid") - STD_DEV * pl.col("bb_std"),
    ).with_columns(
        bb_width=((pl.col("bb_upper") - pl.col("bb_lower")) / pl.col("close"))
    ).drop_nulls("bb_width")

    if analysis_df.is_empty():
        print(f"No data available for '{instrument_key}' to calculate percentiles after dropping nulls. Skipping.")
        continue

    # --- CALCULATE DAILY PERCENTILES ---
    daily_stats = analysis_df.group_by("date", maintain_order=True).agg(
        p10_bb_width=pl.col("bb_width").quantile(0.10),
        p95_bb_width=pl.col("bb_width").quantile(0.95)
    )

    # --- GET TOP 5 DAYS FOR LOW AND HIGH BB_WIDTH PERCENTILES ---
    top_5_p10 = daily_stats.sort("p10_bb_width").head(5)
    top_5_p95 = daily_stats.sort("p95_bb_width", descending=True).head(5)

    # --- DISPLAY RESULTS IN TABLES ---
    print(f"\n--- Top 5 Days with Lowest 10th Percentile BB_Width (Contraction) for {instrument_key} ---")
    print(top_5_p10)

    print(f"\n--- Top 5 Days with Highest 95th Percentile BB_Width (Expansion) for {instrument_key} ---")
    print(top_5_p95)
