"""Base class for entry type implementations."""

from abc import ABC, abstractmethod
from typing import Dict, Tuple

class BaseEntry(ABC):
    """Abstract base class for entry types."""
    
    @abstractmethod
    def check_entry_conditions(self, candle: Dict, levels: Dict) -> Tuple[bool, str]:
        """Check entry conditions for this entry type."""
        pass 