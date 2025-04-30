"""
Base Entry Strategy Interface

This module defines the abstract base class for all entry strategies.
Entry strategies are responsible for determining when to enter a trade.
"""

from abc import ABC, abstractmethod
from typing import Dict, Optional, List, Any
from datetime import datetime

from ..models import Signal, SignalType, SignalDirection

class EntryStrategy(ABC):
    """Abstract interface for different entry strategies."""
    
    def __init__(self, config):
        """Initialize the entry strategy."""
        self.config = config
        self.state = {}
        self.reset_state()
    
    @abstractmethod
    async def check_entry_conditions(self, 
                               candle: Dict[str, Any], 
                               mr_values: Dict[str, Any]) -> Optional[Signal]:
        """
        Check if entry conditions are met for this candle.
        
        Args:
            candle: The current candle data
            mr_values: Morning range values
            
        Returns:
            Signal object if entry conditions are met, None otherwise
        """
        pass
    
    def reset_state(self) -> None:
        """Reset the entry strategy state."""
        self.state = {
            'last_signal_time': None,
            'signals_generated': [],
            'can_generate_long': True,
            'can_generate_short': True
        }
    
    def can_generate_signal(self, signal_type: str, direction: str) -> bool:
        """
        Check if the strategy can generate a signal of the given type and direction.
        
        Args:
            signal_type: The type of signal to check
            direction: The direction of the signal (LONG or SHORT)
            
        Returns:
            True if the signal can be generated, False otherwise
        """
        direction_lower = direction.lower()
        return self.state.get(f'can_generate_{direction_lower}', True)
        
    def update_signal_state(self, signal_type: str, direction: str) -> None:
        """
        Update state after generating a signal to prevent duplicate signals.
        
        Args:
            signal_type: The type of signal generated
            direction: The direction of the signal (LONG or SHORT)
        """
        direction_lower = direction.lower()
        self.state[f'can_generate_{direction_lower}'] = False
        
        # Record signal generation time
        self.state['last_signal_time'] = datetime.now()
        
        # Add to signals generated
        if 'signals_generated' not in self.state:
            self.state['signals_generated'] = []
            
        self.state['signals_generated'].append({
            'type': signal_type,
            'direction': direction,
            'timestamp': datetime.now()
        }) 