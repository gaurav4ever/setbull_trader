"""Implementation of immediate breakout (1ST_ENTRY) strategy."""

from typing import Dict, Tuple
from .base_entry import BaseEntry
import logging

logger = logging.getLogger(__name__)

class ImmediateBreakout(BaseEntry):
    """1ST_ENTRY implementation."""
    
    def check_entry_conditions(self, candle: Dict, levels: Dict) -> Tuple[bool, str]:
        """Check for immediate breakout entry conditions."""
        # Check long entry
        if candle['high'] >= levels['long_entry']:
            return True, "LONG"
        
        # Check short entry
        if candle['low'] <= levels['short_entry']:
            return True, "SHORT"
        
        return False, "" 