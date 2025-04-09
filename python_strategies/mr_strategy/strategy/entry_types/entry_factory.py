"""Factory for creating entry type strategies."""

from typing import Dict
from .base_entry import BaseEntry
from .immediate_breakout import ImmediateBreakout
from .retest_entry import RetestEntry

class EntryFactory:
    """Factory class for creating entry strategies."""
    
    @staticmethod
    def create_entry_strategy(entry_type: str) -> BaseEntry:
        """Create entry strategy instance."""
        if entry_type == "1ST_ENTRY":
            return ImmediateBreakout()
        elif entry_type == "RETEST_ENTRY":
            return RetestEntry()
        else:
            raise ValueError(f"Unknown entry type: {entry_type}") 