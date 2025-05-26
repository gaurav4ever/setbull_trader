"""
Signal Generator for Morning Range strategy.

This module processes morning range data to generate trading signals
when price breaks out of the defined morning range.
"""

import pandas as pd
import numpy as np
from typing import Dict, List, Optional, Union, Any, Tuple
from datetime import datetime, time, timedelta
import logging

from .morning_range import MorningRangeCalculator
from .config import MRStrategyConfig, BreakoutState
from .models import Signal, SignalType, SignalDirection, SignalGroup
from ..data.data_processor import CandleProcessor
from .entry_strategies.factory import EntryStrategyFactory

# Configure logging
logger = logging.getLogger(__name__)
logger.setLevel(logging.DEBUG)

class SignalGenerator:
    """Generator for Morning Range strategy trading signals."""
    
    def __init__(self, 
                config: Optional[MRStrategyConfig] = None,
                buffer_ticks: int = 5,
                tick_size: float = 0.05,
                entry_type: str = "1ST_ENTRY"):
        """
        Initialize the Signal Generator.
        
        Args:
            config: Strategy configuration (optional)
            buffer_ticks: Number of ticks to add as buffer for entries
            tick_size: Size of one price tick
            entry_type: Type of entry strategy to use
        """
        if config is None:
            config = MRStrategyConfig(
                buffer_ticks=buffer_ticks,
                tick_size=tick_size,
                breakout_percentage=0.003,  # 0.3% default
                invalidation_percentage=0.005  # 0.5% default
            )
        
        self.config = config
        self.state = BreakoutState()
        self.active_signal_groups: List[SignalGroup] = []
        self.breakout_candles: List[Dict[str, Any]] = []  # Track candles after breakout
        self.data_processor = CandleProcessor()  # Add data processor instance
        
        # Create entry strategy
        self.entry_strategy = EntryStrategyFactory.create_strategy(entry_type, config)
        self.entry_type = entry_type
        
        # Add state tracking for signal generation
        self.last_signal_type: Optional[SignalType] = None
        self.last_signal_direction: Optional[SignalDirection] = None
        self.in_breakout_state: bool = False
        self.breakout_direction: Optional[SignalDirection] = None
        
        logger.info(f"Initialized SignalGenerator with config: {config}")
        logger.debug(f"Buffer ticks: {buffer_ticks}, Tick size: {tick_size}, Entry type: {entry_type}")
    
    def _format_candle_info(self, candle_data: Dict) -> str:
        """Format candle information for logging."""
        if not candle_data:
            return ""
            
        time_str = candle_data.get("timestamp", "unknown")
        open_price = candle_data.get("open", 0)
        high_price = candle_data.get("high", 0)
        low_price = candle_data.get("low", 0)
        close_price = candle_data.get("close", 0)
        
        return f"[{time_str}] [O:{open_price:.2f} H:{high_price:.2f} L:{low_price:.2f} C:{close_price:.2f}] - "

    def reset_signal_and_state(self) -> None:
        """Reset the signal and state."""
        self.last_signal_type = None
        self.last_signal_direction = None
        self.in_breakout_state = False
        self.breakout_direction = None
        self.entry_strategy.reset_state()
    
    async def process_candle(self, candle: Dict[str, Any], mr_values: Dict[str, Any]) -> List[Signal]:
        """
        Process a candle and generate signals based on morning range analysis.
        
        Args:
            candle: Current candle data
            mr_values: Morning range values for the day
            
        Returns:
            List of signals (empty if conditions not met)
        """
        # Skip if MR is not valid
        if self.entry_strategy.config.entry_type == "1ST_ENTRY" and not mr_values.get('is_valid', False):
            logger.debug("Skipping signal generation - Invalid MR values")
            return []
            
        # Check entry conditions using the entry strategy
        signal = await self.entry_strategy.check_entry_conditions(candle, mr_values)
        
        if signal:
            # Update signal state for backward compatibility
            self.update_signal_state(signal.type, signal.direction)
            
            # Create signal group for tracking if needed
            if signal.type in [SignalType.IMMEDIATE_BREAKOUT, SignalType.TWO_THIRTY_ENTRY]:
                signal_group = SignalGroup(
                    signals=[signal],
                    start_time=signal.timestamp,
                    end_time=signal.timestamp,
                    status='active'
                )
                self.active_signal_groups.append(signal_group)
                logger.debug(f"Created signal group with status: {signal_group.status}")
            
            logger.info(f"Generated signal: {signal}")
            return [signal]
            
        return []
    
    def process_candles(self, candles: pd.DataFrame, mr_values: Dict[str, Any]) -> List[Signal]:
        """
        Process multiple candles and generate all signals.
        
        Args:
            candles: DataFrame with candle data
            mr_values: Morning range values
            
        Returns:
            List of all generated signals
        """
        logger.info(f"Processing {len(candles)} candles")
        all_signals = []
        
        for _, candle in candles.iterrows():
            candle_dict = candle.to_dict()
            signals = self.process_candle(candle_dict, mr_values)
            all_signals.extend(signals)
        
        logger.info(f"Generated total of {len(all_signals)} signals")
        return all_signals
    
    def update_signal_state(self, signal_type: SignalType, direction: SignalDirection) -> None:
        """
        Update the signal state after generating a signal.
        
        Args:
            signal_type: Type of signal generated
            direction: Direction of the signal
        """
        self.last_signal_type = signal_type
        self.last_signal_direction = direction
        
        if signal_type == SignalType.IMMEDIATE_BREAKOUT:
            self.in_breakout_state = True
            self.breakout_direction = direction
        elif signal_type == SignalType.RETEST_ENTRY:
            # Reset breakout state after retest entry
            self.in_breakout_state = False
            self.breakout_direction = None
        elif signal_type == SignalType.TWO_THIRTY_ENTRY:
            # For 2:30 entry, we don't track breakout state
            # as it's a time-based entry
            self.in_breakout_state = False
            self.breakout_direction = None
            logger.debug(f"Updated state for 2:30 entry signal - Direction: {direction}")
    
    def reset_state(self) -> None:
        """Reset all state variables."""
        # Reset signal generator state
        self.reset_signal_and_state()
        self.active_signal_groups = []
        self.breakout_candles = []
        
        # Reset entry strategy state
        self.entry_strategy.reset_state()
        logger.debug("Reset all signal generator and entry strategy state")
    
    def scan_for_breakout(self, 
                        candles: pd.DataFrame, 
                        mr_values: Dict[str, Any],
                        entry_prices: Dict[str, float],
                        skip_morning_range: bool = True) -> Dict[str, Any]:
        """
        Scan a series of candles for the first breakout of the morning range.
        
        Args:
            candles: DataFrame with candle data
            mr_values: Morning range values dict
            entry_prices: Entry price levels dict
            skip_morning_range: If True, skip candles within the morning range time period
            
        Returns:
            Dict with breakout information or None if no breakout found
        """
        logger.info(f"Scanning {len(candles)} candles for breakout")
        logger.debug(f"MR Values - High: {mr_values.get('high')}, Low: {mr_values.get('low')}")
        logger.debug(f"Entry prices - Long: {entry_prices.get('long_entry')}, Short: {entry_prices.get('short_entry')}")
        
        if candles.empty:
            logger.warning("Empty candle data provided for breakout scanning")
            return None
        
        # Reset index if timestamp is the index
        if isinstance(candles.index, pd.DatetimeIndex):
            candles = candles.reset_index()
        
        # Log morning range values and entry prices for reference
        logger.info(f"Scanning for breakout with MR high={mr_values.get('high')}, low={mr_values.get('low')}")
        logger.info(f"Entry prices: long={entry_prices.get('long_entry')}, short={entry_prices.get('short_entry')}")
        
        # Reset state before scanning
        self.reset_state()
        
        # Process each candle
        for idx, candle in candles.iterrows():
            candle_dict = candle.to_dict()
            signals = self.process_candle(candle_dict, mr_values)
            
            if signals:
                signal = signals[0]  # Take first signal
                breakout_result = {
                    'has_breakout': True,
                    'breakout_type': signal.direction.value,
                    'breakout_price': signal.price,
                    'timestamp': signal.timestamp,
                    'candle_index': idx,
                    'signal_type': signal.type.value,
                    'entry_type': self.entry_type
                }
                logger.info(f"Breakout found: {breakout_result['breakout_type']} at index {idx}, timestamp {signal.timestamp}")
                return breakout_result
        
        # No breakout found
        logger.info("No breakout found in the provided candles")
        return None