import polars as pl
import argparse
import mysql.connector
import pandas as pd
from tqdm import tqdm

def get_all_instrument_keys(db_connection):
    """Fetches all unique daily instrument keys and their symbols from the database, filtering out ETFs and non-stock instruments."""
    print("Fetching all instrument keys and symbols...")
    try:
        # Join with stock_universe to get the symbol and name
        query = """
        SELECT DISTINCT scd.instrument_key, su.symbol, su.name
        FROM stock_candle_data scd
        JOIN stock_universe su ON scd.instrument_key = su.instrument_key
        WHERE scd.time_interval = 'day'
          AND su.active = TRUE
        """
        df_pandas = pd.read_sql(query, db_connection)

        # Comprehensive filtering for ETFs and non-stock instruments
        initial_count = len(df_pandas)
        exclusions = [
            'LIQUID', 'ETF', 'BEES', 'NIFTY', 'BANKNIFTY', 'FINNIFTY',
            'SENSEX', 'TOP100', 'TOP50', 'TOP200', 'TOP500',
            'INDEX', 'INDIA', 'GOLD', 'SILVER', 'COPPER', 'CRUDE',
            'USDINR', 'EURINR', 'GBPINR', 'JPYINR',
            'GOVT', 'CORP', 'POWERGRID', 'ONGC', 'COALINDIA',
            'MUTUAL', 'FUND', 'BOND', 'DEBT', 'MONEY',
            'LIQUIDBEES', 'LIQUIDETF', 'LIQUIDFUND',
            'NIFTYBEES', 'BANKBEES', 'GOLDBEES',
            'JUNIOR', 'SMALL', 'MID', 'LARGE', 'MULTI',
            'CONSUMPTION', 'ENERGY', 'FINANCIAL', 'HEALTHCARE',
            'INDUSTRIAL', 'MATERIALS', 'REALESTATE', 'TECHNOLOGY',
            'UTILITIES', 'COMMUNICATION', 'CONSUMER', 'DISCRETIONARY'
        ]
        
        filtered_reasons = {}
        
        for exclusion in exclusions:
            # Check both symbol and name columns
            symbol_matches = df_pandas[df_pandas['symbol'].str.contains(exclusion, case=False, na=False)]
            name_matches = df_pandas[df_pandas['name'].str.contains(exclusion, case=False, na=False)]
            
            if len(symbol_matches) > 0 or len(name_matches) > 0:
                filtered_reasons[exclusion] = len(symbol_matches) + len(name_matches)
            
            # Filter out matches from both symbol and name
            df_pandas = df_pandas[~df_pandas['symbol'].str.contains(exclusion, case=False, na=False)]
            df_pandas = df_pandas[~df_pandas['name'].str.contains(exclusion, case=False, na=False)]
        
        # Additional filtering for common patterns
        # Remove instruments with very short symbols (likely indices)
        df_pandas = df_pandas[df_pandas['symbol'].str.len() >= 3]
        
        # Remove instruments with all caps and common index patterns
        df_pandas = df_pandas[~df_pandas['symbol'].str.match(r'^[A-Z]{2,5}$')]  # Remove 2-5 letter all caps
        
        # Remove instruments with numbers only
        df_pandas = df_pandas[~df_pandas['symbol'].str.match(r'^\d+$')]
        
        # Remove instruments with special characters (except dots and dashes)
        df_pandas = df_pandas[~df_pandas['symbol'].str.contains(r'[^A-Za-z0-9.-]')]

        filtered_count = len(df_pandas)
        total_filtered = initial_count - filtered_count
        
        print(f"Data sanitization results:")
        print(f"  Initial instruments: {initial_count}")
        print(f"  Filtered out: {total_filtered}")
        print(f"  Remaining instruments: {filtered_count}")
        
        if filtered_reasons:
            print("  Filtering breakdown:")
            for reason, count in filtered_reasons.items():
                if count > 0:
                    print(f"    {reason}: {count} instruments")

        keys = df_pandas.to_dict('records')
        print(f"Found {len(keys)} unique instruments for analysis.")
        return keys
    except Exception as e:
        print(f"Error fetching instrument keys: {e}")
        return []

def analyze_instrument_volatility(db_connection, instrument_key: str, symbol: str, bb_period: int, bb_std_dev: float, lookback_period: int, check_period: int):
    """
    Analyzes a single instrument's daily data to find low Bollinger Band Width.
    Returns metadata if the instrument is currently in a low volatility state.
    """
    try:
        query = """
        SELECT timestamp AS date, close, volume
        FROM stock_candle_data
        WHERE instrument_key = %s
          AND time_interval = 'day'
        ORDER BY timestamp ASC
        """
        df_pandas = pd.read_sql(query, db_connection, params=(instrument_key,))
        if df_pandas.empty:
            return None
        daily_df = pl.from_pandas(df_pandas)
    except Exception as e:
        # Log error for specific instrument and continue
        # print(f"Could not process {instrument_key}: {e}")
        return None

    # Need enough data for the initial BB calculation
    if len(daily_df) < bb_period or len(daily_df) < 50:
        return None

    # Calculate Bollinger Bands and BBW
    daily_df = daily_df.with_columns(
        bb_mid=pl.col("close").rolling_mean(bb_period),
        bb_std=pl.col("close").rolling_std(bb_period),
    ).with_columns(
        bb_upper=pl.col("bb_mid") + bb_std_dev * pl.col("bb_std"),
        bb_lower=pl.col("bb_mid") - bb_std_dev * pl.col("bb_std"),
    ).with_columns(
        bb_width=((pl.col("bb_upper") - pl.col("bb_lower")) / pl.col("bb_mid"))
    ).drop_nulls(["bb_width", "volume", "bb_upper", "bb_lower"])

    # Filter out any non-positive BBW values
    daily_df = daily_df.filter(pl.col("bb_width") > 0)

    # Need enough data for the lookback period
    if len(daily_df) < lookback_period or len(daily_df) < 50:
        return None

    # Use the last `lookback_period` of data to establish the percentile.
    lookback_df = daily_df.tail(lookback_period)
    
    # Calculate 10th percentile threshold from the lookback data
    percentile_10_threshold = lookback_df.select(pl.col("bb_width").quantile(0.10)).item()

    # Calculate average BBW over the lookback period
    avg_bb_width_lookback = lookback_df.select(pl.col("bb_width").mean()).item()

    if percentile_10_threshold is None:
        return None

    # Check the last `check_period` days for a squeeze signal
    recent_days_df = daily_df.tail(check_period)
    
    # Filter for days where BBW is in the 10th percentile
    low_vol_days = recent_days_df.filter(pl.col("bb_width") <= percentile_10_threshold)

    if not low_vol_days.is_empty():
        latest_day = daily_df.tail(1)
        latest_close = latest_day.select("close").item()
        latest_bb_width = latest_day.select("bb_width").item()
        latest_upper_band = latest_day.select("bb_upper").item()
        latest_lower_band = latest_day.select("bb_lower").item()
        latest_volume = latest_day.select("volume").item()

        # Squeeze Tightness Score
        squeeze_ratio = latest_bb_width / avg_bb_width_lookback if avg_bb_width_lookback else None

        # Volume Contraction Ratio
        last_5_vol = daily_df.tail(5).select(pl.col("volume")).to_series().mean()
        last_50_vol = daily_df.tail(50).select(pl.col("volume")).to_series().mean()
        volume_ratio = last_5_vol / last_50_vol if last_50_vol else None

        # Breakout Readiness Score
        if (latest_upper_band - latest_lower_band) != 0:
            breakout_readiness = (latest_close - latest_lower_band) / (latest_upper_band - latest_lower_band)
        else:
            breakout_readiness = None

        return {
            "instrument_key": instrument_key,
            "symbol": symbol,
            "latest_date": latest_day.select("date").item(),
            "latest_close": latest_close,
            "latest_bb_width": latest_bb_width,
            "10_percentile_threshold": percentile_10_threshold,
            "avg_bb_width_lookback": avg_bb_width_lookback,
            "squeeze_ratio": squeeze_ratio,
            "volume_ratio": volume_ratio,
            "breakout_readiness": breakout_readiness
        }
        
    return None

def main():
    """Main function to parse arguments and run the analysis for all stocks."""
    parser = argparse.ArgumentParser(
        description="Find stocks with low volatility based on Bollinger Band Width across all instruments.",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter
    )
    
    parser.add_argument("--bb-period", type=int, default=20, help="Bollinger Bands period.")
    parser.add_argument("--bb-std", type=float, default=2.0, help="Bollinger Bands standard deviations.")
    parser.add_argument("--lookback", type=int, default=100, help="Historical lookback period (days) for percentile calculation (approx. 1 year).")
    parser.add_argument("--check-days", type=int, default=5, help="Number of recent days to check for low volatility.")
    parser.add_argument("--output-file", type=str, default="low_volatility_stocks.csv", help="Name of the output CSV file.")
    
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
        print("Successfully connected to the database.")
        
        instrument_keys = get_all_instrument_keys(db_connection)
        
        if not instrument_keys:
            print("No instrument keys with daily data found. Exiting.")
            return

        low_volatility_stocks = []
        
        # Use tqdm for a progress bar as this may take time
        for stock in tqdm(instrument_keys, desc="Analyzing stocks"):
            result = analyze_instrument_volatility(
                db_connection=db_connection,
                instrument_key=stock['instrument_key'],
                symbol=stock['symbol'],
                bb_period=args.bb_period,
                bb_std_dev=args.bb_std,
                lookback_period=args.lookback,
                check_period=args.check_days
            )
            if result:
                low_volatility_stocks.append(result)

        if not low_volatility_stocks:
            print("\nNo stocks found matching the low volatility criteria.")
            return
            
        # Create a Polars DataFrame and save to CSV
        results_df = pl.DataFrame(low_volatility_stocks)
        # Reorder columns for better readability
        results_df = results_df.select([
            "symbol", "instrument_key", "latest_date", "latest_close",
            "latest_bb_width", "10_percentile_threshold", "avg_bb_width_lookback",
            "squeeze_ratio", "volume_ratio", "breakout_readiness"
        ])
        # Sort by the lowest width to see the most compressed stocks first
        results_df = results_df.sort("latest_bb_width")
        
        results_df.write_csv(args.output_file)
        
        print(f"\nAnalysis complete. Found {len(low_volatility_stocks)} stocks with low volatility.")
        print(f"Results saved to '{args.output_file}'.")
        print("\nTop 10 stocks with the lowest BB Width:")
        print(results_df.head(10))

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