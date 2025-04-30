"""
Two Thirty Entry Strategy Implementation

This module implements the 2:30 PM entry strategy (2_30_ENTRY) which looks for
trading opportunities at 2:30 PM based on the morning range.
"""

from typing import Dict, Optional, Any
from datetime import datetime, time
import logging
import pandas as pd

from .base import EntryStrategy
from ..models import Signal, SignalType, SignalDirection

logger = logging.getLogger(__name__)

class TwoThirtyEntryStrategy(EntryStrategy):
    """Implementation of the 2:30 PM entry (2_30_ENTRY) strategy."""
    
    def __init__(self, config):
        """
        Initialize the strategy.
        
        Args:
            config: Strategy configuration
        """
        super().__init__(config)
        # Default entry time is 14:30 (2:30 PM)
        self.entry_time = time(14, 30)
        # Minimum price movement required from MR levels (in %)
        self.min_price_movement = 0.1  # 0.1% minimum movement
        # Trading hours
        self.market_open = time(9, 15)
        self.market_close = time(15, 30)
    
    async def check_entry_conditions(self, 
                               candle: Dict[str, Any], 
                               mr_values: Dict[str, Any]) -> Optional[Signal]:
        """
        Check for 2:30 PM entry conditions.
        
        Args:
            candle: The current candle data
            mr_values: Morning range values
            
        Returns:
            Signal if entry conditions are met, None otherwise
        """
        candle_info = self._format_candle_info(candle)
        
        # Skip if MR values are invalid
        if not mr_values or 'mr_high' not in mr_values or 'mr_low' not in mr_values:
            logger.warning(f"{candle_info}Missing morning range high/low values")
            return None
            
        # Convert timestamp if needed
        timestamp = candle.get('timestamp')
        if isinstance(timestamp, str):
            timestamp = pd.to_datetime(timestamp)
            
        # Validate trading hours
        candle_time = timestamp.time()
        if not (self.market_open <= candle_time <= self.market_close):
            logger.debug(f"{candle_info}Outside trading hours")
            return None
            
        # Only check at entry time (2:30 PM by default)
        if candle_time != self.entry_time:
            return None
            
        # Get current price and MR values
        current_price = candle['close']
        mr_high = mr_values['mr_high']
        mr_low = mr_values['mr_low']
        mr_mid = (mr_high + mr_low) / 2
        mr_range = mr_high - mr_low
        
        logger.debug(f"{candle_info}Checking entry conditions - Price: {current_price}, MR High: {mr_high}, MR Low: {mr_low}")
        
        # Check if price is above MR high for long entry
        if current_price > mr_high:
            # Calculate price movement
            price_movement_pct = ((current_price - mr_high) / mr_high) * 100
            
            if price_movement_pct < self.min_price_movement:
                logger.debug(f"{candle_info}Price movement ({price_movement_pct:.2f}%) below minimum required ({self.min_price_movement}%)")
                return None
                
            if self.can_generate_signal(SignalType.TWO_THIRTY_ENTRY.value, "LONG"):
                logger.info(f"{candle_info}2:30 PM long entry detected - Movement: {price_movement_pct:.2f}%")
                signal = Signal(
                    type=SignalType.IMMEDIATE_BREAKOUT,
                    direction=SignalDirection.LONG,
                    timestamp=timestamp,
                    price=current_price,
                    mr_values=mr_values,
                    metadata={
                        'entry_type': '2_30_entry',
                        'mr_mid': mr_mid,
                        'price_to_mr_ratio': (current_price - mr_high) / mr_range,
                        'price_movement_pct': price_movement_pct,
                        'entry_time': candle_time.strftime('%H:%M')
                    }
                )
                self.update_signal_state(SignalType.IMMEDIATE_BREAKOUT.value, "LONG")
                return signal
                
        # Check if price is below MR low for short entry
        if current_price < mr_low:
            # Calculate price movement
            price_movement_pct = ((mr_low - current_price) / mr_low) * 100
            
            if price_movement_pct < self.min_price_movement:
                logger.debug(f"{candle_info}Price movement ({price_movement_pct:.2f}%) below minimum required ({self.min_price_movement}%)")
                return None
                
            if self.can_generate_signal(SignalType.IMMEDIATE_BREAKOUT.value, "SHORT"):
                logger.info(f"{candle_info}2:30 PM short entry detected - Movement: {price_movement_pct:.2f}%")
                signal = Signal(
                    type=SignalType.IMMEDIATE_BREAKOUT,
                    direction=SignalDirection.SHORT,
                    timestamp=timestamp,
                    price=current_price,
                    mr_values=mr_values,
                    metadata={
                        'entry_type': '2_30_entry',
                        'mr_mid': mr_mid,
                        'price_to_mr_ratio': (mr_low - current_price) / mr_range,
                        'price_movement_pct': price_movement_pct,
                        'entry_time': candle_time.strftime('%H:%M')
                    }
                )
                self.update_signal_state(SignalType.IMMEDIATE_BREAKOUT.value, "SHORT")
                return signal
                
        logger.debug(f"{candle_info}No entry conditions met")
        return None
        
    def _format_candle_info(self, candle: Dict[str, Any]) -> str:
        """Format candle information for logging."""
        if not candle:
            return ""
            
        time_str = candle.get('timestamp', 'unknown')
        if isinstance(time_str, pd.Timestamp):
            time_str = time_str.strftime('%Y-%m-%d %H:%M:%S')
            
        open_price = candle.get('open', 0)
        high_price = candle.get('high', 0)
        low_price = candle.get('low', 0)
        close_price = candle.get('close', 0)
        
        return f"[{time_str}] [O:{open_price:.2f} H:{high_price:.2f} L:{low_price:.2f} C:{close_price:.2f}] - " 