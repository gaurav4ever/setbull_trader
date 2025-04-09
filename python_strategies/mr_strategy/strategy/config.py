"""
Configuration and state management for Morning Range strategy.

This module contains the configuration and state management classes
for the enhanced Morning Range strategy with breakout and retest signals.
"""

from dataclasses import dataclass
from typing import Optional
from datetime import datetime, time
import logging

logger = logging.getLogger(__name__)

@dataclass
class MRStrategyConfig:
    """
    Configuration for Morning Range strategy.
    
    Attributes:
        breakout_percentage: Percentage beyond MR for valid breakout (e.g., 0.003 for 0.3%)
        invalidation_percentage: Maximum adverse move before signal invalidates
        max_retest_candles: Maximum candles to look for retest (None for unlimited)
        buffer_ticks: Number of ticks to add as buffer for entries
        tick_size: Size of one price tick
        range_type: Type of morning range ('5MR' or '15MR')
        market_open: Market open time
        respect_trend: Whether to respect trend direction
    """
    breakout_percentage: float = 0.003  # 0.3% default
    invalidation_percentage: float = 0.005  # 0.5% default
    max_retest_candles: Optional[int] = None  # None for unlimited
    
    # Original MR parameters
    buffer_ticks: int = 5
    tick_size: float = 0.05
    range_type: str = '5MR'
    market_open: time = time(9, 15)
    respect_trend: bool = True
    
    def __post_init__(self):
        """Validate configuration values after initialization."""
        if self.breakout_percentage <= 0:
            raise ValueError("breakout_percentage must be positive")
        if self.invalidation_percentage <= 0:
            raise ValueError("invalidation_percentage must be positive")
        if self.buffer_ticks < 0:
            raise ValueError("buffer_ticks must be non-negative")
        if self.tick_size <= 0:
            raise ValueError("tick_size must be positive")
        if self.range_type not in ['5MR', '15MR']:
            raise ValueError("range_type must be either '5MR' or '15MR'")

@dataclass
class BreakoutState:
    """
    State management for breakout and retest signals.
    
    This class maintains the state of a confirmed breakout until either:
    1. A retest signal is generated
    2. The signal is invalidated
    3. The state is manually reset
    
    Attributes:
        is_breakout_confirmed: Whether a breakout has been confirmed
        breakout_type: Type of breakout ('LONG' or 'SHORT')
        breakout_price: Price at which breakout was confirmed
        breakout_time: Time at which breakout was confirmed
        mr_level: MR level being tested (MR_High for long, MR_Low for short)
        threshold_level: Price threshold for breakout confirmation
    """
    is_breakout_confirmed: bool = False
    breakout_type: Optional[str] = None  # 'LONG' or 'SHORT'
    breakout_price: Optional[float] = None
    breakout_time: Optional[datetime] = None
    mr_level: Optional[float] = None  # MR_High for long, MR_Low for short
    threshold_level: Optional[float] = None
    
    def reset(self) -> None:
        """Reset all state variables to their initial values."""
        self.is_breakout_confirmed = False
        self.breakout_type = None
        self.breakout_price = None
        self.breakout_time = None
        self.mr_level = None
        self.threshold_level = None
        logger.debug("Breakout state reset")
    
    def is_valid(self) -> bool:
        """Check if the current state is valid."""
        if not self.is_breakout_confirmed:
            return False
            
        required_fields = [
            self.breakout_type,
            self.breakout_price,
            self.breakout_time,
            self.mr_level,
            self.threshold_level
        ]
        
        return all(field is not None for field in required_fields)
    
    def to_dict(self) -> dict:
        """Convert state to dictionary for logging/debugging."""
        return {
            'is_breakout_confirmed': self.is_breakout_confirmed,
            'breakout_type': self.breakout_type,
            'breakout_price': self.breakout_price,
            'breakout_time': self.breakout_time,
            'mr_level': self.mr_level,
            'threshold_level': self.threshold_level
        } 