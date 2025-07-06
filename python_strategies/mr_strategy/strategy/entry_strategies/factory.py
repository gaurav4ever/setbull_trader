"""
Entry Strategy Factory

This module provides a factory for creating entry strategy instances.
"""

from typing import Dict, Any
from .base import EntryStrategy

class EntryStrategyFactory:
    """Factory to create appropriate entry strategy instances."""
    
    @staticmethod
    def create_strategy(entry_type: str, config: Any) -> EntryStrategy:
        """
        Create entry strategy based on specified type.
        
        Args:
            entry_type: Type of entry strategy to create
            config: Strategy configuration
            
        Returns:
            EntryStrategy instance
            
        Raises:
            ValueError: If the entry type is unknown
        """
        # Import here to avoid circular imports
        from .first_entry import FirstEntryStrategy
        from .two_thirty_entry import TwoThirtyEntryStrategy
        from .bb_width_entry import BBWidthEntryStrategy
        
        if entry_type == "1ST_ENTRY":
            return FirstEntryStrategy(config)
        elif entry_type == "2_30_ENTRY":
            return TwoThirtyEntryStrategy(config)
        elif entry_type == "BB_WIDTH_ENTRY":
            return BBWidthEntryStrategy(config)
        else:
            raise ValueError(f"Unknown entry type: {entry_type}") 