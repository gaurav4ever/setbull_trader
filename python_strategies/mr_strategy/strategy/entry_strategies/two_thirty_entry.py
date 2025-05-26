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
        self.entry_time = time(int(config.entry_candle.split(':')[0]), int(config.entry_candle.split(':')[1]))
        # Minimum price movement required from MR levels (in %)
        self.min_price_movement = 0.1  # 0.1% minimum movement
        # Trading hours
        self.market_open = time(9, 15)
        self.market_close = time(15, 30)
        self.range_high = None
        self.range_low = None
        self.range_high_entry_price = None
        self.range_low_entry_price = None
        self.in_long_trade = False
        self.in_short_trade = False
    
    async def check_entry_conditions(self, 
                               candle: Dict[str, Any], 
                               mr_values: Dict[str, Any]) -> Optional[Signal]:
        """
        Check for 2:30 PM entry conditions.
        
        Args:
            candle: The current candle data
            
        Returns:
            Signal if entry conditions are met, None otherwise
        """
        candle_info = self._format_candle_info(candle)    
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
        if candle_time < self.entry_time:
            return None
        
        if candle_time == self.entry_time:            
            # Get current price and MR values
            # convert to string self.entry_time
            self.entry_time_str = self.entry_time.strftime('%H:%M')
            logger.debug(f"{candle_info} Got {self.entry_time_str} entry time")
            self.range_high = candle['high']
            self.range_low = candle['low']
            self.range_high_entry_price = self.range_high + (self.range_high * 0.0003)
            self.range_low_entry_price = self.range_low - (self.range_low * 0.0003)
            return None
        
        if self.range_high is None or self.range_low is None:
            logger.debug(f"{candle_info} Range high or low is not set, skipping entry")
            return None
        
        logger.debug(f"{candle_info}Checking entry conditions - Price: {candle['close']}, Range High: {self.range_high}, Range Low: {self.range_low}")

        direction = self.config.instrument_key.get("direction")
        
        # Check if price is above MR high for long entry
        if candle['high'] > self.range_high_entry_price and not self.in_long_trade and not self.in_short_trade and direction == "BULLISH":
            if self.can_generate_signal(SignalType.IMMEDIATE_BREAKOUT.value, "LONG"):
                self.in_long_trade = True
                logger.info(f"{candle_info} {self.entry_time_str} long entry detected - Movement")
                signal = Signal(
                    type=SignalType.IMMEDIATE_BREAKOUT,
                    direction=SignalDirection.LONG,
                    timestamp=timestamp,
                    price=self.range_high_entry_price,
                    range_values={
                        'range_high': self.range_high,
                        'range_low': self.range_low,
                        'range_high_entry_price': self.range_high_entry_price,
                        'range_low_entry_price': self.range_low_entry_price
                    },
                    mr_values={},
                    metadata={
                        'entry_type': self.entry_time_str,
                        'entry_time': self.entry_time_str
                    }
                )
                self.update_signal_state(SignalType.IMMEDIATE_BREAKOUT.value, "LONG")
                return signal
                
        # Check if price is below MR low for short entry
        if candle['low'] < self.range_low_entry_price and not self.in_short_trade and not self.in_long_trade and direction == "BEARISH":
            if self.can_generate_signal(SignalType.IMMEDIATE_BREAKOUT.value, "SHORT"):
                self.in_short_trade = True
                logger.info(f"{candle_info} {self.entry_time_str} short entry detected")
                signal = Signal(
                    type=SignalType.IMMEDIATE_BREAKOUT,
                    direction=SignalDirection.SHORT,
                    timestamp=timestamp,
                    price=self.range_low_entry_price,
                    range_values={
                        'range_high': self.range_high,
                        'range_low': self.range_low,
                        'range_high_entry_price': self.range_high_entry_price,
                        'range_low_entry_price': self.range_low_entry_price
                    },
                    mr_values={},
                    metadata={
                        'entry_type': self.entry_time_str,
                        'entry_time': self.entry_time_str
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

    def reset_state(self) -> None:
        """Reset the entry strategy state."""
        self.in_long_trade = False
        self.in_short_trade = False
        self.range_high = None
        self.range_low = None
        self.range_high_entry_price = None
        self.range_low_entry_price = None
        self.can_generate_long = True
        self.can_generate_short = True
        self.state['can_generate_long'] = True
        self.state['can_generate_short'] = True