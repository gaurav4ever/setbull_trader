#!/usr/bin/env python3
"""
Volatility Squeeze Trading System - Comprehensive Analyzer
==========================================================

This script implements a complete volatility squeeze trading system that identifies
stocks experiencing periods of exceptionally low volatility (consolidation) that
typically precede significant price movements.

Author: Gaurav Sharma - CEO, Setbull Trader
Version: 1.0.0
Date: 2025-06-30
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
import json
warnings.filterwarnings('ignore')

# =============================================================================
# SECTION 1: CONFIGURATION & SETUP
# =============================================================================
# Purpose: Database connections, parameters, logging setup
# Dependencies: None
# Outputs: Database connection, configuration object, logger

class ConfigurationManager:
    """Manages all configuration parameters for the volatility squeeze analyzer."""
    
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
            'pool_name': 'volatility_pool',
            'connection_timeout': 30
        }
        
        # Trading Parameters
        self.trading_params = {
            'bb_period': 20,                    # Bollinger Bands period
            'bb_std_dev': 2.0,                  # Bollinger Bands standard deviations
            'lookback_period': 126,             # Historical lookback period (days)
            'check_period': 5,                  # Recent days to check for squeeze
            'min_data_days': 126,               # Minimum required data days
            'min_price': 10.0,                  # Minimum stock price filter
            'min_avg_volume': 100000,           # Minimum average volume filter
            'exclusions': [
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
                'UTILITIES', 'COMMUNICATION', 'CONSUMER', 'DISCRETIONARY', 'GROWWLIQID', 'LOWVOL1', 'MONQ50', 'MAFANG', 'HDFCPVTBAN', 'HDFCNIFBAN', 'ABSLPSE', 'TOP10ADD', 'ICICIB22', 
            ],
            'blacklist': ['TOP10ADD', 
                          'ICICIB22', 'GROWWLIQID', 'LOWVOL1', 'MONQ50', 'MAFANG', 
                          'HDFCPVTBAN', 'HDFCNIFBAN', 'ABSLPSE', 'GROWWLIQID'],                     # Additional symbols to blacklist
            'proximity_threshold': 10.0  # Added for new command line argument
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
            'candidates_dir': 'candidates',
            'logs_dir': 'logs',
            'reports_dir': 'reports',
            'csv_filename': 'volatility_squeeze_candidates.csv'
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
        log_filename = os.path.join(log_dir, f"volatility_analysis_{timestamp}.log")
        
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
# Purpose: Database operations, data validation, caching
# Dependencies: Section 1 (Configuration)
# Outputs: Clean, validated data for analysis

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
    
    def validate_volume_data(self, df: pl.DataFrame) -> bool:
        """Validate volume data."""
        try:
            # Check for non-negative volume
            if df.filter(pl.col("volume") < 0).height > 0:
                self.logger.warning("Found negative volume values")
                return False
            
            return True
        except Exception as e:
            self.logger.error(f"Volume validation failed: {e}")
            return False
    
    def check_data_completeness(self, df: pl.DataFrame, min_days: int) -> bool:
        """Check if data has minimum required days."""
        try:
            if df.height < min_days:
                self.logger.warning(f"Insufficient data: {df.height} days < {min_days} required")
                return False
            
            # Check for large gaps in data
            if "timestamp" in df.columns:
                df_sorted = df.sort("timestamp")
                try:
                    # Try to use Polars duration in days
                    ts = df_sorted["timestamp"].cast(pl.Datetime)
                    ts_diff = ts.diff()
                    # Convert to days (duration in microseconds)
                    date_diffs_days = ts_diff.cast(pl.Int64) / (1_000_000 * 60 * 60 * 24)
                    if (date_diffs_days[1:] > 5).any():
                        self.logger.warning("Found gaps > 5 days in data")
                        return False
                except Exception as e:
                    self.logger.warning(f"Gap check skipped in test/mock data: {e}")
            
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
        """Fetch all unique instruments with daily data, excluding ETFs and non-stock instruments."""
        try:
            query = """
            SELECT DISTINCT scd.instrument_key, su.symbol, su.name
            FROM stock_candle_data scd
            JOIN stock_universe su ON scd.instrument_key = su.instrument_key
            WHERE scd.time_interval = 'day'
            """
            
            df = self.db_manager.execute_query(query)
            if df is None or df.empty:
                self.logger.warning("No instruments found with daily data")
                return []
            
            # Filter out ETFs and non-stock instruments
            initial_count = len(df)
            filtered_reasons = {}
            
            # Apply standard exclusions
            for exclusion in self.config.trading_params['exclusions']:
                # Check both symbol and name columns
                symbol_matches = df[df['symbol'].str.contains(exclusion, case=False, na=False)]
                name_matches = df[df['name'].str.contains(exclusion, case=False, na=False)]
                
                if len(symbol_matches) > 0 or len(name_matches) > 0:
                    filtered_reasons[exclusion] = len(symbol_matches) + len(name_matches)
                
                # Filter out matches from both symbol and name
                df = df[~df['symbol'].str.contains(exclusion, case=False, na=False)]
                df = df[~df['name'].str.contains(exclusion, case=False, na=False)]
            
            # Apply blacklist (exact symbol matches)
            if self.config.trading_params['blacklist']:
                blacklist_matches = df[df['symbol'].isin(self.config.trading_params['blacklist'])]
                if len(blacklist_matches) > 0:
                    filtered_reasons['BLACKLIST'] = len(blacklist_matches)
                    self.logger.info(f"Blacklisted symbols found: {blacklist_matches['symbol'].tolist()}")
                
                # Filter out blacklisted symbols
                df = df[~df['symbol'].isin(self.config.trading_params['blacklist'])]
            
            # Additional filtering for common patterns
            # Remove instruments with very short symbols (likely indices)
            df = df[df['symbol'].str.len() >= 3]
            
            # Remove instruments with all caps and common index patterns
            df = df[~df['symbol'].str.match(r'^[A-Z]{2,5}$')]  # Remove 2-5 letter all caps
            
            # Remove instruments with numbers only
            df = df[~df['symbol'].str.match(r'^\d+$')]
            
            # Remove instruments with special characters (except dots and dashes)
            df = df[~df['symbol'].str.contains(r'[^A-Za-z0-9.-]')]
            
            filtered_count = len(df)
            total_filtered = initial_count - filtered_count
            
            # Log filtering results
            self.logger.info(f"Data sanitization results:")
            self.logger.info(f"  Initial instruments: {initial_count}")
            self.logger.info(f"  Filtered out: {total_filtered}")
            self.logger.info(f"  Remaining instruments: {filtered_count}")
            
            if filtered_reasons:
                self.logger.info("  Filtering breakdown:")
                for reason, count in filtered_reasons.items():
                    self.logger.info(f"    {reason}: {count} instruments")
            
            return df.to_dict('records')
        except Exception as e:
            self.logger.error(f"Error fetching instruments: {e}")
            return []
    
    def get_instrument_data(self, instrument_key: str) -> Optional[pl.DataFrame]:
        """Fetch daily data for a specific instrument."""
        try:
            query = """
            SELECT timestamp, open, high, low, close, volume
            FROM stock_candle_data
            WHERE instrument_key = %s
              AND time_interval = 'day'
            ORDER BY timestamp ASC
            """
            
            df_pandas = self.db_manager.execute_query(query, (instrument_key,))
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
    
    def _apply_data_filters(self, df: pl.DataFrame) -> bool:
        """Apply data quality filters."""
        try:
            # Check minimum data requirements
            if not self.validator.check_data_completeness(df, self.config.trading_params['min_data_days']):
                return False
            
            # Validate price data
            if not self.validator.validate_price_data(df):
                return False
            
            # Validate volume data
            if not self.validator.validate_volume_data(df):
                return False
            
            # Apply minimum price filter
            latest_close = df.tail(1).select("close").item()
            if latest_close < self.config.trading_params['min_price']:
                return False
            
            # Apply minimum volume filter
            avg_volume = df.select(pl.col("volume").mean()).item()
            if avg_volume < self.config.trading_params['min_avg_volume']:
                return False
            
            # Additional quality checks
            # Check for excessive price volatility (likely data errors)
            price_changes = df.select(
                pl.col("close").pct_change().abs()
            ).filter(pl.col("close") > 0.5)  # More than 50% daily change
            
            if price_changes.height > 0:
                self.logger.warning("Found excessive price volatility, likely data errors")
                return False
            
            # Check for zero or very low volume days (more than 20% of data)
            zero_volume_days = df.filter(pl.col("volume") == 0).height
            if zero_volume_days > (len(df) * 0.2):
                self.logger.warning("Too many zero volume days")
                return False
            
            # Check for stale data (no recent updates)
            latest_date = df.tail(1).select("timestamp").item()
            if isinstance(latest_date, str):
                latest_date = pd.to_datetime(latest_date)
            
            days_since_update = (datetime.now() - latest_date).days
            if days_since_update > 30:  # More than 30 days old
                self.logger.warning(f"Data too old: {days_since_update} days since last update")
                return False
            
            return True
        except Exception as e:
            self.logger.error(f"Data filtering failed: {e}")
            return False

# =============================================================================
# SECTION 3: ANALYSIS ENGINE
# =============================================================================
# Purpose: BBW calculations, squeeze detection, metrics computation
# Dependencies: Section 2 (Data Layer)
# Outputs: Calculated metrics for each instrument

class BollingerBandCalculator:
    """Calculates Bollinger Bands and related metrics."""
    
    def __init__(self, config: ConfigurationManager):
        self.config = config
        self.logger = logging.getLogger(__name__)
    
    def calculate_bollinger_bands(self, df: pl.DataFrame) -> pl.DataFrame:
        """Calculate Bollinger Bands and BBW for the given data."""
        try:
            bb_period = self.config.trading_params['bb_period']
            bb_std_dev = self.config.trading_params['bb_std_dev']
            
            # Calculate Bollinger Bands
            df = df.with_columns([
                pl.col("close").rolling_mean(bb_period).alias("bb_mid"),
                pl.col("close").rolling_std(bb_period).alias("bb_std")
            ]).with_columns([
                (pl.col("bb_mid") + bb_std_dev * pl.col("bb_std")).alias("bb_upper"),
                (pl.col("bb_mid") - bb_std_dev * pl.col("bb_std")).alias("bb_lower")
            ]).with_columns([
                ((pl.col("bb_upper") - pl.col("bb_lower")) / pl.col("bb_mid")).alias("bb_width")
            ])
            
            # Drop null values
            df = df.drop_nulls(["bb_width", "bb_upper", "bb_lower", "volume"])
            
            # Filter out non-positive BBW values
            df = df.filter(pl.col("bb_width") > 0)
            
            return df
        except Exception as e:
            self.logger.error(f"Bollinger Band calculation failed: {e}")
            return df

class SqueezeDetector:
    """Detects squeeze conditions and calculates optimal ranges."""
    
    def __init__(self, config: ConfigurationManager):
        self.config = config
        self.logger = logging.getLogger(__name__)
    
    def calculate_optimal_bb_range(self, df: pl.DataFrame) -> Dict:
        """Calculate the shortest optimal BBW range for a stock."""
        try:
            # Use last 252 days (1 year) for optimal range calculation
            lookback_days = min(252, len(df))
            historical_df = df.tail(lookback_days)
            
            if historical_df.is_empty():
                return None
            
            # Calculate BBW percentiles to find the shortest optimal range
            bbw_series = historical_df.select("bb_width").to_series()
            
            # Find the shortest range with highest win rate
            # We'll use 10th to 25th percentile as the optimal range
            percentile_10 = bbw_series.quantile(0.10)
            percentile_25 = bbw_series.quantile(0.25)
            
            # Calculate average BBW for the optimal range
            optimal_range_avg = (percentile_10 + percentile_25) / 2
            
            return {
                "optimal_range_min": percentile_10,
                "optimal_range_max": percentile_25,
                "optimal_range_avg": optimal_range_avg,
                "bbw_10th": percentile_10,
                "bbw_25th": percentile_25,
                "bbw_avg": bbw_series.mean()
            }
        except Exception as e:
            self.logger.error(f"Optimal range calculation failed: {e}")
            return None
    
    def analyze_bbw_trend(self, df: pl.DataFrame, days: int = 5) -> Dict:
        """Analyze BBW trend over the last N days to determine if contracting or expanding."""
        try:
            if len(df) < days:
                return None
            
            # Get last N days of BBW data
            recent_df = df.tail(days)
            bbw_series = recent_df.select("bb_width").to_series()
            
            if len(bbw_series) < 2:
                return None
            
            # Calculate trend direction using proper Polars syntax
            first_bbw = bbw_series.head(1).item()
            last_bbw = bbw_series.tail(1).item()
            
            # Calculate trend percentage
            trend_percentage = ((last_bbw - first_bbw) / first_bbw) * 100 if first_bbw != 0 else 0
            
            # Determine trend direction
            if trend_percentage < -5.0:  # More than 5% decrease
                trend_direction = "CONTRACTING"
                trend_strength = "STRONG" if trend_percentage < -15.0 else "MODERATE"
            elif trend_percentage > 5.0:  # More than 5% increase
                trend_direction = "EXPANDING"
                trend_strength = "STRONG" if trend_percentage > 15.0 else "MODERATE"
            else:
                trend_direction = "STABLE"
                trend_strength = "WEAK"
            
            # Calculate consecutive days of trend
            consecutive_days = 0
            bbw_values = bbw_series.to_list()
            
            for i in range(1, len(bbw_values)):
                if trend_direction == "CONTRACTING" and bbw_values[i-1] > bbw_values[i]:
                    consecutive_days += 1
                elif trend_direction == "EXPANDING" and bbw_values[i-1] < bbw_values[i]:
                    consecutive_days += 1
                else:
                    break
            
            return {
                "trend_direction": trend_direction,
                "trend_strength": trend_strength,
                "trend_percentage": trend_percentage,
                "consecutive_days": consecutive_days,
                "first_bbw": first_bbw,
                "last_bbw": last_bbw,
                "days_analyzed": days
            }
        except Exception as e:
            self.logger.error(f"BBW trend analysis failed: {e}")
            return None
    
    def calculate_range_proximity(self, current_bbw: float, optimal_range: Dict) -> Dict:
        """Calculate how close current BBW is to the optimal range."""
        try:
            optimal_min = optimal_range["optimal_range_min"]
            optimal_max = optimal_range["optimal_range_max"]
            optimal_avg = optimal_range["optimal_range_avg"]
            
            # Check if currently in optimal range
            in_optimal_range = optimal_min <= current_bbw <= optimal_max
            
            # Calculate distance to optimal range
            if current_bbw < optimal_min:
                distance_to_range = optimal_min - current_bbw
                range_status = "BELOW_OPTIMAL"
            elif current_bbw > optimal_max:
                distance_to_range = current_bbw - optimal_max
                range_status = "ABOVE_OPTIMAL"
            else:
                distance_to_range = 0
                range_status = "IN_OPTIMAL"
            
            # Calculate proximity as percentage of average BBW
            proximity_percentage = (distance_to_range / optimal_avg) * 100 if optimal_avg > 0 else 0
            
            # Check if about to enter optimal range (within threshold)
            threshold = self.config.trading_params.get('proximity_threshold', 10.0)
            about_to_enter = proximity_percentage <= threshold and not in_optimal_range
            
            return {
                "in_optimal_range": in_optimal_range,
                "about_to_enter": about_to_enter,
                "range_status": range_status,
                "proximity_percentage": proximity_percentage,
                "distance_to_range": distance_to_range,
                "optimal_range_min": optimal_min,
                "optimal_range_max": optimal_max,
                "optimal_range_avg": optimal_avg
            }
        except Exception as e:
            self.logger.error(f"Range proximity calculation failed: {e}")
            return None
    
    def categorize_stock(self, current_bbw: float, optimal_range: Dict, bbw_trend: Dict) -> str:
        """Categorize stock based on current BBW, optimal range, and trend."""
        try:
            if not optimal_range or not bbw_trend:
                return "C"
            
            optimal_min = optimal_range["optimal_range_min"]
            optimal_max = optimal_range["optimal_range_max"]
            trend_direction = bbw_trend["trend_direction"]
            trend_strength = bbw_trend["trend_strength"]
            
            # Category A: Currently in optimal BBW range
            if optimal_min <= current_bbw <= optimal_max:
                return "A"
            
            # Category B: About to enter optimal range (contracting and approaching)
            if trend_direction == "CONTRACTING" and trend_strength in ["STRONG", "MODERATE"]:
                if current_bbw > optimal_max:  # Above optimal range and contracting
                    return "B"
            
            # Category C: About to exit optimal range (expanding and moving away)
            if trend_direction == "EXPANDING" and trend_strength in ["STRONG", "MODERATE"]:
                if optimal_min <= current_bbw <= optimal_max:  # In optimal range but expanding
                    return "C"
                elif current_bbw < optimal_min:  # Below optimal range and expanding further
                    return "C"
            
            # Default to Category C for other cases
            return "C"
            
        except Exception as e:
            self.logger.error(f"Stock categorization failed: {e}")
            return "C"
    
    def detect_squeeze(self, df: pl.DataFrame) -> Optional[Dict]:
        """Detect if the instrument is currently in a squeeze condition."""
        try:
            lookback_period = self.config.trading_params['lookback_period']
            check_period = self.config.trading_params['check_period']
            
            # Need enough data for analysis
            if len(df) < lookback_period:
                return None
            
            # Use the last lookback_period of data to establish baseline
            lookback_df = df.tail(lookback_period)
            
            # Calculate 10th percentile threshold
            percentile_10_threshold = lookback_df.select(
                pl.col("bb_width").quantile(0.10)
            ).item()
            
            # Calculate average BBW over lookback period
            avg_bb_width_lookback = lookback_df.select(
                pl.col("bb_width").mean()
            ).item()
            
            if percentile_10_threshold is None:
                return None
            
            # Check recent days for squeeze signal
            recent_days_df = df.tail(check_period)
            low_vol_days = recent_days_df.filter(
                pl.col("bb_width") <= percentile_10_threshold
            )
            
            if low_vol_days.is_empty():
                return None
            
            # Get latest data
            latest_day = df.tail(1)
            latest_bb_width = latest_day.select("bb_width").item()
            latest_close = latest_day.select("close").item()
            latest_date = latest_day.select("timestamp").item()
            
            # Calculate squeeze ratio
            squeeze_ratio = latest_bb_width / avg_bb_width_lookback if avg_bb_width_lookback > 0 else 1.0
            
            # Calculate volume ratio (5-day avg / 50-day avg)
            if len(df) >= 50:
                recent_volume = df.tail(5).select("volume").mean().item()
                historical_volume = df.tail(50).select("volume").mean().item()
                volume_ratio = recent_volume / historical_volume if historical_volume > 0 else 1.0
            else:
                volume_ratio = 1.0
            
            # Calculate breakout readiness
            bb_upper = latest_day.select("bb_upper").item()
            bb_lower = latest_day.select("bb_lower").item()
            bb_range = bb_upper - bb_lower
            
            if bb_range > 0:
                breakout_readiness = (latest_close - bb_lower) / bb_range
            else:
                breakout_readiness = 0.5
            
            # Calculate optimal range and trend analysis
            optimal_range = self.calculate_optimal_bb_range(df)
            bbw_trend = self.analyze_bbw_trend(df, days=5)
            
            # Calculate range proximity
            range_proximity = None
            if optimal_range:
                range_proximity = self.calculate_range_proximity(latest_bb_width, optimal_range)
            
            # Categorize the stock
            category = self.categorize_stock(latest_bb_width, optimal_range, bbw_trend)
            
            # Compile results
            result = {
                "latest_date": latest_date,
                "latest_close": latest_close,
                "latest_bb_width": latest_bb_width,
                "10_percentile_threshold": percentile_10_threshold,
                "avg_bb_width_lookback": avg_bb_width_lookback,
                "squeeze_ratio": squeeze_ratio,
                "volume_ratio": volume_ratio,
                "breakout_readiness": breakout_readiness,
                "category": category
            }
            
            # Add optimal range data
            if optimal_range:
                result.update({
                    "optimal_range_min": optimal_range["optimal_range_min"],
                    "optimal_range_max": optimal_range["optimal_range_max"],
                    "optimal_range_avg": optimal_range["optimal_range_avg"],
                    "bbw_10th": optimal_range["bbw_10th"],
                    "bbw_25th": optimal_range["bbw_25th"],
                    "bbw_avg": optimal_range["bbw_avg"]
                })
            
            # Add range proximity data
            if range_proximity:
                result.update({
                    "in_optimal_range": range_proximity["in_optimal_range"],
                    "about_to_enter": range_proximity["about_to_enter"],
                    "range_status": range_proximity["range_status"],
                    "proximity_percentage": range_proximity["proximity_percentage"],
                    "distance_to_range": range_proximity["distance_to_range"]
                })
            
            # Add BBW trend data
            if bbw_trend:
                result.update({
                    "trend_direction": bbw_trend["trend_direction"],
                    "trend_strength": bbw_trend["trend_strength"],
                    "trend_percentage": bbw_trend["trend_percentage"],
                    "consecutive_days": bbw_trend["consecutive_days"],
                    "first_bbw": bbw_trend["first_bbw"],
                    "last_bbw": bbw_trend["last_bbw"],
                    "days_analyzed": bbw_trend["days_analyzed"]
                })
            
            return result
            
        except Exception as e:
            self.logger.error(f"Squeeze detection failed: {e}")
            return None

class MetricsCalculator:
    """Calculates additional trading metrics."""
    
    def __init__(self, config: ConfigurationManager):
        self.config = config
        self.logger = logging.getLogger(__name__)
    
    def calculate_breakdown_readiness(self, breakout_readiness: float) -> float:
        """Calculate breakdown readiness score."""
        if breakout_readiness is not None:
            return 1 - breakout_readiness
        return None
    
    def calculate_percentile_rank(self, current_bbw: float, historical_bbw: List[float]) -> float:
        """Calculate percentile rank of current BBW in historical context."""
        try:
            if not historical_bbw:
                return None
            
            # Count how many historical values are less than current
            count_less = sum(1 for x in historical_bbw if x < current_bbw)
            percentile_rank = (count_less / len(historical_bbw)) * 100
            
            return percentile_rank
        except Exception as e:
            self.logger.error(f"Percentile rank calculation failed: {e}")
            return None

class VolatilityAnalyzer:
    """Main analyzer that orchestrates the analysis process."""
    
    def __init__(self, config: ConfigurationManager, db_manager: DatabaseManager):
        self.config = config
        self.db_manager = db_manager
        self.data_fetcher = DataFetcher(config, db_manager)
        self.bb_calculator = BollingerBandCalculator(config)
        self.squeeze_detector = SqueezeDetector(config)
        self.metrics_calculator = MetricsCalculator(config)
        self.logger = logging.getLogger(__name__)
    
    def analyze_instrument(self, instrument_key: str, symbol: str) -> Optional[Dict]:
        """Analyze a single instrument for volatility squeeze conditions."""
        try:
            # Fetch instrument data
            df = self.data_fetcher.get_instrument_data(instrument_key)
            if df is None:
                return None
            
            # Calculate Bollinger Bands and BBW
            df = self.bb_calculator.calculate_bollinger_bands(df)
            if df.is_empty():
                return None
            
            # Detect squeeze conditions
            squeeze_data = self.squeeze_detector.detect_squeeze(df)
            if squeeze_data is None:
                return None
            
            # Calculate additional metrics
            breakdown_readiness = self.metrics_calculator.calculate_breakdown_readiness(
                squeeze_data["breakout_readiness"]
            )
            
            # Calculate percentile rank
            historical_bbw = df.select("bb_width").to_series().to_list()
            percentile_rank = self.metrics_calculator.calculate_percentile_rank(
                squeeze_data["latest_bb_width"], historical_bbw
            )
            
            # Compile results with all new columns
            result = {
                "instrument_key": instrument_key,
                "symbol": symbol,
                "latest_date": squeeze_data["latest_date"],
                "latest_close": squeeze_data["latest_close"],
                "latest_bb_width": squeeze_data["latest_bb_width"],
                "10_percentile_threshold": squeeze_data["10_percentile_threshold"],
                "avg_bb_width_lookback": squeeze_data["avg_bb_width_lookback"],
                "squeeze_ratio": squeeze_data["squeeze_ratio"],
                "volume_ratio": squeeze_data["volume_ratio"],
                "breakout_readiness": squeeze_data["breakout_readiness"],
                "breakdown_readiness": breakdown_readiness,
                "percentile_rank": percentile_rank,
                "category": squeeze_data["category"]
            }
            
            # Add optimal range analysis if available
            if "optimal_range_min" in squeeze_data:
                result.update({
                    "optimal_range_min": squeeze_data["optimal_range_min"],
                    "optimal_range_max": squeeze_data["optimal_range_max"],
                    "optimal_range_avg": squeeze_data["optimal_range_avg"],
                    "bbw_10th": squeeze_data["bbw_10th"],
                    "bbw_25th": squeeze_data["bbw_25th"],
                    "bbw_avg": squeeze_data["bbw_avg"]
                })
            
            # Add range proximity analysis if available
            if "in_optimal_range" in squeeze_data:
                result.update({
                    "in_optimal_range": squeeze_data["in_optimal_range"],
                    "about_to_enter": squeeze_data["about_to_enter"],
                    "range_status": squeeze_data["range_status"],
                    "proximity_percentage": squeeze_data["proximity_percentage"],
                    "distance_to_range": squeeze_data["distance_to_range"]
                })
            
            # Add BBW trend analysis if available
            if "trend_direction" in squeeze_data:
                result.update({
                    "trend_direction": squeeze_data["trend_direction"],
                    "trend_strength": squeeze_data["trend_strength"],
                    "trend_percentage": squeeze_data["trend_percentage"],
                    "consecutive_days": squeeze_data["consecutive_days"],
                    "first_bbw": squeeze_data["first_bbw"],
                    "last_bbw": squeeze_data["last_bbw"],
                    "days_analyzed": squeeze_data["days_analyzed"]
                })
            
            return result
            
        except Exception as e:
            self.logger.error(f"Analysis failed for {symbol}: {e}")
            return None
    
    def analyze_universe(self) -> List[Dict]:
        """Analyze the entire universe of instruments."""
        try:
            # Get all instruments
            instruments = self.data_fetcher.get_all_instruments()
            if not instruments:
                self.logger.error("No instruments found for analysis")
                return []
            
            self.logger.info(f"Starting analysis of {len(instruments)} instruments")
            
            # Analyze each instrument
            results = []
            for instrument in tqdm(instruments, desc="Analyzing instruments"):
                result = self.analyze_instrument(
                    instrument['instrument_key'], 
                    instrument['symbol']
                )
                if result:
                    results.append(result)
            
            self.logger.info(f"Analysis complete. Found {len(results)} squeeze candidates")
            return results
            
        except Exception as e:
            self.logger.error(f"Universe analysis failed: {e}")
            return []

# =============================================================================
# SECTION 4: INDIVIDUAL ANALYSIS (Phase 3)
# =============================================================================
# Purpose: Detailed individual stock analysis with historical context
# Dependencies: Section 3 (Analysis Engine)
# Outputs: Comprehensive individual stock reports

class IndividualAnalyzer:
    """Performs detailed individual stock analysis with historical context."""
    
    def __init__(self, config: ConfigurationManager):
        self.config = config
        self.logger = logging.getLogger(__name__)
    
    def analyze_individual_stock(self, instrument_key: str, symbol: str, df: pl.DataFrame) -> Dict:
        """Perform comprehensive individual stock analysis."""
        try:
            # Calculate Bollinger Bands and BBW
            bb_calculator = BollingerBandCalculator(self.config)
            df_with_bb = bb_calculator.calculate_bollinger_bands(df)
            
            if df_with_bb.is_empty():
                return None
            
            # Get latest data
            latest_data = df_with_bb.tail(1)
            latest_close = latest_data.select("close").item()
            latest_bb_width = latest_data.select("bb_width").item()
            latest_date = latest_data.select("timestamp").item()
            
            # Historical context analysis (126-day lookback)
            lookback_period = self.config.trading_params['lookback_period']
            historical_df = df_with_bb.tail(lookback_period)
            
            # Calculate historical percentiles
            bbw_percentiles = self._calculate_historical_percentiles(historical_df)
            
            # Contraction confirmation (3-5 day analysis)
            contraction_analysis = self._analyze_contraction_confirmation(df_with_bb)
            
            # Tradable range analysis
            tradable_range_analysis = self._analyze_tradable_range(df_with_bb)
            
            # Performance profile analysis
            performance_profile = self._generate_performance_profile(df_with_bb)
            
            # Compile comprehensive analysis
            analysis_result = {
                "instrument_key": instrument_key,
                "symbol": symbol,
                "analysis_date": latest_date,
                "latest_close": latest_close,
                "latest_bb_width": latest_bb_width,
                "historical_percentiles": bbw_percentiles,
                "contraction_analysis": contraction_analysis,
                "tradable_range_analysis": tradable_range_analysis,
                "performance_profile": performance_profile,
                "analysis_summary": self._generate_analysis_summary(
                    latest_bb_width, bbw_percentiles, contraction_analysis, 
                    tradable_range_analysis, performance_profile
                )
            }
            
            return analysis_result
            
        except Exception as e:
            self.logger.error(f"Individual analysis failed for {symbol}: {e}")
            return None
    
    def _calculate_historical_percentiles(self, historical_df: pl.DataFrame) -> Dict:
        """Calculate historical BBW percentiles for context."""
        try:
            bbw_series = historical_df.select("bb_width").to_series()
            
            percentiles = {
                "current_bbw": bbw_series.tail(1).item(),
                "percentile_5": bbw_series.quantile(0.05),
                "percentile_10": bbw_series.quantile(0.10),
                "percentile_25": bbw_series.quantile(0.25),
                "percentile_50": bbw_series.quantile(0.50),
                "percentile_75": bbw_series.quantile(0.75),
                "percentile_90": bbw_series.quantile(0.90),
                "percentile_95": bbw_series.quantile(0.95),
                "mean": bbw_series.mean(),
                "std": bbw_series.std(),
                "min": bbw_series.min(),
                "max": bbw_series.max()
            }
            
            # Calculate current percentile rank
            current_bbw = percentiles["current_bbw"]
            percentile_rank = (bbw_series < current_bbw).sum() / len(bbw_series) * 100
            percentiles["current_percentile_rank"] = percentile_rank
            
            return percentiles
            
        except Exception as e:
            self.logger.error(f"Historical percentiles calculation failed: {e}")
            return {}
    
    def _analyze_contraction_confirmation(self, df: pl.DataFrame) -> Dict:
        """Analyze recent contraction confirmation (3-5 days)."""
        try:
            # Analyze last 5 days for contraction confirmation
            recent_df = df.tail(5)
            bbw_trend = recent_df.select("bb_width").to_series()
            
            # Calculate contraction metrics using Polars operations
            if len(bbw_trend) >= 2:
                bbw_decline = (bbw_trend.head(1).item() - bbw_trend.tail(1).item()) / bbw_trend.head(1).item() * 100
                
                # Count consecutive declines
                consecutive_declines = 0
                for i in range(1, len(bbw_trend)):
                    if bbw_trend.head(i).item() > bbw_trend.head(i+1).item():
                        consecutive_declines += 1
                    else:
                        break
                
                # Volume analysis for confirmation
                volume_trend = recent_df.select("volume").to_series()
                if len(volume_trend) >= 2:
                    volume_decline = (volume_trend.head(1).item() - volume_trend.tail(1).item()) / volume_trend.head(1).item() * 100
                else:
                    volume_decline = 0
                
                contraction_analysis = {
                    "bbw_decline_percent": bbw_decline,
                    "consecutive_declines": consecutive_declines,
                    "volume_decline_percent": volume_decline,
                    "is_contracting": bbw_decline > 5.0,  # 5% decline threshold
                    "volume_confirms": volume_decline > 10.0,  # 10% volume decline
                    "contraction_strength": "STRONG" if bbw_decline > 15.0 else "MODERATE" if bbw_decline > 5.0 else "WEAK"
                }
            else:
                contraction_analysis = {
                    "bbw_decline_percent": 0,
                    "consecutive_declines": 0,
                    "volume_decline_percent": 0,
                    "is_contracting": False,
                    "volume_confirms": False,
                    "contraction_strength": "WEAK"
                }
            
            return contraction_analysis
            
        except Exception as e:
            self.logger.error(f"Contraction analysis failed: {e}")
            return {}
    
    def _analyze_tradable_range(self, df: pl.DataFrame) -> Dict:
        """Analyze tradable range characteristics."""
        try:
            # Use last 252 days for tradable range analysis
            lookback_days = min(252, len(df))
            historical_df = df.tail(lookback_days)
            
            # Calculate optimal ranges
            bbw_series = historical_df.select("bb_width").to_series()
            
            # Define multiple range categories
            ranges = {
                "ultra_tight": (bbw_series.quantile(0.05), bbw_series.quantile(0.10)),
                "tight": (bbw_series.quantile(0.10), bbw_series.quantile(0.25)),
                "normal": (bbw_series.quantile(0.25), bbw_series.quantile(0.75)),
                "wide": (bbw_series.quantile(0.75), bbw_series.quantile(0.90)),
                "ultra_wide": (bbw_series.quantile(0.90), bbw_series.quantile(0.95))
            }
            
            current_bbw = bbw_series.tail(1).item()
            
            # Determine current range category
            current_range = "UNKNOWN"
            for range_name, (min_val, max_val) in ranges.items():
                if min_val <= current_bbw <= max_val:
                    current_range = range_name.upper()
                    break
            
            # Calculate range statistics
            range_analysis = {
                "current_range": current_range,
                "ranges": ranges,
                "current_bbw": current_bbw,
                "optimal_range_min": ranges["tight"][0],
                "optimal_range_max": ranges["tight"][1],
                "optimal_range_avg": (ranges["tight"][0] + ranges["tight"][1]) / 2,
                "distance_to_optimal": min(abs(current_bbw - ranges["tight"][0]), 
                                         abs(current_bbw - ranges["tight"][1]))
            }
            
            return range_analysis
            
        except Exception as e:
            self.logger.error(f"Tradable range analysis failed: {e}")
            return {}
    
    def _generate_performance_profile(self, df: pl.DataFrame) -> Dict:
        """Generate historical performance profile."""
        try:
            # Use last 252 days for performance analysis
            lookback_days = min(252, len(df))
            historical_df = df.tail(lookback_days)
            
            # Calculate BBW-based performance metrics
            bbw_series = historical_df.select("bb_width").to_series()
            close_series = historical_df.select("close").to_series()
            
            # Calculate returns for different BBW ranges
            performance_by_range = {}
            
            # Define BBW ranges
            bbw_ranges = [
                ("ultra_tight", bbw_series.quantile(0.05), bbw_series.quantile(0.10)),
                ("tight", bbw_series.quantile(0.10), bbw_series.quantile(0.25)),
                ("normal", bbw_series.quantile(0.25), bbw_series.quantile(0.75)),
                ("wide", bbw_series.quantile(0.75), bbw_series.quantile(0.90))
            ]
            
            for range_name, min_bbw, max_bbw in bbw_ranges:
                # Find periods in this BBW range using Polars filter
                in_range_mask = (bbw_series >= min_bbw) & (bbw_series <= max_bbw)
                in_range_indices = []
                
                # Convert mask to indices
                for i, in_range in enumerate(in_range_mask):
                    if in_range:
                        in_range_indices.append(i)
                
                if len(in_range_indices) > 0:
                    # Calculate returns for periods in this range
                    range_returns = []
                    for i in range(1, len(in_range_indices)):
                        if in_range_indices[i] - in_range_indices[i-1] == 1:  # Consecutive days
                            prev_idx = in_range_indices[i-1]
                            curr_idx = in_range_indices[i]
                            if prev_idx < len(close_series) and curr_idx < len(close_series):
                                prev_price = close_series.head(prev_idx + 1).tail(1).item()
                                curr_price = close_series.head(curr_idx + 1).tail(1).item()
                                ret = (curr_price - prev_price) / prev_price * 100
                                range_returns.append(ret)
                    
                    if range_returns:
                        performance_by_range[range_name] = {
                            "avg_return": sum(range_returns) / len(range_returns),
                            "win_rate": sum(1 for r in range_returns if r > 0) / len(range_returns) * 100,
                            "max_return": max(range_returns),
                            "min_return": min(range_returns),
                            "volatility": (sum((r - sum(range_returns)/len(range_returns))**2 for r in range_returns) / len(range_returns))**0.5,
                            "periods_count": len(range_returns)
                        }
            
            # Find best performing range
            best_range = None
            best_win_rate = 0
            for range_name, metrics in performance_by_range.items():
                if metrics["win_rate"] > best_win_rate:
                    best_win_rate = metrics["win_rate"]
                    best_range = range_name
            
            performance_profile = {
                "performance_by_range": performance_by_range,
                "best_performing_range": best_range,
                "best_win_rate": best_win_rate,
                "total_analysis_periods": len(bbw_series),
                "current_bbw_percentile": (bbw_series < bbw_series.tail(1).item()).sum() / len(bbw_series) * 100
            }
            
            return performance_profile
            
        except Exception as e:
            self.logger.error(f"Performance profile generation failed: {e}")
            return {}
    
    def _generate_analysis_summary(self, current_bbw: float, percentiles: Dict, 
                                 contraction: Dict, tradable_range: Dict, 
                                 performance: Dict) -> Dict:
        """Generate comprehensive analysis summary."""
        try:
            # Determine squeeze status
            current_percentile = percentiles.get("current_percentile_rank", 50)
            is_in_squeeze = current_percentile <= 25  # Bottom 25%
            
            # Determine trading recommendation
            recommendation = "HOLD"
            confidence = "LOW"
            
            if is_in_squeeze:
                if contraction.get("is_contracting", False) and contraction.get("volume_confirms", False):
                    recommendation = "STRONG_BUY"
                    confidence = "HIGH"
                elif contraction.get("is_contracting", False):
                    recommendation = "BUY"
                    confidence = "MEDIUM"
                else:
                    recommendation = "WATCH"
                    confidence = "LOW"
            
            # Risk assessment
            risk_level = "LOW"
            if current_percentile <= 10:
                risk_level = "VERY_LOW"
            elif current_percentile <= 25:
                risk_level = "LOW"
            elif current_percentile <= 50:
                risk_level = "MEDIUM"
            else:
                risk_level = "HIGH"
            
            summary = {
                "squeeze_status": "IN_SQUEEZE" if is_in_squeeze else "NOT_IN_SQUEEZE",
                "current_percentile": current_percentile,
                "recommendation": recommendation,
                "confidence": confidence,
                "risk_level": risk_level,
                "contraction_strength": contraction.get("contraction_strength", "UNKNOWN"),
                "optimal_range_status": tradable_range.get("current_range", "UNKNOWN"),
                "best_performing_range": performance.get("best_performing_range", "UNKNOWN")
            }
            
            return summary
            
        except Exception as e:
            self.logger.error(f"Analysis summary generation failed: {e}")
            return {}

# =============================================================================
# SECTION 5: HISTORICAL PERFORMANCE ANALYSIS (Phase 4)
# =============================================================================
# Purpose: Historical backtesting, optimal range calculation, performance profiling
# Dependencies: Section 4 (Individual Analysis)
# Outputs: Performance profiles, backtesting results, optimal ranges

class BacktestEngine:
    """Historical backtesting engine for squeeze strategies."""
    
    def __init__(self, config: ConfigurationManager):
        self.config = config
        self.logger = logging.getLogger(__name__)
    
    def backtest_squeeze_strategy(self, df: pl.DataFrame, symbol: str) -> Dict:
        """Backtest squeeze strategy on historical data."""
        try:
            # Calculate Bollinger Bands and BBW
            bb_calculator = BollingerBandCalculator(self.config)
            df_with_bb = bb_calculator.calculate_bollinger_bands(df)
            
            if df_with_bb.is_empty():
                return None
            
            # Define squeeze entry conditions
            squeeze_entries = self._identify_squeeze_entries(df_with_bb)
            
            # Calculate returns for each squeeze period
            trade_results = self._calculate_trade_returns(df_with_bb, squeeze_entries)
            
            # Generate performance metrics
            performance_metrics = self._calculate_performance_metrics(trade_results)
            
            # Risk analysis
            risk_metrics = self._calculate_risk_metrics(trade_results)
            
            backtest_result = {
                "symbol": symbol,
                "total_trades": len(trade_results),
                "squeeze_entries": squeeze_entries,
                "trade_results": trade_results,
                "performance_metrics": performance_metrics,
                "risk_metrics": risk_metrics,
                "backtest_summary": self._generate_backtest_summary(performance_metrics, risk_metrics)
            }
            
            return backtest_result
            
        except Exception as e:
            self.logger.error(f"Backtest failed for {symbol}: {e}")
            return None
    
    def _identify_squeeze_entries(self, df: pl.DataFrame) -> List[Dict]:
        """Identify squeeze entry points in historical data."""
        try:
            entries = []
            lookback_period = self.config.trading_params['lookback_period']
            
            # Need enough data for analysis
            if len(df) < lookback_period + 20:
                return entries
            
            # Analyze each potential entry point
            for i in range(lookback_period, len(df) - 5):  # Leave 5 days for exit
                # Get historical context
                historical_df = df.slice(i - lookback_period, lookback_period)
                current_bbw = df.slice(i, 1).select("bb_width").item()
                
                # Calculate 10th percentile threshold
                threshold = historical_df.select(pl.col("bb_width").quantile(0.10)).item()
                
                # Check if current BBW is in squeeze territory
                if current_bbw <= threshold:
                    # Additional confirmation: check if BBW is declining
                    recent_bbw = df.slice(i-5, 5).select("bb_width").to_series()
                    if len(recent_bbw) >= 2:
                        bbw_decline = (recent_bbw.head(1).item() - recent_bbw.tail(1).item()) / recent_bbw.head(1).item() * 100
                        
                        if bbw_decline > 5.0:  # 5% decline confirmation
                            entry = {
                                "entry_date": df.slice(i, 1).select("timestamp").item(),
                                "entry_price": df.slice(i, 1).select("close").item(),
                                "entry_bbw": current_bbw,
                                "threshold": threshold,
                                "bbw_decline": bbw_decline,
                                "entry_index": i
                            }
                            entries.append(entry)
            
            return entries
            
        except Exception as e:
            self.logger.error(f"Squeeze entry identification failed: {e}")
            return []
    
    def _calculate_trade_returns(self, df: pl.DataFrame, entries: List[Dict]) -> List[Dict]:
        """Calculate returns for each squeeze trade."""
        try:
            trade_results = []
            
            for entry in entries:
                entry_index = entry["entry_index"]
                entry_price = entry["entry_price"]
                
                # Define exit conditions (5 days max hold)
                max_hold_days = 5
                exit_index = min(entry_index + max_hold_days, len(df) - 1)
                
                # Get exit data
                exit_data = df.slice(exit_index, 1)
                exit_price = exit_data.select("close").item()
                exit_date = exit_data.select("timestamp").item()
                
                # Calculate return
                return_pct = (exit_price - entry_price) / entry_price * 100
                
                # Determine exit reason
                exit_reason = "MAX_HOLD"
                if exit_index == entry_index + max_hold_days:
                    exit_reason = "MAX_HOLD"
                else:
                    exit_reason = "END_OF_DATA"
                
                trade_result = {
                    "entry_date": entry["entry_date"],
                    "exit_date": exit_date,
                    "entry_price": entry_price,
                    "exit_price": exit_price,
                    "return_pct": return_pct,
                    "hold_days": exit_index - entry_index,
                    "exit_reason": exit_reason,
                    "entry_bbw": entry["entry_bbw"],
                    "bbw_decline": entry["bbw_decline"]
                }
                
                trade_results.append(trade_result)
            
            return trade_results
            
        except Exception as e:
            self.logger.error(f"Trade return calculation failed: {e}")
            return []
    
    def _calculate_performance_metrics(self, trade_results: List[Dict]) -> Dict:
        """Calculate comprehensive performance metrics."""
        try:
            if not trade_results:
                return {}
            
            returns = [trade["return_pct"] for trade in trade_results]
            positive_returns = [r for r in returns if r > 0]
            negative_returns = [r for r in returns if r < 0]
            
            metrics = {
                "total_trades": len(trade_results),
                "winning_trades": len(positive_returns),
                "losing_trades": len(negative_returns),
                "win_rate": len(positive_returns) / len(returns) * 100 if returns else 0,
                "avg_return": sum(returns) / len(returns) if returns else 0,
                "avg_win": sum(positive_returns) / len(positive_returns) if positive_returns else 0,
                "avg_loss": sum(negative_returns) / len(negative_returns) if negative_returns else 0,
                "max_win": max(returns) if returns else 0,
                "max_loss": min(returns) if returns else 0,
                "total_return": sum(returns),
                "profit_factor": abs(sum(positive_returns) / sum(negative_returns)) if negative_returns and sum(negative_returns) != 0 else float('inf'),
                "sharpe_ratio": self._calculate_sharpe_ratio(returns),
                "max_drawdown": self._calculate_max_drawdown(returns)
            }
            
            return metrics
            
        except Exception as e:
            self.logger.error(f"Performance metrics calculation failed: {e}")
            return {}
    
    def _calculate_risk_metrics(self, trade_results: List[Dict]) -> Dict:
        """Calculate risk metrics for the strategy."""
        try:
            if not trade_results:
                return {}
            
            returns = [trade["return_pct"] for trade in trade_results]
            
            # Calculate volatility
            mean_return = sum(returns) / len(returns) if returns else 0
            variance = sum((r - mean_return) ** 2 for r in returns) / len(returns) if returns else 0
            volatility = variance ** 0.5
            
            # Calculate Value at Risk (VaR)
            sorted_returns = sorted(returns)
            var_95 = sorted_returns[int(len(sorted_returns) * 0.05)] if len(sorted_returns) > 0 else 0
            
            # Calculate maximum consecutive losses
            max_consecutive_losses = 0
            current_consecutive_losses = 0
            for r in returns:
                if r < 0:
                    current_consecutive_losses += 1
                    max_consecutive_losses = max(max_consecutive_losses, current_consecutive_losses)
                else:
                    current_consecutive_losses = 0
            
            risk_metrics = {
                "volatility": volatility,
                "var_95": var_95,
                "max_consecutive_losses": max_consecutive_losses,
                "avg_hold_days": sum(trade["hold_days"] for trade in trade_results) / len(trade_results) if trade_results else 0,
                "risk_reward_ratio": abs(sum(r for r in returns if r > 0) / sum(r for r in returns if r < 0)) if any(r < 0 for r in returns) and sum(r for r in returns if r < 0) != 0 else float('inf')
            }
            
            return risk_metrics
            
        except Exception as e:
            self.logger.error(f"Risk metrics calculation failed: {e}")
            return {}
    
    def _calculate_sharpe_ratio(self, returns: List[float]) -> float:
        """Calculate Sharpe ratio (assuming 0% risk-free rate)."""
        try:
            if not returns:
                return 0.0
            
            mean_return = sum(returns) / len(returns)
            variance = sum((r - mean_return) ** 2 for r in returns) / len(returns)
            std_dev = variance ** 0.5
            
            return mean_return / std_dev if std_dev != 0 else 0.0
            
        except Exception as e:
            self.logger.error(f"Sharpe ratio calculation failed: {e}")
            return 0.0
    
    def _calculate_max_drawdown(self, returns: List[float]) -> float:
        """Calculate maximum drawdown."""
        try:
            if not returns:
                return 0.0
            
            cumulative_returns = [1.0]
            for r in returns:
                cumulative_returns.append(cumulative_returns[-1] * (1 + r / 100))
            
            max_drawdown = 0.0
            peak = cumulative_returns[0]
            
            for value in cumulative_returns:
                if value > peak:
                    peak = value
                drawdown = (peak - value) / peak * 100
                max_drawdown = max(max_drawdown, drawdown)
            
            return max_drawdown
            
        except Exception as e:
            self.logger.error(f"Max drawdown calculation failed: {e}")
            return 0.0
    
    def _generate_backtest_summary(self, performance: Dict, risk: Dict) -> Dict:
        """Generate backtest summary with recommendations."""
        try:
            # Determine strategy viability
            win_rate = performance.get("win_rate", 0)
            profit_factor = performance.get("profit_factor", 0)
            sharpe_ratio = performance.get("sharpe_ratio", 0)
            
            strategy_viability = "POOR"
            if win_rate >= 60 and profit_factor >= 1.5 and sharpe_ratio >= 0.5:
                strategy_viability = "EXCELLENT"
            elif win_rate >= 50 and profit_factor >= 1.2 and sharpe_ratio >= 0.3:
                strategy_viability = "GOOD"
            elif win_rate >= 40 and profit_factor >= 1.0:
                strategy_viability = "FAIR"
            
            # Risk assessment
            volatility = risk.get("volatility", 0)
            max_drawdown = risk.get("max_drawdown", 0)
            
            risk_level = "LOW"
            if volatility > 5.0 or max_drawdown > 20.0:
                risk_level = "HIGH"
            elif volatility > 3.0 or max_drawdown > 10.0:
                risk_level = "MEDIUM"
            
            summary = {
                "strategy_viability": strategy_viability,
                "risk_level": risk_level,
                "recommendation": "USE" if strategy_viability in ["EXCELLENT", "GOOD"] else "AVOID",
                "confidence": "HIGH" if strategy_viability == "EXCELLENT" else "MEDIUM" if strategy_viability == "GOOD" else "LOW",
                "key_strengths": self._identify_strengths(performance, risk),
                "key_weaknesses": self._identify_weaknesses(performance, risk)
            }
            
            return summary
            
        except Exception as e:
            self.logger.error(f"Backtest summary generation failed: {e}")
            return {}
    
    def _identify_strengths(self, performance: Dict, risk: Dict) -> List[str]:
        """Identify strategy strengths."""
        strengths = []
        
        if performance.get("win_rate", 0) >= 60:
            strengths.append("High win rate")
        if performance.get("profit_factor", 0) >= 1.5:
            strengths.append("Strong profit factor")
        if performance.get("sharpe_ratio", 0) >= 0.5:
            strengths.append("Good risk-adjusted returns")
        if risk.get("volatility", 0) < 3.0:
            strengths.append("Low volatility")
        if risk.get("max_drawdown", 0) < 10.0:
            strengths.append("Low maximum drawdown")
        
        return strengths
    
    def _identify_weaknesses(self, performance: Dict, risk: Dict) -> List[str]:
        """Identify strategy weaknesses."""
        weaknesses = []
        
        if performance.get("win_rate", 0) < 40:
            weaknesses.append("Low win rate")
        if performance.get("profit_factor", 0) < 1.0:
            weaknesses.append("Poor profit factor")
        if performance.get("sharpe_ratio", 0) < 0.3:
            weaknesses.append("Poor risk-adjusted returns")
        if risk.get("volatility", 0) > 5.0:
            weaknesses.append("High volatility")
        if risk.get("max_drawdown", 0) > 20.0:
            weaknesses.append("High maximum drawdown")
        
        return weaknesses

class RangeOptimizer:
    """Optimizes BBW ranges based on historical performance."""
    
    def __init__(self, config: ConfigurationManager):
        self.config = config
        self.logger = logging.getLogger(__name__)
    
    def find_optimal_bb_range(self, df: pl.DataFrame, symbol: str) -> Dict:
        """Find optimal BBW range for a given stock."""
        try:
            # Calculate Bollinger Bands and BBW
            bb_calculator = BollingerBandCalculator(self.config)
            df_with_bb = bb_calculator.calculate_bollinger_bands(df)
            
            if df_with_bb.is_empty():
                return None
            
            # Test different BBW ranges
            range_performances = self._test_bbw_ranges(df_with_bb)
            
            # Find best performing range
            best_range = self._select_best_range(range_performances)
            
            # Generate optimization summary
            optimization_summary = self._generate_optimization_summary(range_performances, best_range)
            
            optimal_range = {
                "symbol": symbol,
                "best_range": best_range,
                "range_performances": range_performances,
                "optimization_summary": optimization_summary
            }
            
            return optimal_range
            
        except Exception as e:
            self.logger.error(f"Range optimization failed for {symbol}: {e}")
            return None
    
    def _test_bbw_ranges(self, df: pl.DataFrame) -> Dict:
        """Test performance of different BBW ranges."""
        try:
            bbw_series = df.select("bb_width").to_series()
            close_series = df.select("close").to_series()
            
            # Define range boundaries to test
            percentiles = [0.05, 0.10, 0.15, 0.20, 0.25, 0.30, 0.35, 0.40, 0.45, 0.50]
            range_performances = {}
            
            for i in range(len(percentiles) - 1):
                min_percentile = percentiles[i]
                max_percentile = percentiles[i + 1]
                
                min_bbw = bbw_series.quantile(min_percentile)
                max_bbw = bbw_series.quantile(max_percentile)
                
                range_name = f"{min_percentile*100:.0f}-{max_percentile*100:.0f}%"
                
                # Calculate performance for this range
                performance = self._calculate_range_performance(bbw_series, close_series, min_bbw, max_bbw)
                
                range_performances[range_name] = {
                    "min_bbw": min_bbw,
                    "max_bbw": max_bbw,
                    "min_percentile": min_percentile,
                    "max_percentile": max_percentile,
                    "performance": performance
                }
            
            return range_performances
            
        except Exception as e:
            self.logger.error(f"BBW range testing failed: {e}")
            return {}
    
    def _calculate_range_performance(self, bbw_series, close_series, min_bbw: float, max_bbw: float) -> Dict:
        """Calculate performance metrics for a specific BBW range."""
        try:
            # Find periods in this BBW range using Polars filter
            in_range_mask = (bbw_series >= min_bbw) & (bbw_series <= max_bbw)
            in_range_indices = []
            
            # Convert mask to indices
            for i, in_range in enumerate(in_range_mask):
                if in_range:
                    in_range_indices.append(i)
            
            if len(in_range_indices) < 5:  # Need minimum periods
                return {"avg_return": 0, "win_rate": 0, "periods": 0}
            
            # Calculate returns for periods in this range
            range_returns = []
            for i in range(1, len(in_range_indices)):
                if in_range_indices[i] - in_range_indices[i-1] == 1:  # Consecutive days
                    prev_idx = in_range_indices[i-1]
                    curr_idx = in_range_indices[i]
                    if prev_idx < len(close_series) and curr_idx < len(close_series):
                        prev_price = close_series.head(prev_idx + 1).tail(1).item()
                        curr_price = close_series.head(curr_idx + 1).tail(1).item()
                        ret = (curr_price - prev_price) / prev_price * 100
                        range_returns.append(ret)
            
            if not range_returns:
                return {"avg_return": 0, "win_rate": 0, "periods": 0}
            
            # Calculate performance metrics
            avg_return = sum(range_returns) / len(range_returns)
            win_rate = sum(1 for r in range_returns if r > 0) / len(range_returns) * 100
            max_return = max(range_returns)
            min_return = min(range_returns)
            
            return {
                "avg_return": avg_return,
                "win_rate": win_rate,
                "max_return": max_return,
                "min_return": min_return,
                "periods": len(range_returns)
            }
            
        except Exception as e:
            self.logger.error(f"Range performance calculation failed: {e}")
            return {"avg_return": 0, "win_rate": 0, "periods": 0}
    
    def _select_best_range(self, range_performances: Dict) -> Dict:
        """Select the best performing BBW range."""
        try:
            best_range = None
            best_score = -float('inf')
            
            for range_name, range_data in range_performances.items():
                performance = range_data["performance"]
                
                # Calculate composite score (win_rate * avg_return * periods_weight)
                win_rate = performance.get("win_rate", 0)
                avg_return = performance.get("avg_return", 0)
                periods = performance.get("periods", 0)
                
                # Normalize periods (0-1 scale)
                periods_weight = min(periods / 50, 1.0)  # Cap at 50 periods
                
                # Composite score
                score = win_rate * avg_return * periods_weight
                
                if score > best_score:
                    best_score = score
                    best_range = {
                        "range_name": range_name,
                        "range_data": range_data,
                        "score": score
                    }
            
            return best_range
            
        except Exception as e:
            self.logger.error(f"Best range selection failed: {e}")
            return None
    
    def _generate_optimization_summary(self, range_performances: Dict, best_range: Dict) -> Dict:
        """Generate optimization summary."""
        try:
            if not best_range:
                return {"status": "FAILED", "reason": "No valid ranges found"}
            
            best_performance = best_range["range_data"]["performance"]
            
            summary = {
                "status": "SUCCESS",
                "best_range": best_range["range_name"],
                "best_win_rate": best_performance.get("win_rate", 0),
                "best_avg_return": best_performance.get("avg_return", 0),
                "best_periods": best_performance.get("periods", 0),
                "optimization_score": best_range["score"],
                "total_ranges_tested": len(range_performances),
                "recommendation": "USE" if best_performance.get("win_rate", 0) >= 50 else "AVOID"
            }
            
            return summary
            
        except Exception as e:
            self.logger.error(f"Optimization summary generation failed: {e}")
            return {"status": "FAILED", "reason": str(e)}

class PerformanceProfileAnalyzer:
    """Generates and manages performance profiles for stocks."""
    
    def __init__(self, config: ConfigurationManager):
        self.config = config
        self.backtest_engine = BacktestEngine(config)
        self.range_optimizer = RangeOptimizer(config)
        self.logger = logging.getLogger(__name__)
    
    def generate_performance_profile(self, instrument_key: str, symbol: str, df: pl.DataFrame) -> Dict:
        """Generate comprehensive performance profile for a stock."""
        try:
            # Run backtest
            backtest_result = self.backtest_engine.backtest_squeeze_strategy(df, symbol)
            
            # Find optimal range
            optimal_range = self.range_optimizer.find_optimal_bb_range(df, symbol)
            
            # Generate profile summary
            profile_summary = self._generate_profile_summary(backtest_result, optimal_range)
            
            performance_profile = {
                "instrument_key": instrument_key,
                "symbol": symbol,
                "generation_date": datetime.now().isoformat(),
                "backtest_result": backtest_result,
                "optimal_range": optimal_range,
                "profile_summary": profile_summary
            }
            
            return performance_profile
            
        except Exception as e:
            self.logger.error(f"Performance profile generation failed for {symbol}: {e}")
            return None
    
    def _generate_profile_summary(self, backtest_result: Dict, optimal_range: Dict) -> Dict:
        """Generate summary of the performance profile."""
        try:
            if not backtest_result or not optimal_range:
                return {"status": "INCOMPLETE", "reason": "Missing backtest or optimization data"}
            
            backtest_summary = backtest_result.get("backtest_summary", {})
            optimization_summary = optimal_range.get("optimization_summary", {})
            
            # Overall profile assessment
            strategy_viability = backtest_summary.get("strategy_viability", "UNKNOWN")
            optimization_status = optimization_summary.get("status", "UNKNOWN")
            
            overall_status = "EXCELLENT"
            if strategy_viability in ["POOR", "FAIR"] or optimization_status == "FAILED":
                overall_status = "POOR"
            elif strategy_viability == "GOOD" and optimization_status == "SUCCESS":
                overall_status = "GOOD"
            
            summary = {
                "overall_status": overall_status,
                "strategy_viability": strategy_viability,
                "optimization_status": optimization_status,
                "recommendation": "USE" if overall_status in ["EXCELLENT", "GOOD"] else "AVOID",
                "key_metrics": {
                    "win_rate": backtest_result.get("performance_metrics", {}).get("win_rate", 0),
                    "profit_factor": backtest_result.get("performance_metrics", {}).get("profit_factor", 0),
                    "sharpe_ratio": backtest_result.get("performance_metrics", {}).get("sharpe_ratio", 0),
                    "optimal_range": optimal_range.get("best_range", {}).get("range_name", "UNKNOWN")
                }
            }
            
            return summary
            
        except Exception as e:
            self.logger.error(f"Profile summary generation failed: {e}")
            return {"status": "FAILED", "reason": str(e)}

# =============================================================================
# MAIN EXECUTION FLOW
# =============================================================================

def main():
    """Main execution function."""
    # Parse command line arguments
    parser = argparse.ArgumentParser(
        description="Volatility Squeeze Trading System - Comprehensive Analyzer",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter
    )
    
    # Basic analysis parameters
    parser.add_argument("--bb-period", type=int, default=20, 
                       help="Bollinger Bands period")
    parser.add_argument("--bb-std", type=float, default=2.0, 
                       help="Bollinger Bands standard deviations")
    parser.add_argument("--lookback", type=int, default=126, 
                       help="Historical lookback period (days)")
    parser.add_argument("--check-days", type=int, default=5, 
                       help="Number of recent days to check for squeeze")
    parser.add_argument("--output-file", type=str, 
                       default="volatility_squeeze_candidates.csv",
                       help="Output CSV filename")
    parser.add_argument("--blacklist", nargs='+', default=[], 
                       help="Additional symbols to blacklist (e.g., TOP10ADD ICICIB22)")
    parser.add_argument("--category", choices=['A', 'B', 'C', 'ALL'], default='ALL',
                       help="Filter by category: A=In optimal range, B=About to enter, C=Other, ALL=All categories")
    parser.add_argument("--proximity-threshold", type=float, default=10.0,
                       help="Proximity threshold percentage for category B (default: 10.0)")
    
    # Phase 3: Individual Analysis parameters
    parser.add_argument("--individual-analysis", action='store_true',
                       help="Perform detailed individual analysis for each candidate")
    parser.add_argument("--individual-symbol", type=str,
                       help="Perform individual analysis for specific symbol only")
    parser.add_argument("--detailed-report", action='store_true',
                       help="Generate detailed analysis report with all metrics")
    
    # Phase 4: Performance Profiling parameters
    parser.add_argument("--performance-profile", action='store_true',
                       help="Generate performance profiles for candidates")
    parser.add_argument("--backtest-only", action='store_true',
                       help="Run backtesting only (skip range optimization)")
    parser.add_argument("--optimize-ranges", action='store_true',
                       help="Run range optimization for all candidates")
    parser.add_argument("--profile-symbol", type=str,
                       help="Generate performance profile for specific symbol only")
    
    # Output and reporting
    parser.add_argument("--save-profiles", action='store_true',
                       help="Save performance profiles to JSON files")
    parser.add_argument("--report-format", choices=['csv', 'json', 'both'], default='csv',
                       help="Output format for reports")
    parser.add_argument("--verbose", action='store_true',
                       help="Enable verbose logging and detailed output")
    
    args = parser.parse_args()
    
    # Initialize configuration
    config = ConfigurationManager()
    
    # Update config with command line arguments
    config.trading_params['bb_period'] = args.bb_period
    config.trading_params['bb_std_dev'] = args.bb_std
    config.trading_params['lookback_period'] = args.lookback
    config.trading_params['check_period'] = args.check_days
    config.trading_params['blacklist'] = args.blacklist
    config.trading_params['proximity_threshold'] = args.proximity_threshold
    
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
        
        # Initialize analyzers
        analyzer = VolatilityAnalyzer(config, db_manager)
        individual_analyzer = IndividualAnalyzer(config)
        performance_analyzer = PerformanceProfileAnalyzer(config)
        
        # Create output directory
        output_dir = os.path.join(config.output_config['output_dir'], 
                                 config.output_config['candidates_dir'])
        os.makedirs(output_dir, exist_ok=True)
        
        # Phase 1-2: Universe Analysis
        if not args.individual_symbol and not args.profile_symbol:
            logger.info("Starting Phase 1-2: Universe Analysis")
            monitor.start_timer("universe_analysis")
            results = analyzer.analyze_universe()
            monitor.end_timer("universe_analysis")
            
            if not results:
                logger.warning("No squeeze candidates found")
                return
            
            # Convert to DataFrame and categorize results
            results_df = pl.DataFrame(results)
            
            # Filter by category if specified
            if args.category != 'ALL':
                results_df = results_df.filter(pl.col("category") == args.category)
                logger.info(f"Filtered to category {args.category}: {len(results_df)} candidates")
            
            if results_df.is_empty():
                logger.warning(f"No candidates found in category {args.category}")
                return
            
            # Sort by BBW (lowest first - tightest squeezes)
            results_df = results_df.sort("latest_bb_width")
            
            # Categorize results for separate outputs
            category_a = results_df.filter(pl.col("category") == "A")
            category_b = results_df.filter(pl.col("category") == "B")
            category_c = results_df.filter(pl.col("category") == "C")
            
            # Log category breakdown
            logger.info(f"\nCategory Breakdown:")
            logger.info(f"  Category A (In Optimal Range): {len(category_a)} candidates")
            logger.info(f"  Category B (About to Enter): {len(category_b)} candidates")
            logger.info(f"  Category C (Other Squeezes): {len(category_c)} candidates")
            logger.info(f"  Total: {len(results_df)} candidates")
            
            # Save main results to CSV
            output_file = os.path.join(output_dir, args.output_file)
            results_df.write_csv(output_file)
            
            # Save category-specific files
            if len(category_a) > 0:
                category_a_file = os.path.join(output_dir, f"category_A_{args.output_file}")
                category_a.write_csv(category_a_file)
                logger.info(f"Category A results saved to: {category_a_file}")
            
            if len(category_b) > 0:
                category_b_file = os.path.join(output_dir, f"category_B_{args.output_file}")
                category_b.write_csv(category_b_file)
                logger.info(f"Category B results saved to: {category_b_file}")
            
            if len(category_c) > 0:
                category_c_file = os.path.join(output_dir, f"category_C_{args.output_file}")
                category_c.write_csv(category_c_file)
                logger.info(f"Category C results saved to: {category_c_file}")
            
            # Display top candidates by category
            logger.info("\nTop 5 Category A candidates (In Optimal Range):")
            if len(category_a) > 0:
                print(category_a.head(5).select([
                    "symbol", "latest_bb_width", "optimal_range_avg", 
                    "proximity_percentage", "breakout_readiness", "latest_close"
                ]))
            else:
                print("None found")
            
            logger.info("\nTop 5 Category B candidates (About to Enter):")
            if len(category_b) > 0:
                print(category_b.head(5).select([
                    "symbol", "latest_bb_width", "optimal_range_avg", 
                    "proximity_percentage", "range_status", "latest_close"
                ]))
            else:
                print("None found")
            
            logger.info("\nTop 5 Category C candidates (Other Squeezes):")
            if len(category_c) > 0:
                print(category_c.head(5).select([
                    "symbol", "latest_bb_width", "squeeze_ratio", 
                    "volume_ratio", "breakout_readiness", "latest_close"
                ]))
            else:
                print("None found")
        
        # Phase 3: Individual Analysis
        if args.individual_analysis or args.individual_symbol:
            logger.info("Starting Phase 3: Individual Analysis")
            monitor.start_timer("individual_analysis")
            
            if args.individual_symbol:
                # Analyze specific symbol
                logger.info(f"Performing individual analysis for {args.individual_symbol}")
                symbol_data = analyzer.data_fetcher.get_instrument_data(args.individual_symbol)
                if symbol_data is not None:
                    individual_result = individual_analyzer.analyze_individual_stock(
                        args.individual_symbol, args.individual_symbol, symbol_data
                    )
                    if individual_result:
                        self._save_individual_analysis(individual_result, output_dir, args)
                else:
                    logger.error(f"Could not fetch data for symbol {args.individual_symbol}")
            else:
                # Analyze top candidates
                top_candidates = results_df.head(10)  # Analyze top 10 candidates
                individual_results = []
                
                for candidate in tqdm(top_candidates.iter_rows(named=True), 
                                    desc="Individual Analysis", total=len(top_candidates)):
                    symbol_data = analyzer.data_fetcher.get_instrument_data(candidate["instrument_key"])
                    if symbol_data is not None:
                        individual_result = individual_analyzer.analyze_individual_stock(
                            candidate["instrument_key"], candidate["symbol"], symbol_data
                        )
                        if individual_result:
                            individual_results.append(individual_result)
                
                # Save individual analysis results
                if individual_results:
                    self._save_individual_analysis_batch(individual_results, output_dir, args)
            
            monitor.end_timer("individual_analysis")
        
        # Phase 4: Performance Profiling
        if args.performance_profile or args.profile_symbol:
            logger.info("Starting Phase 4: Performance Profiling")
            monitor.start_timer("performance_profiling")
            
            if args.profile_symbol:
                # Profile specific symbol
                logger.info(f"Generating performance profile for {args.profile_symbol}")
                symbol_data = analyzer.data_fetcher.get_instrument_data(args.profile_symbol)
                if symbol_data is not None:
                    profile_result = performance_analyzer.generate_performance_profile(
                        args.profile_symbol, args.profile_symbol, symbol_data
                    )
                    if profile_result:
                        self._save_performance_profile(profile_result, output_dir, args)
                else:
                    logger.error(f"Could not fetch data for symbol {args.profile_symbol}")
            else:
                # Profile top candidates
                top_candidates = results_df.head(5)  # Profile top 5 candidates
                profile_results = []
                
                for candidate in tqdm(top_candidates.iter_rows(named=True), 
                                    desc="Performance Profiling", total=len(top_candidates)):
                    symbol_data = analyzer.data_fetcher.get_instrument_data(candidate["instrument_key"])
                    if symbol_data is not None:
                        profile_result = performance_analyzer.generate_performance_profile(
                            candidate["instrument_key"], candidate["symbol"], symbol_data
                        )
                        if profile_result:
                            profile_results.append(profile_result)
                
                # Save performance profile results
                if profile_results:
                    self._save_performance_profiles_batch(profile_results, output_dir, args)
            
            monitor.end_timer("performance_profiling")
        
        # Log final results
        if not args.individual_symbol and not args.profile_symbol:
            logger.info(f"Analysis complete. Found {len(results_df)} squeeze candidates")
            logger.info(f"Results saved to: {output_file}")
        
        # Log performance metrics
        monitor.end_timer("total_analysis")
        metrics = monitor.get_metrics()
        logger.info(f"Performance metrics: {metrics}")
        
    except Exception as e:
        logger.error(f"Analysis failed: {e}")
        if args.verbose:
            import traceback
            logger.error(traceback.format_exc())
    finally:
        db_manager.disconnect()

def _save_individual_analysis(individual_result: Dict, output_dir: str, args):
    """Save individual analysis results."""
    try:
        symbol = individual_result["symbol"]
        
        # Save detailed report
        if args.detailed_report:
            report_file = os.path.join(output_dir, f"individual_analysis_{symbol}.json")
            with open(report_file, 'w') as f:
                json.dump(individual_result, f, indent=2, default=str)
            logger.info(f"Detailed individual analysis saved to: {report_file}")
        
        # Save summary
        summary_file = os.path.join(output_dir, f"individual_summary_{symbol}.csv")
        summary_data = {
            "symbol": symbol,
            "analysis_date": individual_result["analysis_date"],
            "latest_close": individual_result["latest_close"],
            "latest_bb_width": individual_result["latest_bb_width"],
            "current_percentile": individual_result["historical_percentiles"].get("current_percentile_rank", 0),
            "squeeze_status": individual_result["analysis_summary"].get("squeeze_status", "UNKNOWN"),
            "recommendation": individual_result["analysis_summary"].get("recommendation", "UNKNOWN"),
            "confidence": individual_result["analysis_summary"].get("confidence", "UNKNOWN"),
            "risk_level": individual_result["analysis_summary"].get("risk_level", "UNKNOWN")
        }
        
        summary_df = pl.DataFrame([summary_data])
        summary_df.write_csv(summary_file)
        logger.info(f"Individual analysis summary saved to: {summary_file}")
        
    except Exception as e:
        logger.error(f"Failed to save individual analysis: {e}")

def _save_individual_analysis_batch(individual_results: List[Dict], output_dir: str, args):
    """Save batch individual analysis results."""
    try:
        # Save detailed reports
        if args.detailed_report:
            batch_file = os.path.join(output_dir, "individual_analysis_batch.json")
            with open(batch_file, 'w') as f:
                json.dump(individual_results, f, indent=2, default=str)
            logger.info(f"Batch individual analysis saved to: {batch_file}")
        
        # Save summary
        summary_data = []
        for result in individual_results:
            summary_data.append({
                "symbol": result["symbol"],
                "analysis_date": result["analysis_date"],
                "latest_close": result["latest_close"],
                "latest_bb_width": result["latest_bb_width"],
                "current_percentile": result["historical_percentiles"].get("current_percentile_rank", 0),
                "squeeze_status": result["analysis_summary"].get("squeeze_status", "UNKNOWN"),
                "recommendation": result["analysis_summary"].get("recommendation", "UNKNOWN"),
                "confidence": result["analysis_summary"].get("confidence", "UNKNOWN"),
                "risk_level": result["analysis_summary"].get("risk_level", "UNKNOWN")
            })
        
        summary_file = os.path.join(output_dir, "individual_analysis_summary.csv")
        summary_df = pl.DataFrame(summary_data)
        summary_df.write_csv(summary_file)
        logger.info(f"Individual analysis summary saved to: {summary_file}")
        
    except Exception as e:
        logger.error(f"Failed to save batch individual analysis: {e}")

def _save_performance_profile(profile_result: Dict, output_dir: str, args):
    """Save performance profile results."""
    try:
        symbol = profile_result["symbol"]
        
        # Save full profile
        if args.save_profiles:
            profile_file = os.path.join(output_dir, f"performance_profile_{symbol}.json")
            with open(profile_file, 'w') as f:
                json.dump(profile_result, f, indent=2, default=str)
            logger.info(f"Performance profile saved to: {profile_file}")
        
        # Save summary
        summary_file = os.path.join(output_dir, f"performance_summary_{symbol}.csv")
        summary_data = {
            "symbol": symbol,
            "generation_date": profile_result["generation_date"],
            "overall_status": profile_result["profile_summary"].get("overall_status", "UNKNOWN"),
            "strategy_viability": profile_result["profile_summary"].get("strategy_viability", "UNKNOWN"),
            "recommendation": profile_result["profile_summary"].get("recommendation", "UNKNOWN"),
            "win_rate": profile_result["profile_summary"].get("key_metrics", {}).get("win_rate", 0),
            "profit_factor": profile_result["profile_summary"].get("key_metrics", {}).get("profit_factor", 0),
            "sharpe_ratio": profile_result["profile_summary"].get("key_metrics", {}).get("sharpe_ratio", 0),
            "optimal_range": profile_result["profile_summary"].get("key_metrics", {}).get("optimal_range", "UNKNOWN")
        }
        
        summary_df = pl.DataFrame([summary_data])
        summary_df.write_csv(summary_file)
        logger.info(f"Performance profile summary saved to: {summary_file}")
        
    except Exception as e:
        logger.error(f"Failed to save performance profile: {e}")

def _save_performance_profiles_batch(profile_results: List[Dict], output_dir: str, args):
    """Save batch performance profile results."""
    try:
        # Save full profiles
        if args.save_profiles:
            batch_file = os.path.join(output_dir, "performance_profiles_batch.json")
            with open(batch_file, 'w') as f:
                json.dump(profile_results, f, indent=2, default=str)
            logger.info(f"Batch performance profiles saved to: {batch_file}")
        
        # Save summary
        summary_data = []
        for result in profile_results:
            summary_data.append({
                "symbol": result["symbol"],
                "generation_date": result["generation_date"],
                "overall_status": result["profile_summary"].get("overall_status", "UNKNOWN"),
                "strategy_viability": result["profile_summary"].get("strategy_viability", "UNKNOWN"),
                "recommendation": result["profile_summary"].get("recommendation", "UNKNOWN"),
                "win_rate": result["profile_summary"].get("key_metrics", {}).get("win_rate", 0),
                "profit_factor": result["profile_summary"].get("key_metrics", {}).get("profit_factor", 0),
                "sharpe_ratio": result["profile_summary"].get("key_metrics", {}).get("sharpe_ratio", 0),
                "optimal_range": result["profile_summary"].get("key_metrics", {}).get("optimal_range", "UNKNOWN")
            })
        
        summary_file = os.path.join(output_dir, "performance_profiles_summary.csv")
        summary_df = pl.DataFrame(summary_data)
        summary_df.write_csv(summary_file)
        logger.info(f"Performance profiles summary saved to: {summary_file}")
        
    except Exception as e:
        logger.error(f"Failed to save batch performance profiles: {e}")

if __name__ == "__main__":
    main() 