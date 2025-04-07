"""
Signal Generator for Morning Range strategy.

This module processes morning range data to generate trading signals
when price breaks out of the defined morning range.
"""

import pandas as pd
import numpy as np
from typing import Dict, List, Optional, Union, Any, Tuple
from datetime import datetime, time, timedelta
import logging

from .morning_range import MorningRangeCalculator

logger = logging.getLogger(__name__)

class SignalGenerator:
    """Generator for Morning Range strategy trading signals."""
    
    def __init__(self, 
                buffer_ticks: int = 5,
                tick_size: float = 0.05):
        """
        Initialize the Signal Generator.
        
        Args:
            buffer_ticks: Number of ticks to add as buffer for entries
            tick_size: Size of one price tick
        """
        self.buffer_ticks = buffer_ticks
        self.tick_size = tick_size
        
        logger.info(f"Initialized SignalGenerator with buffer_ticks={buffer_ticks}, tick_size={tick_size}")
    
    def check_breakout(self, 
                     candle: Dict[str, Any], 
                     mr_values: Dict[str, Any],
                     entry_prices: Dict[str, float]) -> Dict[str, Any]:
        """
        Check if a candle breaks out of the morning range.
        
        Args:
            candle: Single candle data (dict with 'high', 'low', etc.)
            mr_values: Morning range values dict
            entry_prices: Entry price levels dict
            
        Returns:
            Dict with breakout status:
                - has_breakout: True if any breakout occurred
                - breakout_type: 'long', 'short', or None
                - breakout_candle: The candle that caused the breakout
        """
        # Check if morning range values are valid
        if 'high' not in mr_values or 'low' not in mr_values:
            logger.warning("Missing morning range high/low values")
            return {
                'has_breakout': False,
                'breakout_type': None,
                'breakout_candle': candle
            }
        
        # Check if entry prices are valid
        if 'long_entry' not in entry_prices or 'short_entry' not in entry_prices:
            logger.warning("Missing entry price values")
            return {
                'has_breakout': False,
                'breakout_type': None,
                'breakout_candle': candle
            }
        
        # Get entry prices
        long_entry = entry_prices['long_entry']
        short_entry = entry_prices['short_entry']
        
        # Check for long breakout
        long_breakout = candle['high'] >= long_entry
        
        # Check for short breakout
        short_breakout = candle['low'] <= short_entry
        
        # Get timestamp info if available for better logging
        timestamp_str = candle['timestamp'].strftime('%Y-%m-%d %H:%M:%S') if 'timestamp' in candle else 'N/A'
        
        # Log the detailed condition checks
        logger.info(f"Candle at {timestamp_str} - High: {candle['high']} vs Long entry: {long_entry} = {long_breakout}")
        logger.info(f"Candle at {timestamp_str} - Low: {candle['low']} vs Short entry: {short_entry} = {short_breakout}")
        
        # Determine breakout type (prefer long if both occur in same candle)
        breakout_type = None
        if long_breakout:
            breakout_type = 'long'
        elif short_breakout:
            breakout_type = 'short'
        
        logger.info(f"Breakout check: long={long_breakout}, short={short_breakout}, type={breakout_type}")
        
        return {
            'has_breakout': long_breakout or short_breakout,
            'breakout_type': breakout_type,
            'breakout_candle': candle
        }
    
    def scan_for_breakout(self, 
                        candles: pd.DataFrame, 
                        mr_values: Dict[str, Any],
                        entry_prices: Dict[str, float],
                        skip_morning_range: bool = True) -> Dict[str, Any]:
        """
        Scan a series of candles for the first breakout of the morning range.
        
        Args:
            candles: DataFrame with candle data
            mr_values: Morning range values dict
            entry_prices: Entry price levels dict
            skip_morning_range: If True, skip candles within the morning range time period
            
        Returns:
            Dict with breakout information or None if no breakout found
        """
        if candles.empty:
            logger.warning("Empty candle data provided for breakout scanning")
            return None
        
        # Reset index if timestamp is the index
        if isinstance(candles.index, pd.DatetimeIndex):
            candles = candles.reset_index()
        
        # Log morning range values and entry prices for reference
        logger.info(f"Scanning for breakout with MR high={mr_values.get('high')}, low={mr_values.get('low')}")
        logger.info(f"Entry prices: long={entry_prices.get('long_entry')}, short={entry_prices.get('short_entry')}")
        
        morning_end_time = None
        if 'range_type' in mr_values:
            if mr_values['range_type'] == '5MR':
                morning_end_time = time(9, 20)
            elif mr_values['range_type'] == '15MR':
                morning_end_time = time(9, 30)
        
        # If not specified, use a default end time (15MR)
        if morning_end_time is None:
            morning_end_time = time(9, 30)
        
        skip_morning_range = False
        logger.info(f"Morning end time set to {morning_end_time}, skip_morning_range={skip_morning_range}")
        logger.info(f"Starting to process {len(candles)} candles for breakout detection")
        
        # Loop through candles looking for breakout
        for idx, candle in candles.iterrows():
            # Add timestamp info to the log if available
            timestamp_str = candle['timestamp'].strftime('%Y-%m-%d %H:%M:%S') if 'timestamp' in candle else 'N/A'
            
            # Skip candles within morning range if requested
            if skip_morning_range and 'timestamp' in candle:
                candle_time = candle['timestamp'].time()
                if candle_time <= morning_end_time:
                    logger.info(f"Skipping candle #{idx} at {timestamp_str} - within morning range")
                    continue
            
            # Log candle being processed with price data
            logger.info(f"Processing candle #{idx} at {timestamp_str} - O:{candle.get('open', 'N/A')} H:{candle.get('high', 'N/A')} L:{candle.get('low', 'N/A')} C:{candle.get('close', 'N/A')}")
            
            # Check for breakout
            breakout_result = self.check_breakout(candle, mr_values, entry_prices)
            
            # Log the check result for each candle
            if breakout_result['has_breakout']:
                logger.info(f"Breakout detected in candle #{idx} at {timestamp_str} - Type: {breakout_result['breakout_type']} - H:{candle.get('high', 'N/A')} L:{candle.get('low', 'N/A')}")
            else:
                logger.info(f"No breakout in candle #{idx} - High {candle.get('high', 'N/A')} vs long entry {entry_prices.get('long_entry')}, Low {candle.get('low', 'N/A')} vs short entry {entry_prices.get('short_entry')}")
            
            if breakout_result['has_breakout']:
                # Add timestamp to result
                if 'timestamp' in candle:
                    breakout_result['timestamp'] = candle['timestamp']
                
                # Add candle index
                breakout_result['candle_index'] = idx
                
                logger.info(f"Breakout found: {breakout_result['breakout_type']} at index {idx}, timestamp {timestamp_str}")
                return breakout_result
        
        # No breakout found
        logger.info("No breakout found in the provided candles")
        return None
    
    def generate_entry_signal(self, 
                            mr_calculator: MorningRangeCalculator, 
                            candles: pd.DataFrame,
                            daily_candles: Optional[pd.DataFrame] = None,
                            respect_trend: bool = True) -> Dict[str, Any]:
        """
        Generate entry signals based on morning range and candle data.
        
        Args:
            mr_calculator: Morning Range Calculator instance
            candles: Intraday candle data
            daily_candles: Daily candle data for ATR calculations
            respect_trend: Whether to respect trend direction
            
        Returns:
            Dict with signal information or None if no valid signal
        """
        if candles.empty:
            logger.warning("Empty candle data provided for signal generation")
            return None
        
        logger.info(f"Generating entry signals for {len(candles)} candles")
        
        # Calculate morning range
        mr_values = mr_calculator.calculate_morning_range(candles)
        logger.info(f"Calculated morning range - High: {mr_values.get('high')}, Low: {mr_values.get('low')}, Size: {mr_values.get('size')}")
        
        # Check if morning range is valid
        is_valid = mr_calculator.is_morning_range_valid(mr_values)
        logger.info(f"Morning range validity check: {is_valid}")
        
        if not is_valid:
            logger.warning("Invalid morning range, cannot generate signals")
            return None
        
        # Add range type to mr_values
        mr_values['range_type'] = mr_calculator.range_type
        
        # Calculate entry prices
        entry_prices = mr_calculator.get_entry_prices(
            mr_values, 
            buffer_ticks=self.buffer_ticks,
            tick_size=self.tick_size
        )
        
        logger.info(f"Entry prices calculated - Long: {entry_prices.get('long_entry')}, Short: {entry_prices.get('short_entry')}")
        
        # Apply additional validations if daily candles are provided
        if daily_candles is not None and not daily_candles.empty:
            logger.info(f"Applying trend and ATR validations with {len(daily_candles)} daily candles")
            
            # Get valid signals with trend and ATR validations
            signal_validation = mr_calculator.get_valid_signals(
                mr_values=mr_values,
                daily_candles=daily_candles,
                intraday_candles=candles,
                buffer_ticks=self.buffer_ticks,
                tick_size=self.tick_size
            )
            
            logger.info(f"Signal validation - Valid long: {signal_validation.get('valid_long')}, Valid short: {signal_validation.get('valid_short')}, Trend: {signal_validation.get('trend')}")
            
            # If no valid signals, return the validation result with error
            if not signal_validation['valid_long'] and not signal_validation['valid_short']:
                logger.warning(f"No valid signals: {signal_validation['validation_reason']}")
                signal_validation['status'] = 'error'
                signal_validation['message'] = signal_validation['validation_reason']
                return signal_validation
            
            # Set validation status
            signal_validation['status'] = 'success'
        else:
            logger.info("No daily candles provided, using basic validation without ATR and trend")
            
            # Basic validation without ATR and trend
            signal_validation = {
                'valid_long': True,
                'valid_short': True,
                'trend': 'neutral',
                'status': 'success',
                'long_entry': entry_prices.get('long_entry', np.nan),
                'short_entry': entry_prices.get('short_entry', np.nan),
                'mr_high': mr_values.get('high', np.nan),
                'mr_low': mr_values.get('low', np.nan),
                'mr_size': mr_values.get('size', np.nan)
            }
        
        # Look for actual breakout in the candle data
        logger.info("Scanning candles for breakout")
        breakout = self.scan_for_breakout(candles, mr_values, entry_prices, skip_morning_range=False)
        
        # If breakout found, add to signal validation
        if breakout is not None:
            breakout_time = breakout.get('timestamp', 'unknown time')
            logger.info(f"Breakout found: {breakout['breakout_type']} at {breakout_time}")
            
            signal_validation['breakout'] = breakout
            signal_validation['has_breakout'] = True
            signal_validation['breakout_type'] = breakout['breakout_type']
            
            # Check if this breakout direction is valid based on trend
            if breakout['breakout_type'] == 'long' and not signal_validation.get('valid_long', True):
                logger.warning(f"Long breakout found but not valid due to trend {signal_validation.get('trend', 'unknown')}")
                signal_validation['status'] = 'error'
                signal_validation['message'] = f"Long breakout found but not valid due to trend"
            elif breakout['breakout_type'] == 'short' and not signal_validation.get('valid_short', True):
                logger.warning(f"Short breakout found but not valid due to trend {signal_validation.get('trend', 'unknown')}")
                signal_validation['status'] = 'error'
                signal_validation['message'] = f"Short breakout found but not valid due to trend"
            else:
                logger.info(f"Valid {breakout['breakout_type']} breakout signal generated")
        else:
            logger.info("No breakout found in the provided candles")
            signal_validation['has_breakout'] = False
        
        return signal_validation
    
    def generate_signals_for_day(self, 
                               mr_calculator: MorningRangeCalculator,
                               intraday_candles: pd.DataFrame,
                               daily_candles: Optional[pd.DataFrame] = None,
                               trading_date: Optional[datetime] = None) -> Dict[str, Any]:
        """
        Generate signals for a specific trading day.
        
        Args:
            mr_calculator: Morning Range Calculator instance
            intraday_candles: All intraday candles
            daily_candles: Daily candle data for ATR calculations
            trading_date: Date to generate signals for (if None, use latest date in intraday_candles)
            
        Returns:
            Dict with signal information for the trading day
        """
        if intraday_candles.empty:
            logger.warning("Empty intraday candle data provided")
            return {
                'status': 'error',
                'message': 'Empty intraday candle data',
                'date': trading_date
            }
        
        # Extract candles for the specific trading day
        if trading_date is not None:
            # Reset index if timestamp is the index
            if isinstance(intraday_candles.index, pd.DatetimeIndex):
                intraday_candles = intraday_candles.reset_index()
            
            # Filter candles for the specific date
            if 'timestamp' in intraday_candles.columns:
                day_candles = intraday_candles[intraday_candles['timestamp'].dt.date == trading_date.date()]
            else:
                logger.error("Cannot filter by date: no timestamp column in candles")
                day_candles = intraday_candles
        else:
            day_candles = intraday_candles
        
        if day_candles.empty:
            logger.warning(f"No candles found for trading date {trading_date}")
            return {
                'status': 'error',
                'message': f'No candles found for trading date {trading_date}',
                'date': trading_date
            }
        
        # Generate signals
        signals = self.generate_entry_signal(
            mr_calculator=mr_calculator,
            candles=day_candles,
            daily_candles=daily_candles
        )
        
        if signals is None:
            return {
                'status': 'error',
                'message': 'Failed to generate signals',
                'date': trading_date
            }
        
        # Add date information
        if trading_date is not None:
            signals['date'] = trading_date
        elif 'timestamp' in day_candles.columns:
            signals['date'] = day_candles['timestamp'].iloc[0].date()
        
        return signals 