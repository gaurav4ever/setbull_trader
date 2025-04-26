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

# Configure logging
logger = logging.getLogger(__name__)
logger.setLevel(logging.DEBUG)

class SignalGenerator:
    """Generator for Morning Range strategy trading signals."""
    
    def __init__(self, 
                config: Optional[MRStrategyConfig] = None,
                buffer_ticks: int = 5,
                tick_size: float = 0.05):
        """
        Initialize the Signal Generator.
        
        Args:
            config: Strategy configuration (optional)
            buffer_ticks: Number of ticks to add as buffer for entries
            tick_size: Size of one price tick
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
        
        # Add state tracking for signal generation
        self.last_signal_type: Optional[SignalType] = None
        self.last_signal_direction: Optional[SignalDirection] = None
        self.in_breakout_state: bool = False
        self.breakout_direction: Optional[SignalDirection] = None
        
        logger.info(f"Initialized SignalGenerator with config: {config}")
        logger.debug(f"Buffer ticks: {buffer_ticks}, Tick size: {tick_size}")
    
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
        """
        Reset the signal and state.
        """
        self.last_signal_type = None
        self.last_signal_direction = None
        self.in_breakout_state = False
        self.breakout_direction = None
        
    def can_generate_signal(self, signal_type: SignalType, direction: SignalDirection) -> bool:
        """
        Check if we can generate a signal based on previous signals.
        
        Args:
            signal_type: Type of signal to generate
            direction: Direction of the signal
            
        Returns:
            True if signal can be generated, False otherwise
        """
        # Always allow retest entries
        if signal_type == SignalType.RETEST_ENTRY:
            return True
            
        # Don't allow consecutive immediate breakouts in same direction
        if (signal_type == SignalType.IMMEDIATE_BREAKOUT and 
            self.last_signal_type == SignalType.IMMEDIATE_BREAKOUT and
            self.last_signal_direction == direction):
            # logger.debug(f"Skipping consecutive immediate breakout in {direction} direction")
            return False
            
        # Don't allow immediate breakout if we're already in a breakout state
        if (signal_type == SignalType.IMMEDIATE_BREAKOUT and 
            self.in_breakout_state and
            self.breakout_direction == direction):
            logger.debug(f"Skipping immediate breakout in {direction} direction - already in breakout state")
            return False
            
        return True
    
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
    
    def is_valid_breakout_candle(self, candle: Dict[str, Any], threshold: float, direction: str) -> bool:
        """
        Check if a candle is a valid breakout candle.
        
        Args:
            candle: Candle data
            threshold: Breakout threshold level
            direction: 'LONG' or 'SHORT'
            
        Returns:
            True if valid breakout candle, False otherwise
        """
        logger.debug(f"Validating breakout candle - Direction: {direction}, Threshold: {threshold}")
        logger.debug(f"Candle data - Open: {candle['open']}, High: {candle['high']}, Low: {candle['low']}, Close: {candle['close']}")
        
        if direction == 'LONG':
            # For long breakout, close must be above threshold
            is_valid = candle['close'] > threshold
            logger.debug(f"Long breakout validation - Close: {candle['close']} > Threshold: {threshold} = {is_valid}")
            return is_valid
        else:
            # For short breakout, close must be below threshold
            is_valid = candle['close'] < threshold
            logger.debug(f"Short breakout validation - Close: {candle['close']} < Threshold: {threshold} = {is_valid}")
            return is_valid
    
    def check_immediate_breakout(self, 
                               candle: Dict[str, Any], 
                               mr_values: Dict[str, Any]) -> Optional[Signal]:
        """
        Check for immediate breakout (original 5MR strategy).
        
        Args:
            candle: Single candle data
            mr_values: Morning range values
            
        Returns:
            Signal if breakout detected, None otherwise
        """
        candle_info = self._format_candle_info(candle)
        logger.debug(f"{candle_info}Checking for immediate breakout")
        logger.debug(f"{candle_info}MR Values - High: {mr_values.get('high')}, Low: {mr_values.get('low')}")
        
        if not mr_values or 'high' not in mr_values or 'low' not in mr_values:
            logger.warning(f"{candle_info}Missing morning range high/low values")
            return None
            
        timestamp = candle.get('timestamp')
        if isinstance(timestamp, str):
            timestamp = pd.to_datetime(timestamp)
            
        # Check long breakout
        if candle['high'] >= mr_values['high']:
            logger.info(f"{candle_info}Immediate long breakout detected")
            logger.debug(f"{candle_info}High: {candle['high']} >= MR High: {mr_values['high']}")
            return Signal(
                type=SignalType.IMMEDIATE_BREAKOUT,
                direction=SignalDirection.LONG,
                timestamp=timestamp,
                price=mr_values['high'],
                mr_values=mr_values,
                metadata={'breakout_type': 'immediate'}
            )
            
        # Check short breakout
        if candle['low'] <= mr_values['low']:
            logger.info(f"{candle_info}Immediate short breakout detected")
            logger.debug(f"{candle_info}Low: {candle['low']} <= MR Low: {mr_values['low']}")
            return Signal(
                type=SignalType.IMMEDIATE_BREAKOUT,
                direction=SignalDirection.SHORT,
                timestamp=timestamp,
                price=mr_values['low'],
                mr_values=mr_values,
                metadata={'breakout_type': 'immediate'}
            )
            
        logger.debug(f"{candle_info}No immediate breakout detected")
        return None
    
    def check_breakout_confirmation(self, 
                                  candle: Dict[str, Any], 
                                  mr_values: Dict[str, Any]) -> Optional[Signal]:
        """
        Check if this candle confirms a breakout beyond threshold.
        
        Args:
            candle: Single candle data
            mr_values: Morning range values
            
        Returns:
            Signal if breakout confirmed, None otherwise
        """
        candle_info = self._format_candle_info(candle)
        logger.debug(f"{candle_info}Checking breakout confirmation")
        logger.debug(f"{candle_info}MR Values - High: {mr_values.get('high')}, Low: {mr_values.get('low')}")
        
        if not mr_values or 'high' not in mr_values or 'low' not in mr_values:
            logger.warning(f"{candle_info}Missing morning range high/low values")
            return None
            
        timestamp = pd.to_datetime(candle['timestamp']) if isinstance(candle['timestamp'], str) else candle['timestamp']
        
        # Calculate threshold levels
        long_threshold = mr_values['high'] * (1 + self.config.breakout_percentage)
        short_threshold = mr_values['low'] * (1 - self.config.breakout_percentage)
        
        logger.debug(f"{candle_info}Threshold levels - Long: {long_threshold}, Short: {short_threshold}")
        
        # Check long breakout confirmation
        if self.is_valid_breakout_candle(candle, long_threshold, 'LONG'):
            logger.debug(f"{candle_info}Potential long breakout detected, validating...")
            # Validate breakout with additional checks
            if self.validate_breakout(candle, mr_values, 'LONG'):
                logger.info(f"{candle_info}Long breakout confirmed")
                self.state.is_breakout_confirmed = True
                self.state.breakout_type = 'LONG'
                self.state.breakout_price = candle['close']
                self.state.breakout_time = timestamp
                self.state.mr_level = mr_values['high']
                self.state.threshold_level = long_threshold
                
                # Create signal group for this breakout
                signal_group = SignalGroup(
                    signals=[],
                    start_time=timestamp,
                    end_time=timestamp,
                    status='active'
                )
                self.active_signal_groups.append(signal_group)
                
                signal = Signal(
                    type=SignalType.BREAKOUT_CONFIRMATION,
                    direction=SignalDirection.LONG,
                    timestamp=timestamp,
                    price=candle['close'],
                    mr_values=mr_values,
                    metadata={
                        'threshold_level': long_threshold,
                        'breakout_percentage': self.config.breakout_percentage,
                        'candle_data': {
                            'open': candle['open'],
                            'high': candle['high'],
                            'low': candle['low'],
                            'close': candle['close']
                        }
                    }
                )
                
                signal_group.add_signal(signal)
                logger.debug(f"{candle_info}Created signal group with status: {signal_group.status}")
                return signal
            
        # Check short breakout confirmation
        if self.is_valid_breakout_candle(candle, short_threshold, 'SHORT'):
            logger.debug(f"{candle_info}Potential short breakout detected, validating...")
            # Validate breakout with additional checks
            if self.validate_breakout(candle, mr_values, 'SHORT'):
                logger.info(f"{candle_info}Short breakout confirmed")
                self.state.is_breakout_confirmed = True
                self.state.breakout_type = 'SHORT'
                self.state.breakout_price = candle['close']
                self.state.breakout_time = timestamp
                self.state.mr_level = mr_values['low']
                self.state.threshold_level = short_threshold
                
                # Create signal group for this breakout
                signal_group = SignalGroup(
                    signals=[],
                    start_time=timestamp,
                    end_time=timestamp,
                    status='active'
                )
                self.active_signal_groups.append(signal_group)
                
                signal = Signal(
                    type=SignalType.BREAKOUT_CONFIRMATION,
                    direction=SignalDirection.SHORT,
                    timestamp=timestamp,
                    price=candle['close'],
                    mr_values=mr_values,
                    metadata={
                        'threshold_level': short_threshold,
                        'breakout_percentage': self.config.breakout_percentage,
                        'candle_data': {
                            'open': candle['open'],
                            'high': candle['high'],
                            'low': candle['low'],
                            'close': candle['close']
                        }
                    }
                )
                
                signal_group.add_signal(signal)
                logger.debug(f"{candle_info}Created signal group with status: {signal_group.status}")
                return signal
            
        logger.debug(f"{candle_info}No breakout confirmation detected")
        return None
    
    def validate_breakout(self, candle: Dict[str, Any], mr_values: Dict[str, Any], direction: str) -> bool:
        """
        Validate a potential breakout with additional checks.
        
        Args:
            candle: Candle data
            mr_values: Morning range values
            direction: 'LONG' or 'SHORT'
            
        Returns:
            True if breakout is valid, False otherwise
        """
        logger.debug(f"Validating {direction} breakout")
        
        # Check if we're already tracking a breakout
        if self.state.is_breakout_confirmed:
            logger.debug("Already tracking a breakout, skipping validation")
            return False
            
        # Check if price has moved too far against the breakout
        if direction == 'LONG':
            invalidation_level = mr_values['high'] * (1 - self.config.invalidation_percentage)
            if candle['low'] < invalidation_level:
                logger.info(f"Long breakout invalidated - price moved below {invalidation_level}")
                logger.debug(f"Low: {candle['low']} < Invalidation: {invalidation_level}")
                return False
        else:
            invalidation_level = mr_values['low'] * (1 + self.config.invalidation_percentage)
            if candle['high'] > invalidation_level:
                logger.info(f"Short breakout invalidated - price moved above {invalidation_level}")
                logger.debug(f"High: {candle['high']} > Invalidation: {invalidation_level}")
                return False
        
        # Check if we have too many candles after breakout
        if (self.config.max_retest_candles is not None and 
            len(self.breakout_candles) >= self.config.max_retest_candles):
            logger.info("Maximum retest candles reached, invalidating breakout")
            logger.debug(f"Candles tracked: {len(self.breakout_candles)}, Max allowed: {self.config.max_retest_candles}")
            return False
        
        # Add candle to breakout tracking
        self.breakout_candles.append(candle)
        logger.debug(f"Added candle to breakout tracking. Total candles: {len(self.breakout_candles)}")
        
        return True
    
    def check_retest(self, candle: Dict[str, Any]) -> Optional[Signal]:
        """
        Check if this candle creates a valid retest signal after confirmed breakout.
        
        Args:
            candle: Single candle data
            
        Returns:
            Signal if retest detected, None otherwise
        """
        candle_info = self._format_candle_info(candle)
        logger.debug(f"{candle_info}Checking for retest")
        logger.debug(f"{candle_info}Current state - Breakout confirmed: {self.state.is_breakout_confirmed}, Type: {self.state.breakout_type}")
        
        if not self.state.is_breakout_confirmed:
            logger.debug(f"{candle_info}No breakout confirmed, skipping retest check")
            return None
            
        timestamp = pd.to_datetime(candle['timestamp']) if isinstance(candle['timestamp'], str) else candle['timestamp']
        
        # Add candle to breakout tracking
        self.breakout_candles.append(candle)
        logger.debug(f"{candle_info}Added candle to breakout tracking. Total candles: {len(self.breakout_candles)}")
        
        if self.state.breakout_type == 'LONG':
            # Check if price has tested the MR level (came down to test MR high)
            if candle['low'] <= self.state.mr_level:
                logger.debug(f"{candle_info}Price tested MR level - Low: {candle['low']} <= MR Level: {self.state.mr_level}")
                # Confirm retest with close above MR level
                if candle['close'] > self.state.mr_level:
                    logger.info(f"{candle_info}Long retest confirmed")
                    signal = Signal(
                        type=SignalType.RETEST_ENTRY,
                        direction=SignalDirection.LONG,
                        timestamp=timestamp,
                        price=candle['close'],
                        mr_values={'high': self.state.mr_level, 'threshold': self.state.threshold_level},
                        metadata={
                            'breakout_price': self.state.breakout_price,
                            'breakout_time': self.state.breakout_time,
                            'candles_since_breakout': len(self.breakout_candles),
                            'candle_data': {
                                'open': candle['open'],
                                'high': candle['high'],
                                'low': candle['low'],
                                'close': candle['close']
                            }
                        }
                    )
                    
                    # Update signal group
                    if self.active_signal_groups:
                        self.active_signal_groups[-1].add_signal(signal)
                        self.active_signal_groups[-1].status = 'completed'
                        logger.debug(f"{candle_info}Updated signal group status to: {self.active_signal_groups[-1].status}")
                    
                    self.reset_state()
                    return signal
                    
        elif self.state.breakout_type == 'SHORT':
            # Check if price has tested the MR level (came up to test MR low)
            if candle['high'] >= self.state.mr_level:
                logger.debug(f"{candle_info}Price tested MR level - High: {candle['high']} >= MR Level: {self.state.mr_level}")
                # Confirm retest with close below MR level
                if candle['close'] < self.state.mr_level:
                    logger.info(f"{candle_info}Short retest confirmed")
                    signal = Signal(
                        type=SignalType.RETEST_ENTRY,
                        direction=SignalDirection.SHORT,
                        timestamp=timestamp,
                        price=candle['close'],
                        mr_values={'low': self.state.mr_level, 'threshold': self.state.threshold_level},
                        metadata={
                            'breakout_price': self.state.breakout_price,
                            'breakout_time': self.state.breakout_time,
                            'candles_since_breakout': len(self.breakout_candles),
                            'candle_data': {
                                'open': candle['open'],
                                'high': candle['high'],
                                'low': candle['low'],
                                'close': candle['close']
                            }
                        }
                    )
                    
                    # Update signal group
                    if self.active_signal_groups:
                        self.active_signal_groups[-1].add_signal(signal)
                        self.active_signal_groups[-1].status = 'completed'
                        logger.debug(f"{candle_info}Updated signal group status to: {self.active_signal_groups[-1].status}")
                    
                    self.reset_state()
                    return signal
        
        # Check if we've exceeded max retest candles
        if (self.config.max_retest_candles is not None and 
            len(self.breakout_candles) >= self.config.max_retest_candles):
            logger.info(f"{candle_info}Maximum retest candles reached, invalidating breakout")
            logger.debug(f"{candle_info}Candles tracked: {len(self.breakout_candles)}, Max allowed: {self.config.max_retest_candles}")
            if self.active_signal_groups:
                self.active_signal_groups[-1].status = 'invalidated'
                logger.debug(f"{candle_info}Updated signal group status to: {self.active_signal_groups[-1].status}")
            self.reset_state()
        
        logger.debug(f"{candle_info}No retest detected")
        return None
    
    def reset_state(self) -> None:
        """Reset the state and clear breakout tracking."""
        logger.info("Resetting state and breakout tracking")
        logger.debug(f"Previous state - Breakout confirmed: {self.state.is_breakout_confirmed}, Type: {self.state.breakout_type}")
        self.state.reset()
        self.breakout_candles = []
        logger.debug("State and breakout tracking reset")
    
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
        if not mr_values.get('is_valid', False):
            logger.debug("Skipping signal generation - Invalid MR values")
            return []
            
        # Handle timestamp conversion
        timestamp = candle['timestamp']
        if isinstance(timestamp, pd.Timestamp):
            candle_time = timestamp.time()
        else:
            try:
                candle_time = datetime.strptime(timestamp, '%Y-%m-%d %H:%M:%S').time()
            except (TypeError, ValueError) as e:
                logger.error(f"Invalid timestamp format: {timestamp}")
                return []
            
        # Skip first 5min candle (9:15 AM)
        if candle_time == time(9, 15):
            logger.debug("Skipping first candle of the day (9:15 AM)")
            return []
            
        # Extract candle data
        high = float(candle['high'])
        low = float(candle['low'])
        close = float(candle['close'])
        
        # Extract MR values
        mr_high = mr_values['mr_high']
        mr_low = mr_values['mr_low']
        
        signals = []
        
        # Check for upper breakout
        if high > mr_high:
            if self.can_generate_signal(SignalType.IMMEDIATE_BREAKOUT, SignalDirection.LONG):
                signals.append(Signal(
                    type=SignalType.IMMEDIATE_BREAKOUT,
                    direction=SignalDirection.LONG,
                    price=mr_high,
                    timestamp=candle['timestamp'],
                    mr_values=mr_values
                ))
                logger.info(f"Generated upper breakout signal at {candle['timestamp']}")
                self.update_signal_state(SignalType.IMMEDIATE_BREAKOUT, SignalDirection.LONG)
            
        # Check for lower breakout
        elif low < mr_low:
            if self.can_generate_signal(SignalType.IMMEDIATE_BREAKOUT, SignalDirection.SHORT):
                signals.append(Signal(
                    type=SignalType.IMMEDIATE_BREAKOUT,
                    direction=SignalDirection.SHORT,
                    price=mr_low,
                    timestamp=candle['timestamp'],
                    mr_values=mr_values
                ))
                logger.info(f"Generated lower breakout signal at {candle['timestamp']}")
                self.update_signal_state(SignalType.IMMEDIATE_BREAKOUT, SignalDirection.SHORT)
            
        return signals
    
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
    
    def check_breakout(self, 
                     candle: Dict[str, Any], 
                     mr_values: Dict[str, Any],
                     entry_prices: Dict[str, float]) -> Dict[str, Any]:
        """
        Check if a candle breaks out of the morning range.
        
        Args:
            candle: Single candle data (dict with 'high', 'low', etc.)
            mr_values: Morning range values dict
            entry_prices: Entry price levels dict
            
        Returns:
            Dict with breakout status:
                - has_breakout: True if any breakout occurred
                - breakout_type: 'long', 'short', or None
                - breakout_candle: The candle that caused the breakout
        """
        logger.debug("Checking for breakout")
        logger.debug(f"Entry prices - Long: {entry_prices.get('long_entry')}, Short: {entry_prices.get('short_entry')}")
        logger.debug(f"Candle data - High: {candle['high']}, Low: {candle['low']}")
        
        # Check if morning range values are valid
        if 'high' not in mr_values or 'low' not in mr_values:
            logger.warning("Missing morning range high/low values")
            return {
                'has_breakout': False,
                'breakout_type': None,
                'breakout_candle': candle
            }
        
        # Check if entry prices are valid
        if 'long_entry' not in entry_prices or 'short_entry' not in entry_prices:
            logger.warning("Missing entry price values")
            return {
                'has_breakout': False,
                'breakout_type': None,
                'breakout_candle': candle
            }
        
        # Get entry prices
        long_entry = entry_prices['long_entry']
        short_entry = entry_prices['short_entry']
        
        # Check for long breakout
        long_breakout = candle['high'] >= long_entry
        logger.debug(f"Long breakout check - High: {candle['high']} >= Entry: {long_entry} = {long_breakout}")
        
        # Check for short breakout
        short_breakout = candle['low'] <= short_entry
        logger.debug(f"Short breakout check - Low: {candle['low']} <= Entry: {short_entry} = {short_breakout}")
        
        # Get timestamp info if available for better logging
        timestamp_str = candle['timestamp'].strftime('%Y-%m-%d %H:%M:%S') if 'timestamp' in candle else 'N/A'
        
        # Log the detailed condition checks
        logger.info(f"Candle at {timestamp_str} - High: {candle['high']} vs Long entry: {long_entry} = {long_breakout}")
        logger.info(f"Candle at {timestamp_str} - Low: {candle['low']} vs Short entry: {short_entry} = {short_breakout}")
        
        # Determine breakout type (prefer long if both occur in same candle)
        breakout_type = None
        if long_breakout:
            breakout_type = 'long'
        elif short_breakout:
            breakout_type = 'short'
        
        logger.info(f"Breakout check: long={long_breakout}, short={short_breakout}, type={breakout_type}")
        
        return {
            'has_breakout': long_breakout or short_breakout,
            'breakout_type': breakout_type,
            'breakout_candle': candle
        }
    
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
        
        morning_end_time = None
        if 'range_type' in mr_values:
            if mr_values['range_type'] == '5MR':
                morning_end_time = time(9, 20)
            elif mr_values['range_type'] == '15MR':
                morning_end_time = time(9, 30)
        
        # If not specified, use a default end time (15MR)
        if morning_end_time is None:
            morning_end_time = time(9, 30)
        
        skip_morning_range = False
        logger.info(f"Morning end time set to {morning_end_time}, skip_morning_range={skip_morning_range}")
        logger.info(f"Starting to process {len(candles)} candles for breakout detection")
        
        # Loop through candles looking for breakout
        for idx, candle in candles.iterrows():
            # Add timestamp info to the log if available
            timestamp_str = candle['timestamp'].strftime('%Y-%m-%d %H:%M:%S') if 'timestamp' in candle else 'N/A'
            
            # Skip candles within morning range if requested
            if skip_morning_range and 'timestamp' in candle:
                candle_time = candle['timestamp'].time()
                if candle_time <= morning_end_time:
                    logger.info(f"Skipping candle #{idx} at {timestamp_str} - within morning range")
                    continue
            
            # Log candle being processed with price data
            logger.info(f"Processing candle #{idx} at {timestamp_str} - O:{candle.get('open', 'N/A')} H:{candle.get('high', 'N/A')} L:{candle.get('low', 'N/A')} C:{candle.get('close', 'N/A')}")
            
            # Check for breakout
            breakout_result = self.check_breakout(candle, mr_values, entry_prices)
            
            # Log the check result for each candle
            if breakout_result['has_breakout']:
                logger.info(f"Breakout detected in candle #{idx} at {timestamp_str} - Type: {breakout_result['breakout_type']} - H:{candle.get('high', 'N/A')} L:{candle.get('low', 'N/A')}")
            else:
                logger.info(f"No breakout in candle #{idx} - High {candle.get('high', 'N/A')} vs long entry {entry_prices.get('long_entry')}, Low {candle.get('low', 'N/A')} vs short entry {entry_prices.get('short_entry')}")
            
            if breakout_result['has_breakout']:
                # Add timestamp to result
                if 'timestamp' in candle:
                    breakout_result['timestamp'] = candle['timestamp']
                
                # Add candle index
                breakout_result['candle_index'] = idx
                
                logger.info(f"Breakout found: {breakout_result['breakout_type']} at index {idx}, timestamp {timestamp_str}")
                return breakout_result
        
        # No breakout found
        logger.info("No breakout found in the provided candles")
        return None
    
    def generate_entry_signal(self, 
                            mr_calculator: MorningRangeCalculator, 
                            candles: pd.DataFrame,
                            daily_candles: Optional[pd.DataFrame] = None,
                            respect_trend: bool = True) -> Dict[str, Any]:
        """
        Generate entry signals based on morning range and candle data.
        
        Args:
            mr_calculator: Morning Range Calculator instance
            candles: Intraday candle data
            daily_candles: Daily candle data for ATR calculations
            respect_trend: Whether to respect trend direction
            
        Returns:
            Dict with signal information or None if no valid signal
        """
        logger.info(f"Generating entry signals for {len(candles)} candles")
        logger.debug(f"Respect trend: {respect_trend}")
        
        if candles.empty:
            logger.warning("Empty candle data provided for signal generation")
            return None
        
        # Calculate morning range
        mr_values = mr_calculator.calculate_morning_range(candles)
        logger.info(f"Calculated morning range - High: {mr_values.get('high')}, Low: {mr_values.get('low')}, Size: {mr_values.get('size')}")
        
        # Check if morning range is valid
        is_valid = mr_calculator.is_morning_range_valid(mr_values)
        logger.info(f"Morning range validity check: {is_valid}")
        
        if not is_valid:
            logger.warning("Invalid morning range, cannot generate signals")
            return None
        
        # Add range type to mr_values
        mr_values['range_type'] = mr_calculator.range_type
        
        # Calculate entry prices
        entry_prices = mr_calculator.get_entry_prices(
            mr_values, 
            buffer_ticks=self.config.buffer_ticks,
            tick_size=self.config.tick_size
        )
        
        logger.info(f"Entry prices calculated - Long: {entry_prices.get('long_entry')}, Short: {entry_prices.get('short_entry')}")
        
        # Apply additional validations if daily candles are provided
        if daily_candles is not None and not daily_candles.empty:
            logger.info(f"Applying trend and ATR validations with {len(daily_candles)} daily candles")
            
            # Get valid signals with trend and ATR validations
            signal_validation = mr_calculator.get_valid_signals(
                mr_values=mr_values,
                daily_candles=daily_candles,
                intraday_candles=candles,
                buffer_ticks=self.config.buffer_ticks,
                tick_size=self.config.tick_size
            )
            
            logger.info(f"Signal validation - Valid long: {signal_validation.get('valid_long')}, Valid short: {signal_validation.get('valid_short')}, Trend: {signal_validation.get('trend')}")
            
            # If no valid signals, return the validation result with error
            if not signal_validation['valid_long'] and not signal_validation['valid_short']:
                logger.warning(f"No valid signals: {signal_validation['validation_reason']}")
                signal_validation['status'] = 'error'
                signal_validation['message'] = signal_validation['validation_reason']
                return signal_validation
            
            # Set validation status
            signal_validation['status'] = 'success'
        else:
            logger.info("No daily candles provided, using basic validation without ATR and trend")
            
            # Basic validation without ATR and trend
            signal_validation = {
                'valid_long': True,
                'valid_short': True,
                'trend': 'neutral',
                'status': 'success',
                'long_entry': entry_prices.get('long_entry', np.nan),
                'short_entry': entry_prices.get('short_entry', np.nan),
                'mr_high': mr_values.get('high', np.nan),
                'mr_low': mr_values.get('low', np.nan),
                'mr_size': mr_values.get('size', np.nan)
            }
        
        # Look for actual breakout in the candle data
        logger.info("Scanning candles for breakout")
        breakout = self.scan_for_breakout(candles, mr_values, entry_prices, skip_morning_range=False)
        
        # If breakout found, add to signal validation
        if breakout is not None:
            breakout_time = breakout.get('timestamp', 'unknown time')
            logger.info(f"Breakout found: {breakout['breakout_type']} at {breakout_time}")
            
            signal_validation['breakout'] = breakout
            signal_validation['has_breakout'] = True
            signal_validation['breakout_type'] = breakout['breakout_type']
            
            # Check if this breakout direction is valid based on trend
            if breakout['breakout_type'] == 'long' and not signal_validation.get('valid_long', True):
                logger.warning(f"Long breakout found but not valid due to trend {signal_validation.get('trend', 'unknown')}")
                signal_validation['status'] = 'error'
                signal_validation['message'] = f"Long breakout found but not valid due to trend"
            elif breakout['breakout_type'] == 'short' and not signal_validation.get('valid_short', True):
                logger.warning(f"Short breakout found but not valid due to trend {signal_validation.get('trend', 'unknown')}")
                signal_validation['status'] = 'error'
                signal_validation['message'] = f"Short breakout found but not valid due to trend"
            else:
                logger.info(f"Valid {breakout['breakout_type']} breakout signal generated")
        else:
            logger.info("No breakout found in the provided candles")
            signal_validation['has_breakout'] = False
        
        return signal_validation
    
    def generate_signals_for_day(self, 
                               mr_calculator: MorningRangeCalculator,
                               intraday_candles: pd.DataFrame,
                               daily_candles: Optional[pd.DataFrame] = None,
                               trading_date: Optional[datetime] = None) -> Dict[str, Any]:
        """
        Generate signals for a specific trading day.
        
        Args:
            mr_calculator: Morning Range Calculator instance
            intraday_candles: All intraday candles
            daily_candles: Daily candle data for ATR calculations
            trading_date: Date to generate signals for (if None, use latest date in intraday_candles)
            
        Returns:
            Dict with signal information for the trading day
        """
        logger.info(f"Generating signals for trading date: {trading_date}")
        
        if intraday_candles.empty:
            logger.warning("Empty intraday candle data provided")
            return {
                'status': 'error',
                'message': 'Empty intraday candle data',
                'date': trading_date
            }
        
        # Extract candles for the specific trading day
        if trading_date is not None:
            # Reset index if timestamp is the index
            if isinstance(intraday_candles.index, pd.DatetimeIndex):
                intraday_candles = intraday_candles.reset_index()
            
            # Filter candles for the specific date
            if 'timestamp' in intraday_candles.columns:
                day_candles = intraday_candles[intraday_candles['timestamp'].dt.date == trading_date.date()]
                logger.info(f"Filtered {len(day_candles)} candles for date {trading_date.date()}")
            else:
                logger.error("Cannot filter by date: no timestamp column in candles")
                day_candles = intraday_candles
        else:
            day_candles = intraday_candles
        
        if day_candles.empty:
            logger.warning(f"No candles found for trading date {trading_date}")
            return {
                'status': 'error',
                'message': f'No candles found for trading date {trading_date}',
                'date': trading_date
            }
        
        # Generate signals
        signals = self.generate_entry_signal(
            mr_calculator=mr_calculator,
            candles=day_candles,
            daily_candles=daily_candles
        )
        
        if signals is None:
            logger.warning("Failed to generate signals")
            return {
                'status': 'error',
                'message': 'Failed to generate signals',
                'date': trading_date
            }
        
        # Add date information
        if trading_date is not None:
            signals['date'] = trading_date
        elif 'timestamp' in day_candles.columns:
            signals['date'] = day_candles['timestamp'].iloc[0].date()
        
        logger.info(f"Successfully generated signals for date: {signals.get('date')}")
        return signals 