"""
Data processor for candle data.

This module processes candle data from the API into formats suitable for strategy calculations,
with a focus on morning range extraction.
"""

import pandas as pd
import numpy as np
from typing import Dict, List, Optional, Union, Any, Tuple
from datetime import datetime, time, timedelta
import logging

logger = logging.getLogger(__name__)

class CandleProcessor:
    """Process and transform candle data for strategy calculations."""
    
    def __init__(self):
        """Initialize the candle processor."""
        pass
    
    @staticmethod
    def parse_candles(candle_data: Dict[str, Any]) -> pd.DataFrame:
        """
        Parse candle data from API response into a pandas DataFrame.
        
        Args:
            candle_data: API response containing candle data
            
        Returns:
            DataFrame with candle data
        """

        # Check if the response contains data
        # if not candle_data or 'data' not in candle_data:
        #     logger.warning("No candle data found in API response")
        #     return pd.DataFrame()
        
        # Extract candles from the response
        candles = candle_data
        # if not candles:
        #     logger.warning("Empty candle list in API response")
        #     return pd.DataFrame()
        
        # Convert to DataFrame
        df = pd.DataFrame(candles)
        
        # Convert timestamp to datetime
        if 'timestamp' in df.columns:
            df['timestamp'] = pd.to_datetime(df['timestamp'])
            df.set_index('timestamp', inplace=True)
        
        # Ensure numeric columns are the correct type
        numeric_cols = ['open', 'high', 'low', 'close', 'volume', 'openInterest']
        for col in numeric_cols:
            if col in df.columns:
                df[col] = pd.to_numeric(df[col], errors='coerce')
        
        return df
    
    def extract_morning_range(self, 
                            df: pd.DataFrame, 
                            range_type: str = '5MR',
                            market_open: time = time(9, 15),
                            tz=None) -> Tuple[pd.DataFrame, Dict[str, float]]:
        """
        Extract the morning range from candle data.
        
        Args:
            df: DataFrame with candle data
            range_type: Type of morning range ('5MR' or '15MR')
            market_open: Market opening time
            tz: Timezone for the data (if None, assumed to be in local timezone)
            
        Returns:
            Tuple containing:
                - DataFrame with only the morning range candles
                - Dict with morning range values (high, low, size)
        """
        if df.empty:
            logger.warning("Empty DataFrame provided for morning range extraction")
            return df, {'high': np.nan, 'low': np.nan, 'size': np.nan}
        
        # Reset index if timestamp is the index
        if isinstance(df.index, pd.DatetimeIndex):
            df = df.reset_index()
        
        # Ensure timestamp column exists
        if 'timestamp' not in df.columns:
            logger.error("DataFrame must contain a 'timestamp' column")
            return df, {'high': np.nan, 'low': np.nan, 'size': np.nan}
        
        # Filter only the candles for the morning range calculation
        morning_end_time = None
        if range_type == '5MR':
            # 5-minute morning range: 9:15 to 9:20
            morning_end_time = time(9, 20)
        elif range_type == '15MR':
            # 15-minute morning range: 9:15 to 9:30
            morning_end_time = time(9, 30)
        else:
            logger.error(f"Invalid range type: {range_type}")
            return df, {'high': np.nan, 'low': np.nan, 'size': np.nan}
        
        # Filter candles within the morning range time window
        def is_in_morning_range(timestamp):
            t = timestamp.time()
            return market_open <= t < morning_end_time
        
        morning_candles = df[df['timestamp'].apply(is_in_morning_range)]
        
        if morning_candles.empty:
            logger.warning(f"No candles found within the {range_type} time window")
            return morning_candles, {'high': np.nan, 'low': np.nan, 'size': np.nan}
        
        # Calculate the morning range values
        mr_high = morning_candles['high'].max()
        mr_low = morning_candles['low'].min()
        mr_size = mr_high - mr_low
        
        mr_values = {
            'high': mr_high,
            'low': mr_low,
            'size': mr_size,
            'candle_count': len(morning_candles)
        }
        
        logger.debug(f"Extracted {range_type} values: high={mr_high}, low={mr_low}, size={mr_size}")
        
        return morning_candles, mr_values
    
    def filter_trading_day_candles(self, 
                                 df: pd.DataFrame, 
                                 trading_date: Optional[datetime] = None,
                                 market_open: time = time(9, 15),
                                 market_close: time = time(15, 30),
                                 tz=None) -> pd.DataFrame:
        """
        Filter candles for a specific trading day.
        
        Args:
            df: DataFrame with candle data
            trading_date: Date to filter (if None, use the latest date in the data)
            market_open: Market opening time
            market_close: Market closing time
            tz: Timezone for the data
            
        Returns:
            DataFrame with filtered candles
        """
        logger.info(f"Filtering trading day candles for date: {trading_date}")
        if df.empty:
            return df
        
        # Reset index if timestamp is the index
        if isinstance(df.index, pd.DatetimeIndex):
            df = df.reset_index()
        
        # Ensure timestamp column exists
        if 'timestamp' not in df.columns:
            logger.error("DataFrame must contain a 'timestamp' column")
            return df
        
        # Determine the trading date if not provided
        if trading_date is None:
            trading_date = df['timestamp'].max().date()
        else:
            trading_date = trading_date.date()
        
        logger.debug(f"Filtering candles for trading date: {trading_date}")
        
        # Filter candles for the trading day and within market hours
        def is_in_trading_day(timestamp):
            if timestamp.date() != trading_date:
                return False
            
            t = timestamp.time()
            return market_open <= t <= market_close
        
        return df[df['timestamp'].apply(is_in_trading_day)]
    
    # Phase 1.7: Added functions for ATR calculation and trading day handling
    
    def calculate_atr(self, df: pd.DataFrame, period: int = 14) -> pd.DataFrame:
        """
        Calculate Average True Range (ATR) for a given period.
        
        Args:
            df: DataFrame with candle data (must contain 'high', 'low', 'close' columns)
            period: Period for ATR calculation (default: 14)
            
        Returns:
            DataFrame with ATR column added
        """
        if df.empty:
            logger.warning("Empty DataFrame provided for ATR calculation")
            return df
        
        # Ensure required columns exist
        required_cols = ['high', 'low', 'close']
        if not all(col in df.columns for col in required_cols):
            logger.error(f"DataFrame must contain columns: {required_cols}")
            return df
        
        # Make a copy to avoid modifying the original
        result = df.copy()
        
        # Calculate True Range
        result['tr0'] = result['high'] - result['low']  # Current high - current low
        result['tr1'] = abs(result['high'] - result['close'].shift(1))  # Current high - previous close
        result['tr2'] = abs(result['low'] - result['close'].shift(1))  # Current low - previous close
        result['tr'] = result[['tr0', 'tr1', 'tr2']].max(axis=1)
        
        # Calculate ATR (Simple Moving Average of True Range)
        result['atr'] = result['tr'].rolling(window=period).mean()
        
        # Clean up intermediate columns
        result = result.drop(['tr0', 'tr1', 'tr2', 'tr'], axis=1)
        
        logger.debug(f"Calculated ATR with period {period}")
        
        return result
    
    def get_valid_trading_dates(self, df: pd.DataFrame) -> List[datetime.date]:
        """
        Extract all valid trading dates from a DataFrame of candles.
        
        Args:
            df: DataFrame with candle data (must contain 'timestamp' column)
            
        Returns:
            List of unique trading dates in ascending order
        """
        if df.empty:
            return []
        
        # Reset index if timestamp is the index
        if isinstance(df.index, pd.DatetimeIndex):
            df = df.reset_index()
        
        # Ensure timestamp column exists
        if 'timestamp' not in df.columns:
            logger.error("DataFrame must contain a 'timestamp' column")
            return []
        
        # Extract unique dates
        dates = df['timestamp'].dt.date.unique()
        
        # Sort in ascending order
        dates.sort()
        
        return list(dates)
    
    def is_valid_trading_day(self, date: datetime.date, 
                            market_holidays: Optional[List[datetime.date]] = None,
                            weekend_days: Optional[List[int]] = None) -> bool:
        """
        Check if a given date is a valid trading day.
        
        Args:
            date: Date to check
            market_holidays: List of market holidays (dates)
            weekend_days: List of weekend day numbers (0=Monday, 6=Sunday)
                         Default is [5, 6] for Saturday and Sunday
            
        Returns:
            True if it's a valid trading day, False otherwise
        """
        if weekend_days is None:
            weekend_days = [5, 6]  # Saturday and Sunday
            
        if market_holidays is None:
            market_holidays = []
            
        # Check if it's a weekend
        if date.weekday() in weekend_days:
            return False
            
        # Check if it's a holiday
        if date in market_holidays:
            return False
            
        return True