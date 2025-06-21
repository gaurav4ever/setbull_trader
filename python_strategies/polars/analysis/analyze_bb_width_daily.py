import polars as pl
import argparse
import mysql.connector
import pandas as pd

def analyze_bb_squeeze(db_connection, instrument_key: str, bb_period: int, bb_std_dev: float, lookback_period: int, confirmation_period: int):
    """
    Analyzes daily data for a given instrument to find Bollinger Band Squeeze patterns.

    A "squeeze" is identified when:
    1. The current Bollinger Band Width (BBW) is the lowest it has been over a long lookback period.
    2. The BBW has been contracting or moving sideways over a shorter confirmation period.
    """

    # --- LOAD DATA FROM DATABASE ---
    try:
        # Fetch daily data. We also fetch bb_width to check if it's pre-calculated.
        query = """
        SELECT timestamp AS date, open, high, low, close, bb_width
        FROM stock_candle_data
        WHERE instrument_key = %s
          AND time_interval = 'day'
        ORDER BY timestamp ASC
        """
        df_pandas = pd.read_sql(query, db_connection, params=(instrument_key,))
        daily_df = pl.from_pandas(df_pandas)

    except Exception as e:
        print(f"Error fetching data from database: {e}")
        return

    if daily_df.is_empty():
        print(f"No daily data found for instrument '{instrument_key}' in the database.")
        return

    # --- CALCULATE BBW IF NOT PRESENT ---
    # Check if 'bb_width' column has no valid data.
    if daily_df["bb_width"].is_null().all():
        print("BBW data not found or is all null. Calculating it on the fly...")
        if len(daily_df) < bb_period:
            print(f"Not enough daily data ({len(daily_df)}) to calculate Bollinger Bands with period {bb_period}.")
            return

        daily_df = daily_df.with_columns(
            bb_mid=pl.col("close").rolling_mean(bb_period),
            bb_std=pl.col("close").rolling_std(bb_period),
        ).with_columns(
            bb_upper=pl.col("bb_mid") + bb_std_dev * pl.col("bb_std"),
            bb_lower=pl.col("bb_mid") - bb_std_dev * pl.col("bb_std"),
        ).with_columns(
            bb_width=((pl.col("bb_upper") - pl.col("bb_lower")) / pl.col("bb_mid"))
        ).drop_nulls("bb_width")

        if daily_df.is_empty():
            print(f"Not enough data to calculate BBW for '{instrument_key}' after dropping nulls.")
            return
    else:
        print("Found pre-calculated BBW data in the database.")
        daily_df = daily_df.drop_nulls("bb_width")
        
    if daily_df.is_empty():
        print(f"No valid BBW data available for '{instrument_key}'.")
        return

    # --- FILTER OUT ZERO BB_WIDTH VALUES ---
    daily_df = daily_df.filter(pl.col("bb_width") > 0)

    if daily_df.is_empty():
        print(f"No data with non-zero BBW available for '{instrument_key}'.")
        return

    if len(daily_df) < lookback_period:
        print(f"Not enough data ({len(daily_df)} days) for lookback period of {lookback_period} days.")
        return

    # --- SQUEEZE CONDITION: BBW is at its lowest point over the lookback period ---
    daily_df = daily_df.with_columns(
        is_squeeze=(pl.col("bb_width") == pl.col("bb_width").rolling_min(lookback_period))
    )

    # --- CONFIRMATION: BBW has been contracting or sideways ---
    daily_df = daily_df.with_columns(
        is_contracting=(pl.col("bb_width").diff(1).fill_null(0) <= 0).rolling_min(confirmation_period)
    )

    # --- FILTER FOR SQUEEZE SIGNALS ---
    squeeze_signals = daily_df.filter(
        pl.col("is_squeeze") & pl.col("is_contracting")
    )

    # --- DISPLAY RESULTS ---
    print(f"\n{'='*20} Analysis for: {instrument_key} {'='*20}")
    print(f"Configuration: BB({bb_period}, {bb_std_dev}), Lookback: {lookback_period}, Contraction: {confirmation_period} days")
    print(f"Analyzed {len(daily_df)} days of data.")

    if not squeeze_signals.is_empty():
        print(f"\n--- Found {len(squeeze_signals)} Squeeze Signals ---")
        print(squeeze_signals.select(["date", "open", "high", "low", "close", "bb_width"]))
    else:
        print("\n--- No Squeeze Signals Found ---")

    # --- LATEST DAY STATUS ---
    latest_day = daily_df.tail(1)
    if not latest_day.is_empty():
        print("\n--- Latest Day Status ---")
        print(latest_day.select(["date", "close", "bb_width", "is_squeeze", "is_contracting"]))
        
        if latest_day.select("is_squeeze").item() and latest_day.select("is_contracting").item():
            print("\n>> ALERT: Volatility Squeeze DETECTED for the most recent day! <<")
        else:
            print("\n>> No active squeeze for the most recent day.")

def main():
    """Main function to parse arguments and run the analysis."""
    parser = argparse.ArgumentParser(
        description="Analyze Bollinger Band Squeeze on daily data for a specific instrument from the database.",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter
    )
    parser.add_argument("instrument_key", type=str, help="Instrument key to analyze (e.g., 'NSE_EQ|INE848E01016').")
    
    # Parameters from the strategy document
    parser.add_argument("--bb-period", type=int, default=20, help="Bollinger Bands period (used if not pre-calculated).")
    parser.add_argument("--bb-std", type=float, default=2.0, help="Bollinger Bands standard deviations (used if not pre-calculated).")
    parser.add_argument("--lookback", type=int, default=126, help="Lookback period for squeeze detection (approx. 6 months).")
    parser.add_argument("--contraction", type=int, default=5, help="Confirmation period for band contraction.")
    
    args = parser.parse_args()
    
    db_config = {
        'host': '127.0.0.1',
        'port': 3306,
        'user': 'root',
        'password': 'root1234',
        'database': 'setbull_trader'
    }
    
    db_connection = None
    try:
        db_connection = mysql.connector.connect(**db_config)
        print("Successfully connected to database.")
        
        analyze_bb_squeeze(
            db_connection=db_connection,
            instrument_key=args.instrument_key,
            bb_period=args.bb_period,
            bb_std_dev=args.bb_std,
            lookback_period=args.lookback,
            confirmation_period=args.contraction
        )
    except mysql.connector.Error as err:
        print(f"Database connection failed: {err}")
    except Exception as e:
        print(f"An error occurred: {e}")
    finally:
        if db_connection and db_connection.is_connected():
            db_connection.close()
            print("Database connection closed.")


if __name__ == "__main__":
    main() 