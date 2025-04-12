"""
Base Strategy Framework for Morning Range Strategy.

This module provides the base classes and interfaces for implementing
different variants of the Morning Range strategy.
"""

from abc import ABC, abstractmethod
from dataclasses import dataclass
from enum import Enum
from typing import Dict, Optional, Union, List, Tuple
import logging
from datetime import datetime, time
import pandas as pd

logger = logging.getLogger(__name__)

class StrategyState(Enum):
    """Strategy execution states."""
    INITIALIZED = "initialized"
    WAITING_FOR_RANGE = "waiting_for_range"
    RANGE_CALCULATED = "range_calculated"
    WAITING_FOR_SETUP = "waiting_for_setup"
    SETUP_CONFIRMED = "setup_confirmed"
    IN_POSITION = "in_position"
    POSITION_CLOSED = "position_closed"
    ERROR = "error"

@dataclass
class StrategyConfig:
    """Base configuration for strategy."""
    instrument_key: str
    range_type: str  # '5MR' or '15MR'
    entry_type: str  # '1ST_ENTRY' or 'RETEST_ENTRY'
    sl_percentage: float
    target_r: float
    buffer_ticks: int
    tick_size: float
    respect_trend: bool = True
    enable_trailing: bool = False
    partial_exits: bool = False
    breakout_percentage: float = 0.003
    invalidation_percentage: float = 0.005
    max_retest_candles: Optional[int] = None

class BaseStrategy(ABC):
    """Abstract base class for strategy implementation."""
    
    def __init__(self, config: StrategyConfig):
        """Initialize base strategy."""
        self.config = config
        self.state = StrategyState.INITIALIZED
        self.morning_range: Dict = {}
        self.entry_levels: Dict = {}
        self.position: Dict = {}
        self.trade_metrics: Dict = {}
        
        logger.info(f"Initialized {self.__class__.__name__} with config: {config}")

    @abstractmethod
    async def calculate_morning_range(self, candles: pd.DataFrame) -> Dict:
        """
        Calculate morning range values.
        
        Args:
            candles: DataFrame containing candle data
            
        Returns:
            Dictionary containing morning range values and validation status
        """
        pass

    @abstractmethod
    def validate_setup(self, candles: pd.DataFrame) -> bool:
        """Validate strategy setup conditions."""
        pass

    @abstractmethod
    def check_entry_conditions(self, candle: Dict) -> Tuple[bool, str]:
        """Check entry conditions."""
        pass

    @abstractmethod
    def check_exit_conditions(self, candle: Dict) -> Tuple[bool, str]:
        """Check exit conditions."""
        pass

    def update_state(self, new_state: StrategyState):
        """Update strategy state."""
        logger.info(f"Strategy state transition: {self.state.value} -> {new_state.value}")
        self.state = new_state 