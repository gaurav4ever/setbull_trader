"""
Morning Range Calculator for trading strategies.

This module provides functionality to calculate morning ranges (5MR or 15MR)
from candle data and evaluate their characteristics.
"""

import pandas as pd
import numpy as np
from typing import Dict, List, Optional, Union, Any, Tuple
from datetime import datetime, time, timedelta
import logging

# Import from other modules in the package
from ..data.data_processor import CandleProcessor

logger = logging.getLogger(__name__)

class MorningRangeCalculator:
    """Calculator for morning range values and characteristics."""
    
    def __init__(self, 
                range_type: str = "5MR", 
                market_open: time = time(9, 15),
                data_processor: Optional[CandleProcessor] = None,
                respect_trend: bool = True):
        """
        Initialize the Morning Range Calculator.
        
        Args:
            range_type: Type of morning range to calculate ('5MR' or '15MR')
            market_open: Market opening time
            data_processor: Optional data processor instance (will create one if None)
            respect_trend: Whether to respect the trend direction when generating signals
        """
        self.range_type = range_type
        self.market_open = market_open
        self.data_processor = data_processor or CandleProcessor()
        self.respect_trend = respect_trend
        
        # Set the end time based on range type
        self.range_end_time = None
        if range_type == '5MR':
            self.range_end_time = time(9, 20)  # 5-minute range
        elif range_type == '15MR':
            self.range_end_time = time(9, 30)  # 15-minute range
        else:
            raise ValueError(f"Invalid range type: {range_type}. Must be '5MR' or '15MR'")
        
        logger.info(f"Initialized {range_type} calculator with market open at {market_open}")
    
    def calculate_morning_range(self, candles: Union[pd.DataFrame, Dict[str, Any]]) -> Dict[str, Any]:
        """
        Calculate the morning range from candle data.
        
        Args:
            candles: Candle data as DataFrame or API response dict
            
        Returns:
            Dict with morning range details:
                - high: Morning range high
                - low: Morning range low
                - size: Size of the range (high - low)
                - candle_count: Number of candles in the range
        """
        # Process candles if they are in API response format
        if isinstance(candles, dict):
            df = self.data_processor.parse_candles(candles)
        else:
            df = candles
        
        if df.empty:
            logger.warning("Empty candle data provided for morning range calculation")
            return {
                'high': np.nan,
                'low': np.nan,
                'size': np.nan,
                'candle_count': 0,
                'status': 'error',
                'message': 'Empty candle data'
            }
        
        # Extract morning range candles and values
        mr_candles, mr_values = self.data_processor.extract_morning_range(
            df, 
            range_type=self.range_type,
            market_open=self.market_open
        )
        
        # Add status field
        if np.isnan(mr_values.get('high', np.nan)) or np.isnan(mr_values.get('low', np.nan)):
            mr_values['status'] = 'error'
            mr_values['message'] = 'Failed to calculate morning range'
        else:
            mr_values['status'] = 'success'
        
        logger.debug(f"Calculated {self.range_type}: {mr_values}")
        
        return mr_values
    
    def get_morning_range_candles(self, candles: Union[pd.DataFrame, Dict[str, Any]]) -> pd.DataFrame:
        """
        Extract only the candles within the morning range time window.
        
        Args:
            candles: Candle data as DataFrame or API response dict
            
        Returns:
            DataFrame containing only morning range candles
        """
        # Process candles if they are in API response format
        if isinstance(candles, dict):
            df = self.data_processor.parse_candles(candles)
        else:
            df = candles
        
        if df.empty:
            logger.warning("Empty candle data provided")
            return pd.DataFrame()
        
        # Extract morning range candles
        mr_candles, _ = self.data_processor.extract_morning_range(
            df, 
            range_type=self.range_type,
            market_open=self.market_open
        )
        
        return mr_candles
    
    def is_morning_range_valid(self, 
                             mr_values: Dict[str, Any],
                             min_candle_count: int = 1) -> bool:
        """
        Check if a calculated morning range is valid.
        
        Args:
            mr_values: Morning range values dict (from calculate_morning_range)
            min_candle_count: Minimum number of candles required in the range
            
        Returns:
            True if the morning range is valid, False otherwise
        """
        # Check for NaN values
        if np.isnan(mr_values.get('high', np.nan)) or np.isnan(mr_values.get('low', np.nan)):
            logger.warning("Invalid morning range: NaN values found")
            return False
        
        # Check for non-positive range size
        if mr_values.get('size', 0) <= 0:
            logger.warning("Invalid morning range: Zero or negative range size")
            return False
        
        # Check for minimum candle count
        if mr_values.get('candle_count', 0) < min_candle_count:
            logger.warning(f"Invalid morning range: Insufficient candles ({mr_values.get('candle_count', 0)} < {min_candle_count})")
            return False
        
        return True
    
    def get_entry_prices(self, 
                       mr_values: Dict[str, Any], 
                       buffer_ticks: int = 5,
                       tick_size: float = 0.05) -> Dict[str, float]:
        """
        Calculate entry prices for long and short based on morning range.
        
        Args:
            mr_values: Morning range values dict (from calculate_morning_range)
            buffer_ticks: Number of ticks to add as buffer for entries
            tick_size: Size of one price tick
            
        Returns:
            Dict with entry prices:
                - long_entry: Entry price for long positions (above high)
                - short_entry: Entry price for short positions (below low)
        """
        if not self.is_morning_range_valid(mr_values):
            logger.warning("Cannot calculate entry prices for invalid morning range")
            return {
                'long_entry': np.nan,
                'short_entry': np.nan,
                'status': 'error',
                'message': 'Invalid morning range'
            }
        
        buffer_amount = buffer_ticks * tick_size
        
        long_entry = mr_values['high'] + buffer_amount
        short_entry = mr_values['low'] - buffer_amount
        
        logger.debug(f"Calculated entry prices with {buffer_ticks} tick buffer: Long {long_entry}, Short {short_entry}")
        
        return {
            'long_entry': long_entry,
            'short_entry': short_entry,
            'buffer_ticks': buffer_ticks,
            'tick_size': tick_size,
            'buffer_amount': buffer_amount,
            'status': 'success'
        }
    
    # Phase 1.10: Extended functionality for range validation and trend detection
    
    def calculate_atr_ratio(self, 
                          mr_values: Dict[str, Any], 
                          daily_candles: Union[pd.DataFrame, Dict[str, Any]],
                          atr_period: int = 14) -> float:
        """
        Calculate the ratio of morning range size to the Average True Range (ATR).
        
        Args:
            mr_values: Morning range values dict (from calculate_morning_range)
            daily_candles: Daily candle data for ATR calculation
            atr_period: Period for ATR calculation
            
        Returns:
            Ratio of morning range size to ATR (mr_size / atr)
        """
        if not self.is_morning_range_valid(mr_values):
            logger.warning("Cannot calculate ATR ratio for invalid morning range")
            return np.nan
        
        # Process candles if they are in API response format
        if isinstance(daily_candles, dict):
            daily_df = self.data_processor.parse_candles(daily_candles)
        else:
            daily_df = daily_candles
        
        if daily_df.empty:
            logger.warning("Empty daily candle data provided for ATR calculation")
            return np.nan
        
        # Calculate ATR
        atr_df = self.data_processor.calculate_atr(daily_df, period=atr_period)
        
        # Get the latest ATR value
        latest_atr = atr_df['atr'].iloc[-1]
        
        if np.isnan(latest_atr) or latest_atr <= 0:
            logger.warning(f"Invalid ATR value: {latest_atr}")
            return np.nan
        
        # Calculate ratio
        mr_size = mr_values['size']
        atr_ratio = mr_size / latest_atr
        
        logger.debug(f"Morning range size: {mr_size}, ATR: {latest_atr}, Ratio: {atr_ratio:.2f}")
        
        return atr_ratio
    
    def is_atr_ratio_valid(self, 
                          atr_ratio: float, 
                          min_ratio: float = 0.3, 
                          max_ratio: float = 2.0) -> bool:
        """
        Check if the ATR ratio is within acceptable bounds.
        
        Args:
            atr_ratio: Ratio of morning range size to ATR
            min_ratio: Minimum acceptable ratio
            max_ratio: Maximum acceptable ratio
            
        Returns:
            True if ATR ratio is valid, False otherwise
        """
        if np.isnan(atr_ratio):
            logger.warning("Cannot validate NaN ATR ratio")
            return False
        
        if atr_ratio < min_ratio:
            logger.warning(f"ATR ratio too small: {atr_ratio:.2f} < {min_ratio}")
            return False
        
        if atr_ratio > max_ratio:
            logger.warning(f"ATR ratio too large: {atr_ratio:.2f} > {max_ratio}")
            return False
        
        logger.debug(f"ATR ratio {atr_ratio:.2f} is valid (between {min_ratio} and {max_ratio})")
        return True
    
    def calculate_ema(self, 
                    candles: Union[pd.DataFrame, Dict[str, Any]], 
                    period: int = 50) -> pd.DataFrame:
        """
        Calculate Exponential Moving Average (EMA) for trend detection.
        
        Args:
            candles: Candle data as DataFrame or API response dict
            period: EMA period (default: 50)
            
        Returns:
            DataFrame with EMA column added
        """
        # Process candles if they are in API response format
        if isinstance(candles, dict):
            df = self.data_processor.parse_candles(candles)
        else:
            df = candles.copy()
        
        if df.empty:
            logger.warning("Empty candle data provided for EMA calculation")
            return df
        
        # Ensure required column exists
        if 'close' not in df.columns:
            logger.error("DataFrame must contain 'close' column for EMA calculation")
            return df
        
        # Calculate EMA
        df[f'ema{period}'] = df['close'].ewm(span=period, adjust=False).mean()
        
        logger.debug(f"Calculated EMA{period}")
        
        return df
    
    def determine_trend(self, 
                      candles: Union[pd.DataFrame, Dict[str, Any]], 
                      ema_period: int = 50) -> str:
        """
        Determine the trend direction based on price relative to EMA.
        
        Args:
            candles: Candle data as DataFrame or API response dict
            ema_period: EMA period for trend detection
            
        Returns:
            Trend direction: 'bullish', 'bearish', or 'neutral'
        """
        # Calculate EMA
        df = self.calculate_ema(candles, period=ema_period)
        
        if df.empty:
            logger.warning("Cannot determine trend from empty data")
            return 'neutral'
        
        # Ensure required columns exist
        ema_col = f'ema{ema_period}'
        if ema_col not in df.columns or 'close' not in df.columns:
            logger.error(f"Required columns missing for trend determination")
            return 'neutral'
        
        # Get last values
        last_close = df['close'].iloc[-1]
        last_ema = df[ema_col].iloc[-1]
        
        # Determine trend direction
        if last_close > last_ema * 1.01:  # 1% buffer for strong bullish trend
            trend = 'bullish'
        elif last_close < last_ema * 0.99:  # 1% buffer for strong bearish trend
            trend = 'bearish'
        else:
            trend = 'neutral'
        
        logger.debug(f"Determined trend: {trend} (Close: {last_close}, EMA{ema_period}: {last_ema})")
        
        return trend
    
    def get_valid_signals(self, 
                        mr_values: Dict[str, Any],
                        daily_candles: Union[pd.DataFrame, Dict[str, Any]],
                        intraday_candles: Union[pd.DataFrame, Dict[str, Any]],
                        buffer_ticks: int = 5,
                        tick_size: float = 0.05,
                        min_atr_ratio: float = 0.3,
                        max_atr_ratio: float = 2.0,
                        ema_period: int = 50) -> Dict[str, Any]:
        """
        Get valid trading signals based on morning range, ATR validation, and trend.
        
        Args:
            mr_values: Morning range values dict (from calculate_morning_range)
            daily_candles: Daily candle data for ATR calculation
            intraday_candles: Intraday candle data for trend determination
            buffer_ticks: Number of ticks for entry buffer
            tick_size: Size of one price tick
            min_atr_ratio: Minimum acceptable ATR ratio
            max_atr_ratio: Maximum acceptable ATR ratio
            ema_period: EMA period for trend detection
            
        Returns:
            Dict with valid signals:
                - valid_long: True if long signal is valid
                - valid_short: True if short signal is valid
                - long_entry: Entry price for long positions
                - short_entry: Entry price for short positions
                - trend: Detected trend direction
                - atr_ratio: Ratio of morning range to ATR
                - validation_reason: Reason for signal validation/invalidation
        """
        # Check if morning range is valid
        if not self.is_morning_range_valid(mr_values):
            return {
                'valid_long': False,
                'valid_short': False,
                'long_entry': np.nan,
                'short_entry': np.nan,
                'trend': 'neutral',
                'atr_ratio': np.nan,
                'validation_reason': 'Invalid morning range'
            }
        
        # Calculate entry prices
        entry_prices = self.get_entry_prices(mr_values, buffer_ticks, tick_size)
        
        # Calculate ATR ratio
        atr_ratio = self.calculate_atr_ratio(mr_values, daily_candles)
        
        # Check if ATR ratio is valid
        atr_valid = self.is_atr_ratio_valid(atr_ratio, min_atr_ratio, max_atr_ratio)
        
        if not atr_valid:
            return {
                'valid_long': False,
                'valid_short': False,
                'long_entry': entry_prices.get('long_entry', np.nan),
                'short_entry': entry_prices.get('short_entry', np.nan),
                'trend': 'neutral',
                'atr_ratio': atr_ratio,
                'validation_reason': f'Invalid ATR ratio: {atr_ratio:.2f}'
            }
        
        # Determine trend
        trend = self.determine_trend(intraday_candles, ema_period)
        
        # Set default valid signals
        valid_long = True
        valid_short = True
        validation_reason = 'All validations passed'
        
        # Apply trend filter if respect_trend is True
        if self.respect_trend:
            if trend == 'bullish':
                valid_short = False
                validation_reason = 'Bullish trend, only long signals allowed'
            elif trend == 'bearish':
                valid_long = False
                validation_reason = 'Bearish trend, only short signals allowed'
        
        return {
            'valid_long': valid_long,
            'valid_short': valid_short,
            'long_entry': entry_prices.get('long_entry', np.nan),
            'short_entry': entry_prices.get('short_entry', np.nan),
            'trend': trend,
            'atr_ratio': atr_ratio,
            'validation_reason': validation_reason,
            'mr_high': mr_values.get('high', np.nan),
            'mr_low': mr_values.get('low', np.nan),
            'mr_size': mr_values.get('size', np.nan)
        } 