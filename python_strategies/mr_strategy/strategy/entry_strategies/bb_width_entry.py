"""
BB Width Entry Strategy Implementation

This module implements the BB Width entry strategy (BB_WIDTH_ENTRY) which looks for
volatility squeeze conditions using Bollinger Bands width analysis.
"""

from typing import Dict, Optional, Any
from datetime import datetime, time
import logging
import pandas as pd
import numpy as np

from .base import EntryStrategy
from ..models import Signal, SignalType, SignalDirection

logger = logging.getLogger(__name__)

class BBWidthEntryStrategy(EntryStrategy):
    """Implementation of the BB Width entry (BB_WIDTH_ENTRY) strategy."""
    
    def __init__(self, config):
        """
        Initialize the strategy.
        
        Args:
            config: Strategy configuration
        """
        super().__init__(config)
        
        # BB Width strategy specific parameters
        self.bb_width_threshold = getattr(config, 'bb_width_threshold', 0.2)  # 20% default
        self.bb_period = getattr(config, 'bb_period', 20)  # 20 period default
        self.bb_std_dev = getattr(config, 'bb_std_dev', 2.0)  # 2 standard deviations default
        self.squeeze_duration_min = getattr(config, 'squeeze_duration_min', 3)  # Minimum 3 candles
        self.squeeze_duration_max = getattr(config, 'squeeze_duration_max', 5)  # Maximum 5 candles
        
        # Trading hours
        self.market_open = time(9, 15)
        self.market_close = time(15, 30)
        
        # Strategy state variables
        self.in_long_trade = False
        self.in_short_trade = False
        self.squeeze_detected = False
        self.squeeze_start_time = None
        self.squeeze_candle_count = 0
        self.lowest_bb_width = float('inf')
        self.bb_upper = None
        self.bb_lower = None
        self.bb_middle = None
        self.current_bb_width = None
        
        # Historical BB width tracking for lowest calculation
        self.bb_width_history = []
        self.max_history_length = 50  # Keep last 50 candles for lowest calculation
    
    async def check_entry_conditions(self, 
                               candle: Dict[str, Any], 
                               mr_values: Dict[str, Any]) -> Optional[Signal]:
        """
        Check for BB Width entry conditions.
        
        Args:
            candle: The current candle data
            mr_values: Morning range values (not used in this strategy)
            
        Returns:
            Signal if entry conditions are met, None otherwise
        """
        candle_info = self._format_candle_info(candle)
        
        # Convert timestamp if needed
        timestamp = candle.get('timestamp')
        if isinstance(timestamp, str):
            timestamp = pd.to_datetime(timestamp)
            
        # Validate trading hours
        candle_time = timestamp.time()
        if not (self.market_open <= candle_time <= self.market_close):
            logger.debug(f"{candle_info}Outside trading hours")
            return None
            
        # Validate required BB data
        if not self._validate_bb_data(candle):
            logger.debug(f"{candle_info}Missing or invalid BB data")
            return None
            
        # Extract BB values
        self.bb_upper = candle.get('bb_upper_x', 0)
        self.bb_lower = candle.get('bb_lower_x', 0)
        self.bb_middle = candle.get('bb_middle_x', 0)
        self.current_bb_width = self.bb_upper - self.bb_lower
        self.current_bb_width = round(self.current_bb_width, 2)
        
        # Update BB width history
        # self._update_bb_width_history(self.current_bb_width)
        
        # Get lowest BB width from CSV file
        self.lowest_bb_width = self._get_lowest_bb_width_from_csv()
        
        
        # Check for squeeze condition
        squeeze_threshold = self.lowest_bb_width * (1 + self.bb_width_threshold)  # ±0.1% of lowest
        squeeze_threshold = round(squeeze_threshold, 2)
        
        if self.current_bb_width <= squeeze_threshold:
            # Squeeze condition detected
            if not self.squeeze_detected:
                # Start new squeeze
                self.squeeze_detected = True
                self.squeeze_start_time = timestamp
                self.squeeze_candle_count = 1
                logger.debug(f"{candle_info}BB squeeze detected - Width: {self.current_bb_width:.6f}, Threshold: {squeeze_threshold:.6f}")
            else:
                # Continue existing squeeze
                self.squeeze_candle_count += 1
                logger.debug(f"{candle_info}BB squeeze continuing - Candle {self.squeeze_candle_count}")
        else:
            # No squeeze condition
            if self.squeeze_detected:
                logger.debug(f"{candle_info}BB squeeze ended - Width: {self.current_bb_width:.6f}")
            self.squeeze_detected = False
            self.squeeze_candle_count = 0
        
        # Check entry conditions only if squeeze duration is within range
        if (self.squeeze_detected and 
            self.squeeze_candle_count >= self.squeeze_duration_min and 
            self.squeeze_candle_count <= self.squeeze_duration_max):
            
            # Check for entry conditions
            entry_signal = self._check_entry_conditions(candle, timestamp, candle_info)
            if entry_signal:
                return entry_signal
        
        return None
    
    def _validate_bb_data(self, candle: Dict[str, Any]) -> bool:
        """Validate that required BB data is present and valid."""
        required_fields = ['bb_upper_x', 'bb_lower_x', 'bb_middle_x']
        
        for field in required_fields:
            if field not in candle:
                logger.warning(f"Missing required BB field: {field}")
                return False
            
            value = candle[field]
            if value is None or pd.isna(value) or value <= 0:
                logger.warning(f"Invalid BB field {field}: {value}")
                return False
        
        # Validate BB relationships
        bb_upper = candle['bb_upper_x']
        bb_lower = candle['bb_lower_x']
        bb_middle = candle['bb_middle_x']

        if bb_upper == bb_lower == bb_middle:
            logger.warning(f"No buy sell on candle {candle['timestamp']}")
            return True
        
        if bb_upper < bb_lower:
            logger.warning(f"Invalid BB relationship: upper ({bb_upper}) < lower ({bb_lower})")
            return False
        
        if not (bb_lower < bb_middle < bb_upper):
            logger.warning(f"Invalid BB relationship: middle ({bb_middle}) not between upper ({bb_upper}) and lower ({bb_lower})")
            return False
        
        return True
    
    def _get_lowest_bb_width_from_csv(self) -> float:
        """
        Get the lowest BB width from the CSV analysis file.
        
        Returns:
            float: The lowest BB width value for the current instrument
        """
        try:
            csv_file_path = "/Users/gaurav/setbull_projects/setbull_trader/python_strategies/output/bb_width_analysis.csv"
            
            # Read the CSV file
            df = pd.read_csv(csv_file_path)
            
            # Get the instrument key from config
            instrument_key = self.config.instrument_key.get("instrument_key")
            if not instrument_key:
                logger.warning("No instrument_key found in config, using default lowest BB width")
                return 0.001  # Default fallback value
            
            # Filter the dataframe for the current instrument
            instrument_data = df[df['instrument_key'] == instrument_key]
            
            if instrument_data.empty:
                logger.warning(f"No data found for instrument_key: {instrument_key}, using default lowest BB width")
                return 0.001  # Default fallback value
            
            # Get the lowest BB width (using lowest_mean_bb_width column)
            lowest_bb_width = instrument_data.iloc[0]['lowest_mean_bb_width']
            lowest_bb_width = lowest_bb_width/2
            
            # Convert to float and validate
            if pd.isna(lowest_bb_width) or lowest_bb_width <= 0:
                logger.warning(f"Invalid lowest BB width value: {lowest_bb_width}, using default")
                return 0.001  # Default fallback value
            
            logger.debug(f"Retrieved lowest BB width for {instrument_key}: {lowest_bb_width}")
            return float(lowest_bb_width)
            
        except FileNotFoundError:
            logger.error(f"BB width analysis CSV file not found: {csv_file_path}")
            return 0.001  # Default fallback value
        except Exception as e:
            logger.error(f"Error reading BB width analysis CSV: {e}")
            return 0.001  # Default fallback value
    
    def _update_bb_width_history(self, bb_width: float) -> None:
        """Update BB width history for lowest calculation."""
        if bb_width > 0:
            self.bb_width_history.append(bb_width)
            
            # Keep only the last max_history_length values
            if len(self.bb_width_history) > self.max_history_length:
                self.bb_width_history = self.bb_width_history[-self.max_history_length:]
    
    def _check_entry_conditions(self, candle: Dict[str, Any], timestamp: datetime, candle_info: str) -> Optional[Signal]:
        """Check for actual entry conditions during squeeze."""
        current_price = candle.get('close', 0)
        direction = self.config.instrument_key.get("direction")
        
        # Check if already in a trade
        if self.in_long_trade or self.in_short_trade:
            return None
        
        # Check for long entry (price above BB upper band)
        if (current_price > self.bb_upper and 
            direction == "BULLISH" and
            self.can_generate_signal(SignalType.BB_WIDTH_ENTRY.value, "LONG")):
            
            self.in_long_trade = True
            logger.info(f"{candle_info}BB Width long entry detected - Price: {current_price:.2f}, BB Upper: {self.bb_upper:.2f}")
            
            signal = Signal(
                type=SignalType.BB_WIDTH_ENTRY,
                direction=SignalDirection.LONG,
                timestamp=timestamp,
                price=self.bb_upper,  # Entry at BB upper band
                range_values={
                    'bb_upper': self.bb_upper,
                    'bb_lower': self.bb_lower,
                    'bb_middle': self.bb_middle,
                    'bb_width': self.current_bb_width,
                    'lowest_bb_width': self.lowest_bb_width,
                    'squeeze_duration': self.squeeze_candle_count,
                    'squeeze_start_time': self.squeeze_start_time.isoformat() if self.squeeze_start_time else None
                },
                mr_values={},
                metadata={
                    'entry_type': 'bb_width_entry',
                    'entry_time': timestamp.strftime('%H:%M'),
                    'strategy': 'BB_WIDTH_ENTRY',
                    'squeeze_detected': True,
                    'squeeze_candle_count': self.squeeze_candle_count
                }
            )
            
            self.update_signal_state(SignalType.BB_WIDTH_ENTRY.value, "LONG")
            return signal
        
        # Check for short entry (price below BB lower band)
        elif (current_price < self.bb_lower and 
              direction == "BEARISH" and
              self.can_generate_signal(SignalType.BB_WIDTH_ENTRY.value, "SHORT")):
            
            self.in_short_trade = True
            logger.info(f"{candle_info}BB Width short entry detected - Price: {current_price:.2f}, BB Lower: {self.bb_lower:.2f}")
            
            signal = Signal(
                type=SignalType.BB_WIDTH_ENTRY,
                direction=SignalDirection.SHORT,
                timestamp=timestamp,
                price=self.bb_lower,  # Entry at BB lower band
                range_values={
                    'bb_upper': self.bb_upper,
                    'bb_lower': self.bb_lower,
                    'bb_middle': self.bb_middle,
                    'bb_width': self.current_bb_width,
                    'lowest_bb_width': self.lowest_bb_width,
                    'squeeze_duration': self.squeeze_candle_count,
                    'squeeze_start_time': self.squeeze_start_time.isoformat() if self.squeeze_start_time else None
                },
                mr_values={},
                metadata={
                    'entry_type': 'bb_width_entry',
                    'entry_time': timestamp.strftime('%H:%M'),
                    'strategy': 'BB_WIDTH_ENTRY',
                    'squeeze_detected': True,
                    'squeeze_candle_count': self.squeeze_candle_count
                }
            )
            
            self.update_signal_state(SignalType.BB_WIDTH_ENTRY.value, "SHORT")
            return signal
        
        return None
    
    def _format_candle_info(self, candle: Dict[str, Any]) -> str:
        """Format candle information for logging."""
        if not candle:
            return ""
            
        time_str = candle.get('timestamp', 'unknown')
        if isinstance(time_str, pd.Timestamp):
            time_str = time_str.strftime('%Y-%m-%d %H:%M:%S')
            
        open_price = candle.get('open', 0)
        high_price = candle.get('high', 0)
        low_price = candle.get('low', 0)
        close_price = candle.get('close', 0)
        
        return f"[{time_str}] [O:{open_price:.2f} H:{high_price:.2f} L:{low_price:.2f} C:{close_price:.2f}] - "
    
    def reset_state(self) -> None:
        """Reset the entry strategy state."""
        self.in_long_trade = False
        self.in_short_trade = False
        self.squeeze_detected = False
        self.squeeze_start_time = None
        self.squeeze_candle_count = 0
        self.lowest_bb_width = float('inf')
        self.bb_upper = None
        self.bb_lower = None
        self.bb_middle = None
        self.current_bb_width = None
        self.bb_width_history = []
        
        # Reset base class state
        self.can_generate_long = True
        self.can_generate_short = True
        self.state['can_generate_long'] = True
        self.state['can_generate_short'] = True 