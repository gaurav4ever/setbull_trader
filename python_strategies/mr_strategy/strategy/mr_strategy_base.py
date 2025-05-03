"""
Base Morning Range Strategy Implementation.

This module implements the core Morning Range strategy logic that is
common across different variants.
"""

from typing import Dict, Optional, Union, List, Tuple
import pandas as pd
import numpy as np
from datetime import datetime, time
import logging

from .base_strategy import BaseStrategy, StrategyState, StrategyConfig
from ..data.data_processor import CandleProcessor
from .position_manager import PositionManager
from .trade_manager import TradeManager, TradeType
from .risk_calculator import RiskCalculator

logger = logging.getLogger(__name__)

class MorningRangeStrategy(BaseStrategy):
    """Base implementation of Morning Range strategy."""
    
    def __init__(self, 
                 config: StrategyConfig,
                 position_manager: PositionManager,
                 trade_manager: TradeManager,
                 risk_calculator: RiskCalculator):
        """Initialize Range strategy."""
        super().__init__(config)
        self.position_manager = position_manager
        self.trade_manager = trade_manager
        self.risk_calculator = risk_calculator
        self.candle_processor = CandleProcessor(config={
            'instrument_key': config.instrument_key
        })
        
        logger.info(f"Initialized Range Strategy: {config.range_type} - {config.entry_type}")

    async def calculate_morning_range(self, candles: pd.DataFrame) -> Dict:
        """
        Calculate morning range values.
        
        Args:
            candles: DataFrame containing candle data
            
        Returns:
            Dictionary containing morning range values and validation status
        """
        if candles.empty:
            logger.warning("Empty candle data provided")
            return {
                'is_valid': False,
                'validation_reason': 'Empty candle data',
                'mr_high': None,
                'mr_low': None,
                'validation_details': {}
            }
            
        try:
            # Calculate morning range using the new async method
            range_values = self._calculate_morning_range(candles)
            
            if not range_values:
                logger.warning("Failed to calculate morning range")
                self.update_state(StrategyState.ERROR)
                return {
                    'is_valid': False,
                    'validation_reason': 'Failed to calculate MR',
                    'mr_high': None,
                    'mr_low': None,
                    'validation_details': {}
                }
                
            # Store MR values and update state
            self.morning_range = {
                'high': range_values['mr_high'],
                'low': range_values['mr_low']
            }
            
            # Add validation details
            range_values['validation_details'] = {
                'range_size': range_values['mr_high'] - range_values['mr_low'],
                'range_type': self.config.range_type,
                'timestamp': datetime.now().isoformat()
            }
            
            # Ensure all required fields are present
            range_values.update({
                'is_valid': range_values['is_valid'],
                'validation_reason': range_values['error']
            })
            
            self.update_state(StrategyState.RANGE_CALCULATED)
            logger.info(f"Calculated morning range: {range_values} for the day {candles['timestamp'].iloc[0].date()}")
            return range_values    
        except Exception as e:
            logger.error(f"Error calculating morning range: {str(e)}")
            self.update_state(StrategyState.ERROR)
            return {
                'is_valid': False,
                'validation_reason': f'Error calculating MR: {str(e)}',
                'mr_high': None,
                'mr_low': None,
                'validation_details': {}
            }

    def calculate_entry_levels(self) -> Dict:
        """Calculate entry and exit levels."""
        if not self.morning_range:
            logger.warning("Morning range not calculated")
            return {}
            
        mr_high = self.morning_range['high']
        mr_low = self.morning_range['low']
        
        # Calculate buffer
        buffer = self.config.buffer_ticks * self.config.tick_size
        
        # Calculate entry levels
        long_entry = mr_high + buffer
        short_entry = mr_low - buffer
        
        # Calculate stop loss levels
        long_sl = long_entry * (1 - self.config.sl_percentage/100)
        short_sl = short_entry * (1 + self.config.sl_percentage/100)
        
        # Calculate target levels based on R-multiple
        long_risk = long_entry - long_sl
        short_risk = short_sl - short_entry
        
        long_target = long_entry + (long_risk * self.config.target_r)
        short_target = short_entry - (short_risk * self.config.target_r)
        
        levels = {
            'long_entry': round(long_entry, 2),
            'short_entry': round(short_entry, 2),
            'long_sl': round(long_sl, 2),
            'short_sl': round(short_sl, 2),
            'long_target': round(long_target, 2),
            'short_target': round(short_target, 2),
            'long_risk': round(long_risk, 2),
            'short_risk': round(short_risk, 2)
        }
        
        self.entry_levels = levels
        logger.info(f"Calculated entry levels: {levels}")
        return levels

    def validate_setup(self, candles: pd.DataFrame) -> bool:
        """Validate strategy setup conditions."""
        if not self.morning_range or not self.entry_levels:
            logger.warning("Missing morning range or entry levels")
            return False
            
        # Validate range size
        range_size = self.morning_range['high'] - self.morning_range['low']
        min_range = 10 * self.config.tick_size
        max_range = 100 * self.config.tick_size
        
        if not min_range <= range_size <= max_range:
            logger.warning(f"Invalid range size: {range_size}")
            return False
        
        # Validate risk-reward
        long_rr = abs(self.entry_levels['long_target'] - self.entry_levels['long_entry']) / \
                  abs(self.entry_levels['long_entry'] - self.entry_levels['long_sl'])
                  
        short_rr = abs(self.entry_levels['short_entry'] - self.entry_levels['short_target']) / \
                   abs(self.entry_levels['short_sl'] - self.entry_levels['short_entry'])
        
        min_rr = 1.5
        if long_rr < min_rr and short_rr < min_rr:
            logger.warning(f"Insufficient risk-reward: long={long_rr}, short={short_rr}")
            return False
        
        self.update_state(StrategyState.SETUP_CONFIRMED)
        return True

    def check_entry_conditions(self, candle: Dict) -> Tuple[bool, str]:
        """Check entry conditions based on the current candle data."""
        
        current_price = candle['close']
        
        # Check for long entry condition
        if current_price >= self.entry_levels['long_entry']:
            return True, "LONG"
        
        # Check for short entry condition
        if current_price <= self.entry_levels['short_entry']:
            return True, "SHORT"
        
        # If no conditions are met, return False
        return False, "No entry conditions met"

    def check_exit_conditions(self, candle: Dict) -> Tuple[bool, str]:
        """Check exit conditions."""
        if not self.position:
            return False, "No active position"
            
        position_type = self.position['position_type']
        current_price = candle['close']
        
        # Check stop loss
        if position_type == "LONG":
            if current_price <= self.position['stop_loss']:
                return True, "Stop loss hit"
        else:  # SHORT
            if current_price >= self.position['stop_loss']:
                return True, "Stop loss hit"
        
        # Check target
        if position_type == "LONG":
            if current_price >= self.position['take_profit']:
                return True, "Target reached"
        else:  # SHORT
            if current_price <= self.position['take_profit']:
                return True, "Target reached"
        
        return False, "Exit conditions not met"

    def process_candle(self, candle: Dict) -> Dict:
        """Process new candle data."""
        if self.state == StrategyState.WAITING_FOR_RANGE:
            return {'action': 'waiting_for_range'}
            
        if self.state == StrategyState.RANGE_CALCULATED:
            if not self.validate_setup(pd.DataFrame([candle])):
                return {'action': 'invalid_setup'}
        
        if self.state == StrategyState.SETUP_CONFIRMED:
            should_enter, entry_type = self.check_entry_conditions(candle)
            if should_enter:
                entry_result = self.execute_entry(candle, entry_type)
                return {'action': 'entry', 'result': entry_result}
        
        if self.state == StrategyState.IN_POSITION:
            should_exit, exit_reason = self.check_exit_conditions(candle)
            if should_exit:
                exit_result = self.execute_exit(candle, exit_reason)
                return {'action': 'exit', 'result': exit_result}
        
        return {'action': 'no_action'}

    def execute_entry(self, candle: Dict, entry_type: str) -> Dict:
        """Execute entry order."""
        current_price = candle['close']
        position_type = "LONG" if current_price >= self.entry_levels['long_entry'] else "SHORT"
        
        # Calculate position size
        risk_amount = self.entry_levels[f'{position_type.lower()}_risk']
        position_size = self.position_manager.calculate_position_size(
            current_price,
            self.config.sl_percentage,
            position_type
        )
        
        # Validate position risk
        position_risk = self.risk_calculator.calculate_position_risk(
            position_size,
            current_price,
            self.entry_levels[f'{position_type.lower()}_sl'],
            self.position_manager.account_info.total_capital
        )
        
        is_valid, reason = self.risk_calculator.validate_position_risk(
            position_risk,
            self.config.instrument_key,
            datetime.now()
        )
        
        if not is_valid:
            logger.warning(f"Position risk validation failed: {reason}")
            return {'status': 'rejected', 'reason': reason}
        
        # Create trade
        trade_type = TradeType.IMMEDIATE_BREAKOUT if entry_type == "LONG" else TradeType.RETEST_ENTRY
        
        trade = self.trade_manager.create_trade(
            instrument_key=self.config.instrument_key,
            entry_price=current_price,
            position_size=position_size,
            position_type=position_type,
            trade_type=trade_type,
            sl_percentage=self.config.sl_percentage
        )
        
        self.position = trade
        self.update_state(StrategyState.IN_POSITION)
        
        logger.info(f"Executed entry: {trade}")
        return {'status': 'success', 'trade': trade}

    def execute_exit(self, candle: Dict, reason: str) -> Dict:
        """Execute exit order."""
        if not self.position:
            return {'status': 'error', 'reason': 'No position to exit'}
        
        exit_price = candle['close']
        
        # Close trade
        trade_result = self.trade_manager.close_trade(
            instrument_key=self.config.instrument_key,
            exit_price=exit_price,
            status=reason
        )
        
        # Update position manager
        self.position_manager.close_position(
            instrument_key=self.config.instrument_key,
            exit_price=exit_price
        )
        
        self.position = {}
        self.update_state(StrategyState.POSITION_CLOSED)
        
        logger.info(f"Executed exit: {trade_result}")
        return {'status': 'success', 'result': trade_result} 
    
    def _calculate_morning_range(self, candles: pd.DataFrame) -> Dict[str, float]:
        """
        Calculate morning range values including MR value.
        
        Args:
            candles: DataFrame with candle data for the day
            
        Returns:
            Dict with:
            - mr_high: Morning range high
            - mr_low: Morning range low
            - mr_size: Morning range size
            - mr_value: Morning range value (14-day ATR / MR size)
            - is_valid: Boolean indicating if MR value > 3
            - error: Error message if any (None if successful)
        """
        if candles.empty:
            logger.warning("Empty DataFrame provided for morning range calculation")
            return {
                'mr_high': 0,
                'mr_low': 0,
                'mr_size': 0,
                'mr_value': 0,
                'is_valid': False,
                'error': 'Empty DataFrame provided'
            }
            
        try:
            # Validate required columns
            required_columns = ['timestamp', 'high', 'low']
            missing_columns = [col for col in required_columns if col not in candles.columns]
            if missing_columns:
                error_msg = f"Missing required columns: {missing_columns}"
                logger.error(error_msg)
                return {
                    'mr_high': 0,
                    'mr_low': 0,
                    'mr_size': 0,
                    'mr_value': 0,
                    'is_valid': False,
                    'error': error_msg
                }
            
            # Calculate morning range values
            morning_candle = candles.iloc[0]
            mr_high = morning_candle['high']
            mr_low = morning_candle['low']
            mr_size = mr_high - mr_low
            
            if mr_size <= 0:
                error_msg = "Invalid morning range size (high <= low)"
                logger.error(error_msg)
                return {
                    'mr_high': mr_high,
                    'mr_low': mr_low,
                    'mr_size': mr_size,
                    'mr_value': 0,
                    'is_valid': False,
                    'error': error_msg
                }

            if mr_size < 1:
                error_msg = "Invalid morning range size. Size is less than 1"
                logger.error(error_msg)
                return {
                    'mr_high': mr_high,
                    'mr_low': mr_low,
                    'mr_size': mr_size,
                    'mr_value': 0,
                    'is_valid': False,
                    'error': error_msg
                }
            
            # Calculate 14-day ATR using daily candles
            logger.info("DAILY 14-ATR: {morning_candle['DAILY_ATR_14']}")
            atr_14 = morning_candle['DAILY_ATR_14']
            
            if atr_14 <= 0:
                error_msg = "Invalid ATR value (must be positive)"
                logger.error(error_msg)
                return {
                    'mr_high': mr_high,
                    'mr_low': mr_low,
                    'mr_size': mr_size,
                    'mr_value': 0,
                    'is_valid': False,
                    'error': error_msg
                }
            
            # Calculate MR value
            mr_value = (atr_14 / mr_size) * 1.2
            
            # Validate MR value
            is_valid = mr_value > 3
            
            logger.info(
                f"Morning Range Calculation - "
                f"High: {mr_high:.2f}, "
                f"Low: {mr_low:.2f}, "
                f"Size: {mr_size:.2f}, "
                f"ATR: {atr_14:.2f}, "
                f"MR Value: {mr_value:.2f}, "
                f"Valid: {is_valid}"
            )
            
            # Marking True for now. Later will remove the logic to specific entry type.
            return {
                'mr_high': mr_high,
                'mr_low': mr_low,
                'mr_size': mr_size,
                'mr_value': mr_value,
                'is_valid': is_valid,
                'error': None
            }
            
        except Exception as e:
            error_msg = f"Error calculating morning range: {str(e)}"
            logger.error(error_msg)
            return {
                'mr_high': 0,
                'mr_low': 0,
                'mr_size': 0,
                'mr_value': 0,
                'is_valid': False,
                'error': error_msg
            }