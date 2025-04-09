"""Implementation of retest entry (RETEST_ENTRY) strategy."""

from typing import Dict, Tuple
from .base_entry import BaseEntry
import logging

logger = logging.getLogger(__name__)

class RetestEntry(BaseEntry):
    """RETEST_ENTRY implementation."""
    
    def __init__(self):
        """Initialize retest entry strategy."""
        self.breakout_confirmed = False
        self.breakout_level = None
        self.breakout_type = None

    def check_entry_conditions(self, candle: Dict, levels: Dict) -> Tuple[bool, str]:
        """Check for retest entry conditions."""
        # First check for breakout confirmation
        if not self.breakout_confirmed:
            if candle['close'] >= levels['long_entry']:
                self.breakout_confirmed = True
                self.breakout_level = levels['long_entry']
                self.breakout_type = "LONG"
            elif candle['close'] <= levels['short_entry']:
                self.breakout_confirmed = True
                self.breakout_level = levels['short_entry']
                self.breakout_type = "SHORT"
            return False, ""
        
        # Then check for retest
        if self.breakout_type == "LONG":
            if candle['low'] <= self.breakout_level and candle['close'] > self.breakout_level:
                return True, "LONG"
        else:  # SHORT
            if candle['high'] >= self.breakout_level and candle['close'] < self.breakout_level:
                return True, "SHORT"
        
        return False, "" 