"""
First Entry Strategy Implementation

This module implements the first entry strategy (1ST_ENTRY) which looks for
immediate breakouts of the morning range.
"""

from typing import Dict, Optional, Any
from datetime import datetime, time
import logging
import pandas as pd

from .base import EntryStrategy
from ..models import Signal, SignalType, SignalDirection

logger = logging.getLogger(__name__)

class FirstEntryStrategy(EntryStrategy):
    """Implementation of the first entry (1ST_ENTRY) strategy."""
    
    def __init__(self, config):
        super().__init__(config)
        self.in_long_trade = False
        self.in_short_trade = False
        self.mr_high_with_buffer = None
        self.mr_low_with_buffer = None
    
    async def check_entry_conditions(self, 
                               candle: Dict[str, Any], 
                               mr_values: Dict[str, Any]) -> Optional[Signal]:
        """
        Check for immediate breakout entry conditions.
        
        Args:
            candle: The current candle data
            mr_values: Morning range values
            
        Returns:
            Signal if entry conditions are met, None otherwise
        """
        # Skip if MR values are invalid
        if not mr_values or 'mr_high' not in mr_values or 'mr_low' not in mr_values:
            logger.warning("Missing morning range high/low values")
            return None
            
        # Convert timestamp if needed
        timestamp = candle.get('timestamp')
        if isinstance(timestamp, str):
            timestamp = pd.to_datetime(timestamp)
            
        # Skip first 5min candle (9:15 AM)
        if timestamp.time() == time(9, 15):
            logger.debug("Skipping first candle of the day (9:15 AM)")
            return None
            
        # Check long breakout
        # take 0.07% buffer from mr_high
        mr_high_with_buffer, mr_low_with_buffer = self._add_buffer_to_mr_values(mr_values, 0.0007)
        if candle['high'] >= mr_high_with_buffer and not self.in_long_trade:
            if self.can_generate_signal(SignalType.IMMEDIATE_BREAKOUT.value, "LONG"):
                self.in_long_trade = True
                logger.info("Immediate long breakout detected")
                signal = Signal(
                    type=SignalType.IMMEDIATE_BREAKOUT,
                    direction=SignalDirection.LONG,
                    timestamp=timestamp,
                    price=mr_values['mr_high'],
                    mr_values=mr_values,
                    range_values={},
                    metadata={'breakout_type': 'immediate', 'entry_type': '1st_entry', 'entry_time': timestamp.strftime('%H:%M')}
                )
                self.update_signal_state(SignalType.IMMEDIATE_BREAKOUT.value, "LONG")
                return signal
            
        # Check short breakout
        if candle['low'] <= mr_low_with_buffer and not self.in_short_trade:
            if self.can_generate_signal(SignalType.IMMEDIATE_BREAKOUT.value, "SHORT"):
                self.in_short_trade = True
                logger.info("Immediate short breakout detected")
                signal = Signal(
                    type=SignalType.IMMEDIATE_BREAKOUT,
                    direction=SignalDirection.SHORT,
                    timestamp=timestamp,
                    price=mr_values['mr_low'],
                    mr_values=mr_values,
                    range_values={},
                    metadata={'breakout_type': 'immediate', 'entry_type': '1st_entry', 'entry_time': timestamp.strftime('%H:%M')}
                )
                self.update_signal_state(SignalType.IMMEDIATE_BREAKOUT.value, "SHORT")
                return signal
            
        return None 

    def _add_buffer_to_mr_values(self, mr_values, buffer_percentage):
        mr_high_with_buffer = mr_values['mr_high'] * (1 + buffer_percentage)
        mr_low_with_buffer = mr_values['mr_low'] * (1 - buffer_percentage)
        # round to 2 decimal places
        mr_high_with_buffer = round(mr_high_with_buffer, 2)
        mr_low_with_buffer = round(mr_low_with_buffer, 2)
        return mr_high_with_buffer, mr_low_with_buffer
    
    def reset_state(self) -> None:
        """Reset the entry strategy state."""
        self.in_long_trade = False
        self.in_short_trade = False
        self.mr_high_with_buffer = None
        self.mr_low_with_buffer = None
        self.can_generate_long = True
        self.can_generate_short = True
        self.state['can_generate_long'] = True
        self.state['can_generate_short'] = True
