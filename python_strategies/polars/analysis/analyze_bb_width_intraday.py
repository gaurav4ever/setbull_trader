#!/usr/bin/env python3
"""
BB Width Intraday Analysis - Database-Driven Analyzer
=====================================================

This script analyzes Bollinger Band Width (BBW) for intraday data from the database,
identifying days with lowest BBW (contraction) and highest BBW (expansion).

Author: Gaurav Sharma - CEO, Setbull Trader
Version: 2.0.0
Date: 2025-01-30
"""

import polars as pl
import argparse
import mysql.connector
import pandas as pd
import logging
import time
import os
from datetime import datetime, timedelta
from typing import List, Dict, Optional, Tuple
from tqdm import tqdm
import warnings
warnings.filterwarnings('ignore')

# =============================================================================
# SECTION 1: CONFIGURATION & SETUP
# =============================================================================

class ConfigurationManager:
    """Manages all configuration parameters for the BB width analyzer."""
    
    def __init__(self):
        # Database Configuration
        self.db_config = {
            'host': '127.0.0.1',
            'port': 3306,
            'user': 'root',
            'password': 'root1234',
            'database': 'setbull_trader',
            'autocommit': True,
            'pool_size': 10,
            'pool_name': 'bbw_pool',
            'connection_timeout': 30
        }
        
        # Analysis Parameters
        self.analysis_params = {
            'bb_period': 20,                    # Bollinger Bands period
            'bb_std_dev': 2.0,                  # Bollinger Bands standard deviations
            'market_start': "09:15",            # Market start time
            'market_end': "15:30",              # Market end time
            'time_interval': '5m',              # Aggregation interval
            'min_data_points': 20,              # Minimum data points required
            'default_lookback_days': 20         # Default lookback period
        }
        
        # Performance Parameters
        self.performance_params = {
            'batch_size': 1000,                 # Batch processing size
            'chunk_size': 5000,                 # Memory chunk size
            'max_connections': 10,              # Maximum database connections
            'connection_timeout': 30            # Connection timeout (seconds)
        }
        
        # Output Configuration
        self.output_config = {
            'output_dir': 'output',
            'logs_dir': 'logs',
            'csv_filename': 'bb_width_analysis.csv'
        }

class DatabaseManager:
    """Manages database connections and operations."""
    
    def __init__(self, config: ConfigurationManager):
        self.config = config
        self.connection = None
        self.logger = logging.getLogger(__name__)
    
    def connect(self) -> bool:
        """Establish database connection with connection pooling."""
        try:
            self.connection = mysql.connector.connect(
                host=self.config.db_config['host'],
                port=self.config.db_config['port'],
                user=self.config.db_config['user'],
                password=self.config.db_config['password'],
                database=self.config.db_config['database'],
                autocommit=self.config.db_config['autocommit'],
                pool_name=self.config.db_config['pool_name'],
                pool_size=self.config.db_config['pool_size'],
                connection_timeout=self.config.db_config['connection_timeout']
            )
            self.logger.info("Successfully connected to database")
            return True
        except mysql.connector.Error as err:
            self.logger.error(f"Database connection failed: {err}")
            return False
    
    def disconnect(self):
        """Close database connection."""
        if self.connection and self.connection.is_connected():
            self.connection.close()
            self.logger.info("Database connection closed")
    
    def execute_query(self, query: str, params: tuple = None) -> Optional[pd.DataFrame]:
        """Execute a database query and return results as DataFrame."""
        try:
            if params:
                df = pd.read_sql(query, self.connection, params=params)
            else:
                df = pd.read_sql(query, self.connection)
            return df
        except Exception as e:
            self.logger.error(f"Query execution failed: {e}")
            return None

class LoggingManager:
    """Manages logging configuration and setup."""
    
    def __init__(self, config: ConfigurationManager):
        self.config = config
        self.setup_logging()
    
    def setup_logging(self):
        """Setup comprehensive logging configuration."""
        # Create logs directory if it doesn't exist
        log_dir = os.path.join(self.config.output_config['output_dir'], 
                              self.config.output_config['logs_dir'])
        os.makedirs(log_dir, exist_ok=True)
        
        # Generate log filename with timestamp
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        log_filename = os.path.join(log_dir, f"bb_width_analysis_{timestamp}.log")
        
        # Configure logging
        logging.basicConfig(
            level=logging.INFO,
            format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
            handlers=[
                logging.FileHandler(log_filename),
                logging.StreamHandler()
            ]
        )
        
        # Create logger instance
        self.logger = logging.getLogger(__name__)
        self.logger.info("Logging system initialized")

class PerformanceMonitor:
    """Monitors and tracks performance metrics."""
    
    def __init__(self):
        self.start_time = None
        self.metrics = {}
        self.logger = logging.getLogger(__name__)
    
    def start_timer(self, operation: str):
        """Start timing an operation."""
        self.start_time = time.time()
        self.logger.info(f"Starting operation: {operation}")
    
    def end_timer(self, operation: str) -> float:
        """End timing an operation and return duration."""
        if self.start_time:
            duration = time.time() - self.start_time
            self.metrics[operation] = duration
            self.logger.info(f"Completed {operation} in {duration:.2f} seconds")
            return duration
        return 0.0
    
    def get_metrics(self) -> Dict[str, float]:
        """Get all performance metrics."""
        return self.metrics

# =============================================================================
# SECTION 2: DATA LAYER
# =============================================================================

class DataValidator:
    """Validates data quality and completeness."""
    
    def __init__(self, config: ConfigurationManager):
        self.config = config
        self.logger = logging.getLogger(__name__)
    
    def validate_price_data(self, df: pl.DataFrame) -> bool:
        """Validate OHLC price data for logical consistency."""
        try:
            # Check for positive prices
            if df.filter(
                (pl.col("open") <= 0) | 
                (pl.col("high") <= 0) | 
                (pl.col("low") <= 0) | 
                (pl.col("close") <= 0)
            ).height > 0:
                self.logger.warning("Found non-positive price values")
                return False
            
            # Check for logical OHLC relationships
            invalid_ohlc = df.filter(
                (pl.col("high") < pl.col("low")) |
                (pl.col("high") < pl.col("open")) |
                (pl.col("high") < pl.col("close")) |
                (pl.col("low") > pl.col("open")) |
                (pl.col("low") > pl.col("close"))
            )
            
            if invalid_ohlc.height > 0:
                self.logger.warning("Found invalid OHLC relationships")
                return False
            
            return True
        except Exception as e:
            self.logger.error(f"Price validation failed: {e}")
            return False
    
    def check_data_completeness(self, df: pl.DataFrame, min_points: int) -> bool:
        """Check if data has minimum required points."""
        try:
            if df.height < min_points:
                self.logger.warning(f"Insufficient data: {df.height} points < {min_points} required")
                return False
            return True
        except Exception as e:
            self.logger.error(f"Data completeness check failed: {e}")
            return False

class DataFetcher:
    """Fetches and filters data from database."""
    
    def __init__(self, config: ConfigurationManager, db_manager: DatabaseManager):
        self.config = config
        self.db_manager = db_manager
        self.validator = DataValidator(config)
        self.logger = logging.getLogger(__name__)
    
    def get_all_instruments(self) -> List[Dict]:
        """Fetch all unique instruments with 1minute intraday data."""
        try:
            query = """
            SELECT DISTINCT scd.instrument_key, su.symbol, su.name
            FROM stock_candle_data scd
            LEFT JOIN stock_universe su ON scd.instrument_key = su.instrument_key
            WHERE scd.time_interval = '1minute'
            """
            
            df = self.db_manager.execute_query(query)
            if df is None or df.empty:
                self.logger.warning("No instruments found with 1minute intraday data")
                return []
            
            # Filter out null symbols
            df = df.dropna(subset=['symbol'])
            
            return df.to_dict('records')
        except Exception as e:
            self.logger.error(f"Error fetching instruments: {e}")
            return []
    
    def get_instrument_data(self, instrument_key: str, lookback_days: Optional[int] = None) -> Optional[pl.DataFrame]:
        """Fetch 1minute intraday data for a specific instrument."""
        try:
            # Build query with optional lookback
            if lookback_days:
                query = """
                SELECT timestamp, open, high, low, close, volume, time_interval
                FROM stock_candle_data
                WHERE instrument_key = %s
                  AND time_interval = '1minute'
                  AND timestamp >= DATE_SUB(NOW(), INTERVAL %s DAY)
                ORDER BY timestamp ASC
                """
                params = (instrument_key, lookback_days)
            else:
                query = """
                SELECT timestamp, open, high, low, close, volume, time_interval
                FROM stock_candle_data
                WHERE instrument_key = %s
                  AND time_interval = '1minute'
                ORDER BY timestamp ASC
                """
                params = (instrument_key,)
            
            df_pandas = self.db_manager.execute_query(query, params)
            if df_pandas is None or df_pandas.empty:
                return None
            
            # Convert to Polars DataFrame
            df = pl.from_pandas(df_pandas)
            
            # Apply data quality filters
            if not self._apply_data_filters(df):
                return None
            
            return df
        except Exception as e:
            self.logger.error(f"Error fetching data for {instrument_key}: {e}")
            return None
    
    def get_instruments_by_symbols(self, symbols: List[str], lookback_days: Optional[int] = None) -> List[Dict]:
        """Fetch instruments by symbol list (only with 1minute data)."""
        try:
            # First, let's check if the symbols exist in stock_universe table
            placeholders = ','.join(['%s'] * len(symbols))
            check_query = f"""
            SELECT symbol, instrument_key, name
            FROM stock_universe
            WHERE symbol IN ({placeholders})
            """
            
            check_df = self.db_manager.execute_query(check_query, symbols)
            if check_df is None or check_df.empty:
                self.logger.warning(f"No symbols found in stock_universe table: {symbols}")
                return []
            
            # Create placeholders for the IN clause
            placeholders = ','.join(['%s'] * len(symbols))
            
            # Build query with optional lookback
            if lookback_days:
                query = f"""
                SELECT DISTINCT scd.instrument_key, su.symbol, su.name
                FROM stock_candle_data scd
                LEFT JOIN stock_universe su ON scd.instrument_key = su.instrument_key
                WHERE scd.time_interval = '1minute'
                  AND su.symbol IN ({placeholders})
                  AND scd.timestamp >= DATE_SUB(NOW(), INTERVAL %s DAY)
                """
                params = symbols + [lookback_days]
            else:
                query = f"""
                SELECT DISTINCT scd.instrument_key, su.symbol, su.name
                FROM stock_candle_data scd
                LEFT JOIN stock_universe su ON scd.instrument_key = su.instrument_key
                WHERE scd.time_interval = '1minute'
                  AND su.symbol IN ({placeholders})
                """
                params = symbols
            
            df = self.db_manager.execute_query(query, params)
            if df is None or df.empty:
                self.logger.warning(f"No instruments found for symbols (with 1minute data): {symbols}")
                return []
            
            return df.to_dict('records')
        except Exception as e:
            self.logger.error(f"Error fetching instruments by symbols: {e}")
            return []
    
    def _apply_data_filters(self, df: pl.DataFrame) -> bool:
        """Apply data quality filters."""
        try:
            # Check minimum data requirements
            if not self.validator.check_data_completeness(df, self.config.analysis_params['min_data_points']):
                return False
            
            # Validate price data
            if not self.validator.validate_price_data(df):
                return False
            
            return True
        except Exception as e:
            self.logger.error(f"Data filtering failed: {e}")
            return False

# =============================================================================
# SECTION 3: ANALYSIS ENGINE
# =============================================================================

class BollingerBandCalculator:
    """Calculates Bollinger Bands and BB width for the given data."""
    
    def __init__(self, config: ConfigurationManager):
        self.config = config
        self.logger = logging.getLogger(__name__)
    
    def calculate_bollinger_bands(self, df: pl.DataFrame) -> pl.DataFrame:
        """Calculate Bollinger Bands and BB width for the given data."""
        try:
            bb_period = self.config.analysis_params['bb_period']
            bb_std_dev = self.config.analysis_params['bb_std_dev']
            
            # Calculate Bollinger Bands
            df = df.with_columns([
                pl.col("close").rolling_mean(bb_period).alias("bb_mid"),
                pl.col("close").rolling_std(bb_period).alias("bb_std")
            ]).with_columns([
                (pl.col("bb_mid") + bb_std_dev * pl.col("bb_std")).alias("bb_upper"),
                (pl.col("bb_mid") - bb_std_dev * pl.col("bb_std")).alias("bb_lower")
            ]).with_columns([
                (pl.col("bb_upper") - pl.col("bb_lower")).alias("bb_width")
            ])
            
            # Drop null values
            df = df.drop_nulls(["bb_width", "bb_upper", "bb_lower"])
            
            # Filter out non-positive BB width values
            df = df.filter(pl.col("bb_width") > 0)
            
            return df
        except Exception as e:
            self.logger.error(f"Bollinger Band calculation failed: {e}")
            return df

class IntradayAnalyzer:
    """Main analyzer that orchestrates the intraday analysis process."""
    
    def __init__(self, config: ConfigurationManager, db_manager: DatabaseManager):
        self.config = config
        self.db_manager = db_manager
        self.data_fetcher = DataFetcher(config, db_manager)
        self.bb_calculator = BollingerBandCalculator(config)
        self.logger = logging.getLogger(__name__)
    
    def analyze_instrument(self, instrument_key: str, symbol: str, lookback_days: Optional[int] = None) -> Optional[Dict]:
        """Analyze a single instrument for BB width patterns (strictly intraday)."""
        try:
            # Fetch instrument data (1minute only)
            df = self.data_fetcher.get_instrument_data(instrument_key, lookback_days)
            if df is None or df.is_empty():
                self.logger.warning(f"No 1minute data for {symbol} ({instrument_key}), skipping.")
                return None
            
            # Filter for market hours
            market_hours_df = self._filter_market_hours(df)
            if market_hours_df.is_empty():
                self.logger.warning(f"No market hours data for {symbol} ({instrument_key}), skipping.")
                return None
            
            # Aggregate to 5-minute candles
            aggregated_df = self._aggregate_to_5min(market_hours_df)
            if aggregated_df.is_empty():
                self.logger.warning(f"No 5-minute aggregated data for {symbol} ({instrument_key}), skipping.")
                return None
            
            return self._analyze_intraday_data(aggregated_df, instrument_key, symbol, lookback_days)
        except Exception as e:
            self.logger.error(f"Analysis failed for {symbol}: {e}")
            return None
    
    def analyze_multiple_instruments(self, instruments: List[Dict], lookback_days: Optional[int] = None) -> List[Dict]:
        """Analyze multiple instruments."""
        try:
            self.logger.info(f"Starting analysis of {len(instruments)} instruments")
            
            results = []
            for instrument in tqdm(instruments, desc="Analyzing instruments"):
                result = self.analyze_instrument(
                    instrument['instrument_key'], 
                    instrument['symbol'],
                    lookback_days
                )
                if result:
                    results.append(result)
            
            self.logger.info(f"Analysis complete. Processed {len(results)} instruments")
            return results
            
        except Exception as e:
            self.logger.error(f"Multiple instrument analysis failed: {e}")
            return []
    
    def _filter_market_hours(self, df: pl.DataFrame) -> pl.DataFrame:
        """Filter data for market hours only."""
        try:
            market_start = datetime.strptime(self.config.analysis_params['market_start'], "%H:%M").time()
            market_end = datetime.strptime(self.config.analysis_params['market_end'], "%H:%M").time()
            
            return df.filter(
                pl.col("timestamp").dt.time().is_between(market_start, market_end)
            )
        except Exception as e:
            self.logger.error(f"Market hours filtering failed: {e}")
            return df
    
    def _aggregate_to_5min(self, df: pl.DataFrame) -> pl.DataFrame:
        """Aggregate 1-minute data to 5-minute candles."""
        try:
            grouped = df.group_by(
                pl.col("timestamp").dt.truncate("5m"), maintain_order=True
            ).agg(
                pl.col("open").first().alias("open"),
                pl.col("high").max().alias("high"),
                pl.col("low").min().alias("low"),
                pl.col("close").last().alias("close"),
                pl.col("volume").sum().alias("volume")
            ).rename({"timestamp": "dt_5min"})
            
            # Add date column for day splitting
            grouped = grouped.with_columns(
                pl.col("dt_5min").dt.date().alias("date")
            )
            
            return grouped
        except Exception as e:
            self.logger.error(f"5-minute aggregation failed: {e}")
            return df
    
    def _calculate_daily_stats(self, df: pl.DataFrame) -> pl.DataFrame:
        """Calculate daily BB width statistics."""
        try:
            # Check if we have a 'date' column (from intraday aggregation) or need to extract from 'timestamp'
            if 'date' in df.columns:
                # Intraday data already has date column
                group_col = 'date'
            else:
                # Daily data - extract date from timestamp
                df = df.with_columns(pl.col("timestamp").dt.date().alias("date"))
                group_col = 'date'
            
            daily_stats = df.group_by(group_col, maintain_order=True).agg(
                p10_bb_width=pl.col("bb_width").quantile(0.10).round(2),
                p25_bb_width=pl.col("bb_width").quantile(0.25).round(2),
                p50_bb_width=pl.col("bb_width").quantile(0.50).round(2),
                p75_bb_width=pl.col("bb_width").quantile(0.75).round(2),
                p90_bb_width=pl.col("bb_width").quantile(0.90).round(2),
                p95_bb_width=pl.col("bb_width").quantile(0.95).round(2),
                mean_bb_width=pl.col("bb_width").mean().round(2),
                std_bb_width=pl.col("bb_width").std().round(2),
                min_bb_width=pl.col("bb_width").min().round(2),
                max_bb_width=pl.col("bb_width").max().round(2),
                data_points=pl.count()
            )
            
            return daily_stats
        except Exception as e:
            self.logger.error(f"Daily stats calculation failed: {e}")
            return df
    
    def _find_lowest_bb_day(self, daily_stats: pl.DataFrame) -> Dict:
        """Find the day with the lowest BB width."""
        try:
            if daily_stats.is_empty():
                return {}
            
            # Find day with lowest 10th percentile BB width
            lowest_p10 = daily_stats.sort("p10_bb_width").head(1)
            
            if lowest_p10.is_empty():
                return {}
            
            lowest_day = lowest_p10.to_dicts()[0]
            
            return {
                "date": lowest_day["date"],
                "p10_bb_width": lowest_day["p10_bb_width"],
                "mean_bb_width": lowest_day["mean_bb_width"],
                "min_bb_width": lowest_day["min_bb_width"],
                "max_bb_width": lowest_day["max_bb_width"],
                "data_points": lowest_day["data_points"]
            }
        except Exception as e:
            self.logger.error(f"Lowest BB day calculation failed: {e}")
            return {}
    
    def _analyze_intraday_data(self, df: pl.DataFrame, instrument_key: str, symbol: str, lookback_days: Optional[int] = None) -> Optional[Dict]:
        """Analyze intraday data (5-minute aggregated)."""
        try:
            # Calculate Bollinger Bands and BB width
            bb_df = self.bb_calculator.calculate_bollinger_bands(df)
            if bb_df.is_empty():
                return None
            
            # Calculate daily statistics
            daily_stats = self._calculate_daily_stats(bb_df)
            if daily_stats.is_empty():
                return None
            
            # Find lowest BB width day
            lowest_bb_day = self._find_lowest_bb_day(daily_stats)
            
            # Compile results
            result = {
                "instrument_key": instrument_key,
                "symbol": symbol,
                "analysis_date": datetime.now().isoformat(),
                "lookback_days": lookback_days or "ALL",
                "total_days_analyzed": len(daily_stats),
                "data_type": "intraday_5min",
                "lowest_bb_day": lowest_bb_day,
                "daily_stats": daily_stats.to_dicts()
            }
            
            return result
            
        except Exception as e:
            self.logger.error(f"Intraday analysis failed for {symbol}: {e}")
            return None

# =============================================================================
# SECTION 4: OUTPUT GENERATION
# =============================================================================

class OutputGenerator:
    """Generates output files and reports."""
    
    def __init__(self, config: ConfigurationManager):
        self.config = config
        self.logger = logging.getLogger(__name__)
    
    def generate_csv_output(self, results: List[Dict], output_filename: str) -> str:
        """Generate CSV output with analysis results (Master CSV approach)."""
        try:
            # Create output directory
            output_dir = self.config.output_config['output_dir']
            os.makedirs(output_dir, exist_ok=True)
            
            output_path = os.path.join(output_dir, output_filename)
            
            # Prepare new data for CSV
            new_data = []
            for result in results:
                lowest_day = result.get("lowest_bb_day", {})
                new_data.append({
                    "instrument_key": str(result["instrument_key"]),
                    "symbol": str(result["symbol"]),
                    "analysis_date": str(result["analysis_date"]),
                    "lookback_days": str(result["lookback_days"]),
                    "total_days_analyzed": str(result["total_days_analyzed"]),
                    "data_type": str(result.get("data_type", "unknown")),
                    "lowest_bb_date": str(lowest_day.get("date", "")),
                    "lowest_p10_bb_width": f"{lowest_day.get('p10_bb_width', 0):.2f}",
                    "lowest_mean_bb_width": f"{lowest_day.get('mean_bb_width', 0):.2f}",
                    "lowest_min_bb_width": f"{lowest_day.get('min_bb_width', 0):.2f}",
                    "lowest_max_bb_width": f"{lowest_day.get('max_bb_width', 0):.2f}",
                    "lowest_day_data_points": str(lowest_day.get("data_points", 0))
                })
            
            # Create DataFrame for new data
            new_df = pl.DataFrame(new_data)
            
            # Check if existing CSV file exists
            if os.path.exists(output_path):
                try:
                    # Read existing CSV, force all columns to string
                    existing_df = pl.read_csv(output_path, dtypes={
                        "instrument_key": pl.Utf8,
                        "symbol": pl.Utf8,
                        "analysis_date": pl.Utf8,
                        "lookback_days": pl.Utf8,
                        "total_days_analyzed": pl.Utf8,
                        "data_type": pl.Utf8,
                        "lowest_bb_date": pl.Utf8,
                        "lowest_p10_bb_width": pl.Utf8,
                        "lowest_mean_bb_width": pl.Utf8,
                        "lowest_min_bb_width": pl.Utf8,
                        "lowest_max_bb_width": pl.Utf8,
                        "lowest_day_data_points": pl.Utf8
                    })
                    self.logger.info(f"Found existing CSV with {existing_df.height} records")
                    
                    # Create composite keys for matching (symbol + lookback_days)
                    existing_df = existing_df.with_columns(
                        pl.concat_str([
                            pl.col("symbol"), 
                            pl.col("lookback_days")
                        ], separator="|").alias("composite_key")
                    )
                    
                    new_df = new_df.with_columns(
                        pl.concat_str([
                            pl.col("symbol"), 
                            pl.col("lookback_days")
                        ], separator="|").alias("composite_key")
                    )
                    
                    # Get composite keys from new data
                    new_keys = set(new_df["composite_key"].to_list())
                    self.logger.info(f"Current run composite keys: {new_keys}")
                    
                    # Filter existing data to exclude records that will be updated
                    existing_filtered = existing_df.filter(
                        ~pl.col("composite_key").is_in(new_keys)
                    )
                    
                    # Remove composite_key column from both DataFrames
                    existing_filtered = existing_filtered.drop("composite_key")
                    new_df = new_df.drop("composite_key")
                    
                    self.logger.info(f"Preserved {existing_filtered.height} existing records for other symbol/lookback combinations")
                    
                    # Combine existing (filtered) and new data
                    combined_df = pl.concat([existing_filtered, new_df], how="vertical")
                    
                    self.logger.info(f"Updated CSV: {existing_df.height} original records, {existing_filtered.height} preserved, {len(new_data)} new/updated records, {combined_df.height} total records")
                    
                except Exception as e:
                    self.logger.warning(f"Error reading existing CSV, creating new file: {e}")
                    combined_df = new_df
            else:
                # Create new CSV file
                combined_df = new_df
                self.logger.info(f"Creating new CSV file with {len(new_data)} records")
            
            # Save combined DataFrame to CSV
            combined_df.write_csv(output_path)
            
            self.logger.info(f"CSV output saved to: {output_path}")
            return output_path
            
        except Exception as e:
            self.logger.error(f"CSV output generation failed: {e}")
            return ""
    
    def generate_detailed_report(self, results: List[Dict], output_filename: str) -> str:
        """Generate detailed report with all statistics (Master CSV approach)."""
        try:
            # Create output directory
            output_dir = self.config.output_config['output_dir']
            os.makedirs(output_dir, exist_ok=True)
            
            output_path = os.path.join(output_dir, output_filename)
            
            # Prepare detailed data
            new_detailed_data = []
            for result in results:
                for daily_stat in result.get("daily_stats", []):
                    new_detailed_data.append({
                        "instrument_key": str(result["instrument_key"]),
                        "symbol": str(result["symbol"]),
                        "lookback_days": str(result["lookback_days"]),
                        "date": str(daily_stat["date"]),
                        "p10_bb_width": f"{daily_stat['p10_bb_width']:.2f}",
                        "p25_bb_width": f"{daily_stat['p25_bb_width']:.2f}",
                        "p50_bb_width": f"{daily_stat['p50_bb_width']:.2f}",
                        "p75_bb_width": f"{daily_stat['p75_bb_width']:.2f}",
                        "p90_bb_width": f"{daily_stat['p90_bb_width']:.2f}",
                        "p95_bb_width": f"{daily_stat['p95_bb_width']:.2f}",
                        "mean_bb_width": f"{daily_stat['mean_bb_width']:.2f}",
                        "std_bb_width": f"{daily_stat['std_bb_width']:.2f}",
                        "min_bb_width": f"{daily_stat['min_bb_width']:.2f}",
                        "max_bb_width": f"{daily_stat['max_bb_width']:.2f}",
                        "data_points": str(daily_stat["data_points"])
                    })
            
            # Create DataFrame for new detailed data
            new_df = pl.DataFrame(new_detailed_data)
            
            # Check if existing detailed CSV file exists
            if os.path.exists(output_path):
                try:
                    # Read existing CSV, force all columns to string
                    existing_df = pl.read_csv(output_path, dtypes={
                        "instrument_key": pl.Utf8,
                        "symbol": pl.Utf8,
                        "lookback_days": pl.Utf8,
                        "date": pl.Utf8,
                        "p10_bb_width": pl.Utf8,
                        "p25_bb_width": pl.Utf8,
                        "p50_bb_width": pl.Utf8,
                        "p75_bb_width": pl.Utf8,
                        "p90_bb_width": pl.Utf8,
                        "p95_bb_width": pl.Utf8,
                        "mean_bb_width": pl.Utf8,
                        "std_bb_width": pl.Utf8,
                        "min_bb_width": pl.Utf8,
                        "max_bb_width": pl.Utf8,
                        "data_points": pl.Utf8
                    })
                    self.logger.info(f"Found existing detailed CSV with {existing_df.height} records")
                    
                    # Create composite keys for matching (symbol + lookback_days + date)
                    existing_df = existing_df.with_columns(
                        pl.concat_str([
                            pl.col("symbol"), 
                            pl.col("lookback_days"),
                            pl.col("date")
                        ], separator="|").alias("composite_key")
                    )
                    
                    new_df = new_df.with_columns(
                        pl.concat_str([
                            pl.col("symbol"), 
                            pl.col("lookback_days"),
                            pl.col("date")
                        ], separator="|").alias("composite_key")
                    )
                    
                    # Get composite keys from new data
                    new_keys = set(new_df["composite_key"].to_list())
                    self.logger.info(f"Current run composite keys for detailed report: {len(new_keys)} keys")
                    
                    # Filter existing data to exclude records that will be updated
                    existing_filtered = existing_df.filter(
                        ~pl.col("composite_key").is_in(new_keys)
                    )
                    
                    # Remove composite_key column from both DataFrames
                    existing_filtered = existing_filtered.drop("composite_key")
                    new_df = new_df.drop("composite_key")
                    
                    self.logger.info(f"Preserved {existing_filtered.height} existing detailed records for other symbol/lookback/date combinations")
                    
                    # Combine existing (filtered) and new data
                    combined_df = pl.concat([existing_filtered, new_df], how="vertical")
                    
                    self.logger.info(f"Updated detailed CSV: {existing_df.height} original records, {existing_filtered.height} preserved, {len(new_detailed_data)} new/updated records, {combined_df.height} total records")
                    
                except Exception as e:
                    self.logger.warning(f"Error reading existing detailed CSV, creating new file: {e}")
                    combined_df = new_df
            else:
                # Create new detailed CSV file
                combined_df = new_df
                self.logger.info(f"Creating new detailed CSV file with {len(new_detailed_data)} records")
            
            # Save combined DataFrame to CSV
            combined_df.write_csv(output_path)
            
            self.logger.info(f"Detailed report saved to: {output_path}")
            return output_path
            
        except Exception as e:
            self.logger.error(f"Detailed report generation failed: {e}")
            return ""

# =============================================================================
# MAIN EXECUTION FLOW
# =============================================================================

def main():
    """Main execution function."""
    # Parse command line arguments
    parser = argparse.ArgumentParser(
        description="BB Width Intraday Analysis - Database-Driven Analyzer",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter
    )
    
    # Analysis parameters
    parser.add_argument("--symbols", nargs='+', 
                       help="Specific symbols to analyze (e.g., RELIANCE TCS)")
    parser.add_argument("--lookback-days", type=int, 
                       help="Number of days to look back (default: all available data)")
    parser.add_argument("--bb-period", type=int, default=20, 
                       help="Bollinger Bands period")
    parser.add_argument("--bb-std", type=float, default=2.0, 
                       help="Bollinger Bands standard deviations")
    parser.add_argument("--market-start", type=str, default="09:15",
                       help="Market start time (HH:MM)")
    parser.add_argument("--market-end", type=str, default="15:30",
                       help="Market end time (HH:MM)")
    
    # Output parameters
    parser.add_argument("--output-file", type=str, 
                       default="bb_width_analysis.csv",
                       help="Output CSV filename")
    parser.add_argument("--detailed-report", action='store_true',
                       help="Generate detailed report with all daily statistics")
    parser.add_argument("--verbose", action='store_true',
                       help="Enable verbose logging")
    
    args = parser.parse_args()
    
    # Initialize configuration
    config = ConfigurationManager()
    
    # Update config with command line arguments
    config.analysis_params['bb_period'] = args.bb_period
    config.analysis_params['bb_std_dev'] = args.bb_std
    config.analysis_params['market_start'] = args.market_start
    config.analysis_params['market_end'] = args.market_end
    
    # Setup logging
    logging_manager = LoggingManager(config)
    logger = logging_manager.logger
    
    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)
        logger.info("Verbose logging enabled")
    
    # Initialize performance monitor
    monitor = PerformanceMonitor()
    monitor.start_timer("total_analysis")
    
    # Initialize database manager
    db_manager = DatabaseManager(config)
    
    try:
        # Connect to database
        if not db_manager.connect():
            logger.error("Failed to connect to database. Exiting.")
            return
        
        # Initialize analyzers and output generator
        analyzer = IntradayAnalyzer(config, db_manager)
        output_generator = OutputGenerator(config)
        
        # Determine instruments to analyze
        if args.symbols:
            logger.info(f"Analyzing specific symbols: {args.symbols}")
            instruments = analyzer.data_fetcher.get_instruments_by_symbols(
                args.symbols, args.lookback_days
            )
            if not instruments:
                logger.error(f"No instruments found for symbols: {args.symbols}")
                return
        else:
            logger.info("Analyzing all available instruments")
            instruments = analyzer.data_fetcher.get_all_instruments()
            if not instruments:
                logger.error("No instruments found in database")
                return
        
        # Perform analysis
        logger.info(f"Starting analysis of {len(instruments)} instruments")
        monitor.start_timer("analysis")
        results = analyzer.analyze_multiple_instruments(instruments, args.lookback_days)
        monitor.end_timer("analysis")
        
        if not results:
            logger.warning("No analysis results generated")
            return
        
        # Generate outputs
        logger.info("Generating output files")
        monitor.start_timer("output_generation")
        
        # Generate main CSV output
        csv_path = output_generator.generate_csv_output(results, args.output_file)
        
        # Generate detailed report if requested
        if args.detailed_report:
            detailed_filename = f"detailed_{args.output_file}"
            detailed_path = output_generator.generate_detailed_report(results, detailed_filename)
        
        monitor.end_timer("output_generation")
        
        # Display summary
        logger.info(f"\nAnalysis Summary:")
        logger.info(f"  Instruments analyzed: {len(instruments)}")
        logger.info(f"  Successful analyses: {len(results)}")
        logger.info(f"  Output file: {csv_path}")
        if args.detailed_report:
            logger.info(f"  Detailed report: {detailed_path}")
        
        # Display top 5 lowest BB width instruments
        if results:
            logger.info(f"\nTop 5 Instruments with Lowest BB Width:")
            sorted_results = sorted(results, key=lambda x: x.get("lowest_bb_day", {}).get("p10_bb_width", float('inf')))
            for i, result in enumerate(sorted_results[:5], 1):
                lowest_day = result.get("lowest_bb_day", {})
                logger.info(f"  {i}. {result['symbol']} ({result['instrument_key']})")
                logger.info(f"     Lowest BB Width Date: {lowest_day.get('date', 'N/A')}")
                logger.info(f"     P10 BB Width: {lowest_day.get('p10_bb_width', 0):.2f}")
                logger.info(f"     Mean BB Width: {lowest_day.get('mean_bb_width', 0):.2f}")
        
        # Log performance metrics
        monitor.end_timer("total_analysis")
        metrics = monitor.get_metrics()
        logger.info(f"\nPerformance metrics: {metrics}")
        
    except Exception as e:
        logger.error(f"Analysis failed: {e}")
        if args.verbose:
            import traceback
            logger.error(traceback.format_exc())
    finally:
        db_manager.disconnect()

if __name__ == "__main__":
    main()
