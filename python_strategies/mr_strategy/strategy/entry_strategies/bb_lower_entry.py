"""
BB Lower Entry Strategy Implementation

This module implements the BB Lower Entry Strategy (BB_LOWER_ENTRY) which looks for
trading opportunities when price pulls back to the Bollinger Bands lower band after
establishing a strong trend.
"""

from typing import Dict, Optional, Any
from datetime import datetime, time
import logging
import pandas as pd

from .base import EntryStrategy
from ..models import Signal, SignalType, SignalDirection

logger = logging.getLogger(__name__)

class BBLowerEntryStrategy(EntryStrategy):
    """Implementation of the BB Lower Entry Strategy (BB_LOWER_ENTRY)."""
    
    def __init__(self, config):
        super().__init__(config)
        
        # BB Lower strategy specific parameters
        self.confirmation_time = getattr(self.config, 'bb_lower_confirmation_time', time(12, 0))  # 12:00 PM
        self.stop_loss_percentage = getattr(self.config, 'bb_lower_stop_loss_percentage', 0.002)  # 0.2% stop loss
        self.market_open = time(9, 15)        # Market open
        self.market_close = time(15, 30)      # Market close
        
        # Strategy state variables
        self.in_long_trade = False            # Track long position
        self.in_short_trade = False           # Track short position
        self.observing_confirmation = True    # Track confirmation phase
        self.confirmation_complete = False    # Track if confirmation is done
        
        # Day level tracking
        self.day_high = None                  # Day's highest high
        self.day_low = None                   # Day's lowest low
        self.middle_line = None               # Middle line (day_high + day_low) / 2
        self.day_high_time = None             # Time when day high was established
        self.day_low_time = None              # Time when day low was established
        
        # BB values
        self.bb_upper = None
        self.bb_lower = None
        self.bb_middle = None
        self.current_bb_width = None
        
        # Trade management
        self.entry_price = None
        self.stop_loss_price = None
        self.target_price = None
        
        # Pullback tracking
        self.pullback_above_middle = False    # For bullish trend
        self.pullback_below_middle = False    # For bearish trend
        self.trend_strength_confirmed = False # Trend strength confirmation
        
        # Enhanced confirmation state tracking
        self.confirmation_attempted = False   # Track if confirmation was attempted
        self.confirmation_failed = False      # Track if confirmation failed
        self.price_position_at_confirmation = None  # Track price position at confirmation
        self.day_range_at_confirmation = None       # Track day range at confirmation
    
    async def check_entry_conditions(self, 
                               candle: Dict[str, Any], 
                               mr_values: Dict[str, Any]) -> Optional[Signal]:
        """
        Check for BB Lower entry conditions.
        
        Args:
            candle: The current candle data
            mr_values: Morning range values (not used in this strategy)
            
        Returns:
            Signal if entry conditions are met, None otherwise
        """
        try:
            # Format candle info for logging
            candle_info = self._format_candle_info(candle)
            
            # Convert timestamp if needed
            timestamp = candle.get('timestamp')
            if isinstance(timestamp, str):
                timestamp = pd.to_datetime(timestamp)
            
            # Enhanced timestamp validation
            if timestamp is None:
                logger.warning(f"{candle_info}Invalid timestamp")
                return None
            
            # Check trading hours
            candle_time = timestamp.time()
            if not (self.market_open <= candle_time <= self.market_close):
                logger.debug(f"{candle_info}Outside trading hours")
                return None
            
            # Validate BB data
            if not self._validate_bb_data(candle):
                logger.debug(f"{candle_info}Missing or invalid BB data")
                return None
            
            # Extract BB values
            self.bb_upper = candle.get('bb_upper', 0)
            self.bb_lower = candle.get('bb_lower', 0)
            self.bb_middle = candle.get('bb_middle', 0)
            self.current_bb_width = candle.get('bb_width', 0)
            
            # Update day levels
            self._update_day_levels(candle)
            
            # Check confirmation phase
            if self.observing_confirmation:
                self._check_confirmation_conditions(candle)
                return None  # No signals during confirmation phase
            
            # Check entry conditions after confirmation
            if self.confirmation_complete:
                entry_signal = self._check_entry_conditions(candle, timestamp, candle_info)
                if entry_signal:
                    return entry_signal
            
            return None
            
        except Exception as e:
            logger.error(f"Error in check_entry_conditions: {str(e)}")
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
    
    def _validate_bb_data(self, candle: Dict[str, Any]) -> bool:
        """Validate that required BB data is present and valid."""
        required_fields = ['bb_upper', 'bb_lower', 'bb_middle', 'bb_width']
        
        # Check for required fields
        for field in required_fields:
            if field not in candle:
                logger.warning(f"Missing required BB field: {field}")
                return False
            
            value = candle[field]
            if value is None or pd.isna(value) or value <= 0:
                logger.warning(f"Invalid BB field {field}: {value}")
                return False
        
        # Validate BB relationships
        bb_upper = candle['bb_upper']
        bb_lower = candle['bb_lower']
        bb_middle = candle['bb_middle']
        
        if bb_upper <= bb_lower:
            logger.warning(f"Invalid BB relationship: upper ({bb_upper}) <= lower ({bb_lower})")
            return False
        
        if not (bb_lower <= bb_middle <= bb_upper):
            logger.warning(f"Invalid BB relationship: middle ({bb_middle}) not between upper ({bb_upper}) and lower ({bb_lower})")
            return False
        
        return True
    
    def _update_day_levels(self, candle: Dict[str, Any]) -> None:
        """Update day high, day low, and middle line."""
        current_high = candle.get('high', 0)
        current_low = candle.get('low', 0)
        timestamp = candle.get('timestamp')
        
        # Update day high
        if self.day_high is None or current_high > self.day_high:
            self.day_high = current_high
            self.day_high_time = timestamp
            logger.debug(f"New day high established: {self.day_high}")
        
        # Update day low
        if self.day_low is None or current_low < self.day_low:
            self.day_low = current_low
            self.day_low_time = timestamp
            logger.debug(f"New day low established: {self.day_low}")
        
        # Calculate middle line
        if self.day_high is not None and self.day_low is not None:
            self.middle_line = (self.day_high + self.day_low) / 2
            logger.debug(f"Middle line calculated: {self.middle_line}")
    
    def _check_confirmation_conditions(self, candle: Dict[str, Any]) -> None:
        """Check confirmation conditions at 12:00 PM."""
        timestamp = candle.get('timestamp')
        if isinstance(timestamp, str):
            timestamp = pd.to_datetime(timestamp)
        
        candle_time = timestamp.time()
        current_price = candle.get('close', 0)
        direction = self.config.instrument_key.get("direction")
        
        # Check if it's 12:00 PM
        if candle_time == self.confirmation_time:
            logger.info(f"12:00 PM confirmation check - Day High: {self.day_high}, Day Low: {self.day_low}, Middle Line: {self.middle_line}")
            
            # Mark confirmation as attempted
            self.confirmation_attempted = True
            
            # Validate day levels are established
            if self.day_high is None or self.day_low is None or self.middle_line is None:
                logger.warning("Day levels not established by 12:00 PM, skipping confirmation")
                self.confirmation_failed = True
                self.observing_confirmation = False
                self.confirmation_complete = True
                return
            
            # Calculate day range for trend strength validation
            day_range = self.day_high - self.day_low
            if day_range <= 0:
                logger.warning("Invalid day range (day_high <= day_low), skipping confirmation")
                self.confirmation_failed = True
                self.observing_confirmation = False
                self.confirmation_complete = True
                return
            
            # Store day range for later reference
            self.day_range_at_confirmation = day_range
            
            # Check trend strength based on direction
            if direction == "BULLISH":
                # For bullish trend, check if price stays above middle line after pullback
                if current_price > self.middle_line:
                    # Additional validation: price should be in upper half of day range
                    price_position = (current_price - self.day_low) / day_range
                    self.price_position_at_confirmation = price_position
                    
                    if price_position >= 0.5:  # Price in upper 50% of day range
                        self.pullback_above_middle = True
                        self.trend_strength_confirmed = True
                        logger.info(f"Bullish trend strength confirmed - Price: {current_price}, Middle Line: {self.middle_line}, Position: {price_position:.2%}")
                    else:
                        logger.warning(f"Bullish trend strength insufficient - Price: {current_price}, Position: {price_position:.2%}")
                        self.confirmation_failed = True
                else:
                    # Calculate price position even when below middle line for logging
                    price_position = (current_price - self.day_low) / day_range
                    self.price_position_at_confirmation = price_position
                    logger.warning(f"Bullish trend strength not confirmed - Price: {current_price}, Middle Line: {self.middle_line}, Position: {price_position:.2%}")
                    self.confirmation_failed = True
            
            elif direction == "BEARISH":
                # For bearish trend, check if price stays below middle line after pullback
                if current_price < self.middle_line:
                    # Additional validation: price should be in lower half of day range
                    price_position = (self.day_high - current_price) / day_range
                    self.price_position_at_confirmation = price_position
                    
                    if price_position >= 0.5:  # Price in lower 50% of day range
                        self.pullback_below_middle = True
                        self.trend_strength_confirmed = True
                        logger.info(f"Bearish trend strength confirmed - Price: {current_price}, Middle Line: {self.middle_line}, Position: {price_position:.2%}")
                    else:
                        logger.warning(f"Bearish trend strength insufficient - Price: {current_price}, Position: {price_position:.2%}")
                        self.confirmation_failed = True
                else:
                    # Calculate price position even when above middle line for logging
                    price_position = (self.day_high - current_price) / day_range
                    self.price_position_at_confirmation = price_position
                    logger.warning(f"Bearish trend strength not confirmed - Price: {current_price}, Middle Line: {self.middle_line}, Position: {price_position:.2%}")
                    self.confirmation_failed = True
            
            # Mark confirmation complete
            self.observing_confirmation = False
            self.confirmation_complete = True
            
            if self.trend_strength_confirmed:
                logger.info("12:00 PM confirmation complete - Now looking for BB lower entries")
            else:
                logger.warning("12:00 PM confirmation failed - No entries will be generated")
    
    def _check_entry_conditions(self, candle: Dict[str, Any], timestamp: datetime, candle_info: str) -> Optional[Signal]:
        """Check for actual entry conditions after confirmation."""
        current_price = candle.get('close', 0)
        current_low = candle.get('low', 0)
        current_high = candle.get('high', 0)
        direction = self.config.instrument_key.get("direction")
        
        # Enhanced validation checks
        if not self._validate_entry_prerequisites(candle_info):
            return None
        
        # Check if already in a trade
        if self.in_long_trade or self.in_short_trade:
            logger.debug(f"{candle_info}Already in a trade, skipping entry check")
            return None
        
        # Check if trend strength is confirmed
        if not self.trend_strength_confirmed:
            logger.debug(f"{candle_info}Trend strength not confirmed, skipping entry check")
            return None
        
        # Calculate entry parameters
        if direction == "BULLISH":
            # Long entry: candle low <= BB lower band
            if current_low <= self.bb_lower and self.can_generate_signal(SignalType.BB_LOWER_ENTRY.value, "LONG"):
                # Enhanced entry validation for bullish trend
                if self._validate_bullish_entry(candle, candle_info):
                    self.in_long_trade = True
                    self.entry_price = self.bb_lower
                    self.stop_loss_price = self.entry_price * (1 - self.stop_loss_percentage)
                    self.target_price = self._calculate_target_price("LONG")
                    
                    target_str = f"{self.target_price:.2f}" if self.target_price else "None"
                    logger.info(f"{candle_info}BB Lower long entry detected - Entry: {self.entry_price:.2f}, SL: {self.stop_loss_price:.2f}, Target: {target_str}")
                    
                    return self._create_signal(SignalDirection.LONG, timestamp, candle_info)
        
        elif direction == "BEARISH":
            # Short entry: candle high >= BB upper band (for bearish trend)
            if current_high >= self.bb_upper and self.can_generate_signal(SignalType.BB_LOWER_ENTRY.value, "SHORT"):
                # Enhanced entry validation for bearish trend
                if self._validate_bearish_entry(candle, candle_info):
                    self.in_short_trade = True
                    self.entry_price = self.bb_upper
                    self.stop_loss_price = self.entry_price * (1 + self.stop_loss_percentage)
                    self.target_price = self._calculate_target_price("SHORT")
                    
                    target_str = f"{self.target_price:.2f}" if self.target_price else "None"
                    logger.info(f"{candle_info}BB Upper short entry detected - Entry: {self.entry_price:.2f}, SL: {self.stop_loss_price:.2f}, Target: {target_str}")
                    
                    return self._create_signal(SignalDirection.SHORT, timestamp, candle_info)
        
        return None
    
    def _validate_entry_prerequisites(self, candle_info: str) -> bool:
        """Validate prerequisites for entry conditions."""
        # Check if BB data is available
        if self.bb_upper is None or self.bb_lower is None or self.bb_middle is None:
            logger.warning(f"{candle_info}BB data not available for entry validation")
            return False
        
        # Check if day levels are available
        if self.day_high is None or self.day_low is None or self.middle_line is None:
            logger.warning(f"{candle_info}Day levels not available for entry validation")
            return False
        
        # Check if confirmation was attempted
        if not self.confirmation_attempted:
            logger.warning(f"{candle_info}Confirmation not attempted yet")
            return False
        
        return True
    
    def _validate_bullish_entry(self, candle: Dict[str, Any], candle_info: str) -> bool:
        """Validate bullish entry conditions."""
        current_price = candle.get('close', 0)
        current_low = candle.get('low', 0)
        
        # Additional validation for bullish trend
        # Check if price is still above middle line (trend strength maintained)
        if current_price <= self.middle_line:
            logger.debug(f"{candle_info}Price below middle line, bullish trend strength not maintained")
            return False
        
        # Check if BB lower band is reasonable (not too far from current price)
        price_to_bb_lower_ratio = abs(current_price - self.bb_lower) / current_price
        if price_to_bb_lower_ratio > 0.05:  # More than 5% away from BB lower
            logger.debug(f"{candle_info}BB lower band too far from current price: {price_to_bb_lower_ratio:.2%}")
            return False
        
        # Check if entry makes sense (current low should be close to BB lower)
        if current_low > self.bb_lower * 1.01:  # More than 1% above BB lower
            logger.debug(f"{candle_info}Current low too far above BB lower band")
            return False
        
        return True
    
    def _validate_bearish_entry(self, candle: Dict[str, Any], candle_info: str) -> bool:
        """Validate bearish entry conditions."""
        current_price = candle.get('close', 0)
        current_high = candle.get('high', 0)
        
        # Additional validation for bearish trend
        # Check if price is still below middle line (trend strength maintained)
        if current_price >= self.middle_line:
            logger.debug(f"{candle_info}Price above middle line, bearish trend strength not maintained")
            return False
        
        # Check if BB upper band is reasonable (not too far from current price)
        price_to_bb_upper_ratio = abs(current_price - self.bb_upper) / current_price
        if price_to_bb_upper_ratio > 0.05:  # More than 5% away from BB upper
            logger.debug(f"{candle_info}BB upper band too far from current price: {price_to_bb_upper_ratio:.2%}")
            return False
        
        # Check if entry makes sense (current high should be close to BB upper)
        if current_high < self.bb_upper * 0.99:  # More than 1% below BB upper
            logger.debug(f"{candle_info}Current high too far below BB upper band")
            return False
        
        return True
    
    def _calculate_target_price(self, direction: str) -> Optional[float]:
        """Calculate target price based on direction and configuration."""
        if hasattr(self.config, 'target_price') and self.config.target_price is not None:
            return self.config.target_price
        
        # Calculate target based on day range if no config target
        if self.day_high is not None and self.day_low is not None:
            day_range = self.day_high - self.day_low
            target_multiplier = getattr(self.config, 'bb_lower_target_multiplier', 1.5)
            
            if direction == "LONG":
                # Target at configurable multiplier x day range above entry
                return self.entry_price + (day_range * target_multiplier)
            elif direction == "SHORT":
                # Target at configurable multiplier x day range below entry
                return self.entry_price - (day_range * target_multiplier)
        
        return None
    
    def _create_signal(self, direction: SignalDirection, timestamp: datetime, candle_info: str) -> Signal:
        """Create signal object with comprehensive metadata."""
        # Update signal state to prevent duplicates
        self.update_signal_state(SignalType.BB_LOWER_ENTRY.value, direction.value)
        
        # Calculate additional metrics
        risk_amount = abs(self.entry_price - self.stop_loss_price)
        reward_amount = abs(self.target_price - self.entry_price) if self.target_price else 0
        risk_reward_ratio = reward_amount / risk_amount if risk_amount > 0 else 0
        
        # Calculate day range metrics
        day_range = self.day_high - self.day_low if self.day_high and self.day_low else 0
        entry_position_in_day = (self.entry_price - self.day_low) / day_range if day_range > 0 else 0
        
        return Signal(
            type=SignalType.BB_LOWER_ENTRY,
            direction=direction,
            timestamp=timestamp,
            price=self.entry_price,
            range_values={
                'bb_upper': self.bb_upper,
                'bb_lower': self.bb_lower,
                'bb_middle': self.bb_middle,
                'bb_width': self.current_bb_width,
                'day_high': self.day_high,
                'day_low': self.day_low,
                'middle_line': self.middle_line,
                'entry_price': self.entry_price,
                'stop_loss_price': self.stop_loss_price,
                'target_price': self.target_price,
                'trend_strength_confirmed': self.trend_strength_confirmed,
                'risk_amount': risk_amount,
                'reward_amount': reward_amount,
                'risk_reward_ratio': risk_reward_ratio,
                'day_range': day_range,
                'entry_position_in_day': entry_position_in_day
            },
            mr_values={},  # Not used in this strategy
            metadata={
                'entry_type': 'bb_lower_entry',
                'entry_time': timestamp.strftime('%H:%M'),
                'strategy': 'BB_LOWER_ENTRY',
                'confirmation_time': self.confirmation_time.strftime('%H:%M'),
                'day_high_time': self.day_high_time.isoformat() if self.day_high_time else None,
                'day_low_time': self.day_low_time.isoformat() if self.day_low_time else None,
                'trend_strength_confirmed': self.trend_strength_confirmed,
                'confirmation_attempted': self.confirmation_attempted,
                'confirmation_failed': self.confirmation_failed,
                'price_position_at_confirmation': self.price_position_at_confirmation,
                'day_range_at_confirmation': self.day_range_at_confirmation,
                'risk_amount': risk_amount,
                'reward_amount': reward_amount,
                'risk_reward_ratio': risk_reward_ratio,
                'stop_loss_percentage': self.stop_loss_percentage,
                'entry_position_in_day': entry_position_in_day,
                'bb_width_at_entry': self.current_bb_width,
                'trade_type': 'BB_LOWER_ENTRY'
            }
        )
    
    def reset_state(self) -> None:
        """Reset the entry strategy state."""
        self.in_long_trade = False
        self.in_short_trade = False
        self.observing_confirmation = True
        self.confirmation_complete = False
        self.day_high = None
        self.day_low = None
        self.middle_line = None
        self.day_high_time = None
        self.day_low_time = None
        self.bb_upper = None
        self.bb_lower = None
        self.bb_middle = None
        self.current_bb_width = None
        self.entry_price = None
        self.stop_loss_price = None
        self.target_price = None
        self.pullback_above_middle = False
        self.pullback_below_middle = False
        self.trend_strength_confirmed = False
        
        # Enhanced confirmation state tracking
        self.confirmation_attempted = False
        self.confirmation_failed = False
        self.price_position_at_confirmation = None
        self.day_range_at_confirmation = None
        
        # Reset base class state
        self.can_generate_long = True
        self.can_generate_short = True
        self.state['can_generate_long'] = True
        self.state['can_generate_short'] = True 