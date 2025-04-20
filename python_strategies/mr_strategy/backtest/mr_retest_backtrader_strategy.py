"""
Backtrader strategy implementation for Morning Range Retest strategy.

This module implements the Morning Range Retest strategy using Backtrader's framework,
maintaining the core logic while leveraging Backtrader's features.
"""

import backtrader as bt
import pandas as pd
import numpy as np
import pytz
import math
from typing import Dict, Optional, Any, List
import logging
import os
from datetime import datetime, time, timedelta
from pathlib import Path
from dataclasses import dataclass
from enum import Enum
import csv
from utils.utils import convert_utc_to_ist, get_nearest_price, get_closest_limit_price

# Create logs directory if it doesn't exist
log_dir = Path("logs")
log_dir.mkdir(exist_ok=True)

# Create results directory if it doesn't exist
results_dir = Path("results")
results_dir.mkdir(exist_ok=True)

# Configure logging
def setup_logger(name: str) -> logging.Logger:
    """Setup a logger with file and console handlers."""
    logger = logging.getLogger(name)
    logger.setLevel(logging.INFO)
    
    # Clear any existing handlers
    logger.handlers = []
    
    # Create formatters
    file_formatter = logging.Formatter(
        '%(asctime)s - %(levelname)s - %(message)s',
        datefmt='%Y-%m-%d %H:%M:%S'
    )
    console_formatter = logging.Formatter(
        '%(asctime)s - %(levelname)s - %(message)s',
        datefmt='%H:%M:%S'
    )
    
    # Create file handler with date-based filename
    current_date = datetime.now().strftime('%Y-%m-%d')
    file_handler = logging.FileHandler(
        log_dir / f'mr_retest_strategy_{current_date}.log',
        mode='a'
    )
    file_handler.setLevel(logging.INFO)
    file_handler.setFormatter(file_formatter)
    
    # Create console handler
    console_handler = logging.StreamHandler()
    console_handler.setLevel(logging.INFO)
    console_handler.setFormatter(console_formatter)
    
    # Add handlers to logger
    logger.addHandler(file_handler)
    logger.addHandler(console_handler)
    
    return logger

# Initialize logger
logger = setup_logger(__name__)

class TradeStatus(Enum):
    """Status of a trade."""
    PENDING = "pending"
    ACTIVE = "active"
    PARTIAL_TAKE_PROFIT = "partial_tp"
    TRAILING = "trailing"
    BREAKEVEN = "breakeven"
    STOPPED_OUT = "stopped_out"
    TAKE_PROFIT = "take_profit"
    CLOSED = "closed"
    EXPIRED = "expired"
    REJECTED = "rejected"

class TradeType(Enum):
    """Type of trade entry."""
    RETEST_ENTRY = "RETEST_ENTRY"

@dataclass
class TradeRecord:
    """Record for tracking trade information."""
    date: str
    name: str
    pnl: float
    status: str
    direction: str
    trade_type: str
    max_r_multiple: float
    entry_time: datetime
    exit_time: datetime
    entry_price: float
    exit_price: float
    stop_loss: float
    risk_amount: float

@dataclass
class TakeProfitLevel:
    """Configuration for a take profit level."""
    r_multiple: float  # Target in R multiples
    size_percentage: float  # Percentage of position to close
    trail_activation: bool = False  # Whether to activate trailing stop
    move_sl_to_be: bool = False  # Whether to move stop loss to breakeven

class MorningRangeRetestStrategy(bt.Strategy):
    """
    Backtrader implementation of the Morning Range Retest strategy.
    
    This strategy:
    1. Identifies the morning range (9:15-9:20)
    2. Validates MR range size and conditions
    3. Detects breakouts and retests
    4. Manages positions with advanced features
    """
    
    # ============================================
    # Strategy Parameters
    # ============================================
    params = (
        ('mr_start_time', time(9, 15)),  # Morning range start time
        ('mr_end_time', time(9, 20)),    # Morning range end time
        ('market_close_time', time(15, 20)),  # Market close time
        ('stop_loss_pct', 0.75),         # Stop loss percentage
        ('risk_per_trade', 30),          # Risk per trade in percentage
        ('use_daily_indicators', True),  # Whether to use daily indicators
        ('min_mr_size_pct', 0.002),     # Minimum MR size as percentage of price
        ('max_mr_size_pct', 0.01),      # Maximum MR size as percentage of price
        ('min_risk_reward', 2.0),       # Minimum risk-reward ratio
        ('max_trade_duration', 600),    # Maximum trade duration in minutes
        ('breakeven_r', 1.0),           # R multiple to move to breakeven
        ('trail_activation_r', 2.0),    # R multiple to activate trailing stop
        ('trail_step_pct', 0.002),      # Trailing stop step size
        ('max_retest_distance', 0.1),   # Maximum distance for valid retest (%)
        ('min_breakout_move', 0.3),     # Minimum move away from range (%)
        ('respect_trend', True),        # Whether to respect daily trend
        ('tp1_rr', 3.0),               # First take profit R:R
        ('tp2_rr', 5.0),               # Second take profit R:R
        ('tp3_rr', 7.0),               # Third take profit R:R
        ('tp1_size', 10.0),            # First take profit size (%)
        ('tp2_size', 40.0),            # Second take profit size (%)
    )

    # ============================================
    # Initialization and Setup
    # ============================================
    def __init__(self):
        """Initialize the strategy."""
        # Data references
        self.data = self.datas[0]
        
        # Trade tracking
        self.trade_records: List[TradeRecord] = []
        self.current_trade: Optional[TradeRecord] = None
        self.max_r_multiple = 0.0
        
        # Morning range tracking
        self.mr_high = None
        self.mr_low = None
        self.mr_high_long_entry_price = None
        self.mr_low_short_entry_price = None
        self.mr_size = None
        self.mr_established = False
        self.mr_valid = False
        self.day_skipped = False
        self.trade_closed_today = False
        
        # Breakout tracking
        self.long_breakout_detected = False
        self.short_breakout_detected = False
        self.long_breakout_price = None
        self.short_breakout_price = None
        self.long_max_move = None
        self.short_max_move = None
        
        # Retest tracking
        self.long_retest_detected = False
        self.short_retest_detected = False
        self.long_retest_qualified = False
        self.short_retest_qualified = False
        self.long_retest_price = None
        self.short_retest_price = None
        
        # Position tracking
        self.entry_price_long = None
        self.entry_price_short = None
        self.stop_loss_long = None
        self.stop_loss_short = None
        self.stop_loss_original_long = None
        self.stop_loss_original_short = None
        self.breakeven_level_long = None
        self.breakeven_level_short = None
        self._long_2_r_multiple_achieved = False
        self._short_2_r_multiple_achieved = False
        self.take_profit_levels_long = None
        self.take_profit_levels_short = None
        self.position_size_long = 0
        self.position_size_remaining_long = 0
        self.position_size_short = 0
        self.position_size_remaining_short = 0
        self.trade_status_long = None
        self.trade_status_short = None
        self.entry_time_long = None
        self.entry_time_short = None
        self.trailing_stop_active = False
        self.trailing_stop_level = None
        self.entry_executed_long = False
        self.entry_executed_short = False
        
        # Take profit levels
        self.take_profit_levels_long = [
            {
                "r_multiple": self.p.tp1_rr,
                "size_percentage": self.p.tp1_size,
                "move_sl_to_be": True,
                "trail_activation": True
            },
            {
                "r_multiple": self.p.tp2_rr,
                "size_percentage": self.p.tp2_size,
                "move_sl_to_be": False,
                "trail_activation": True
            },
            {
                "r_multiple": self.p.tp3_rr,
                "size_percentage": 100 - self.p.tp1_size - self.p.tp2_size,
                "move_sl_to_be": False,
                "trail_activation": False
            }
        ]
        self.take_profit_levels_short = self.take_profit_levels_long
        self.executed_take_profits = []
        
        # Indicator tracking
        self.ema_50 = bt.indicators.EMA(self.data.close, period=50)
        self.rsi = bt.indicators.RSI(self.data.close, period=14)
        self.atr = bt.indicators.ATR(self.data, period=14)
        
        logger.info(f"{self._get_log_prefix()} - Initialized MorningRangeRetestStrategy with params: {self.p}")

    # ============================================
    # Utility Methods
    # ============================================
    def _get_log_prefix(self) -> str:
        """Get the log prefix with instrument name and formatted date/time."""
        dt = self.data.datetime.datetime(0)
        # Convert it to IST
        utc = pytz.utc
        ist = pytz.timezone('Asia/Kolkata')
        dt_ist = utc.localize(dt).astimezone(ist)

        # Format the IST datetime
        formatted_dt = dt_ist.strftime("%d-%m-%Y %H:%M:%S")

        return f"[{self.data._name} {formatted_dt}]"

    def _reset_day_state(self):
        """Reset all day-specific state variables."""
        # Reset MR values
        self.mr_high = None
        self.mr_low = None
        self.mr_high_long_entry_price = None
        self.mr_low_short_entry_price = None
        self.mr_size = None
        self.mr_established = False
        self.mr_valid = False
        self.day_skipped = False
        self.trade_closed_today = False
        
        # Reset breakout tracking
        self.long_breakout_detected = False
        self.short_breakout_detected = False
        self.long_breakout_price = None
        self.short_breakout_price = None
        self.long_max_move = None
        self.short_max_move = None
        
        # Reset retest tracking
        self.long_retest_detected = False
        self.short_retest_detected = False
        self.long_retest_qualified = False
        self.short_retest_qualified = False
        self.long_retest_price = None
        self.short_retest_price = None
        
        # Reset position tracking
        self.entry_price_long = None
        self.entry_price_short = None
        self.stop_loss_long = None
        self.stop_loss_short = None
        self.stop_loss_original_long = None
        self.stop_loss_original_short = None
        self.breakeven_level_long = None
        self.breakeven_level_short = None
        self._long_2_r_multiple_achieved = False
        self._short_2_r_multiple_achieved = False
        self.take_profit_levels_long = None
        self.take_profit_levels_short = None
        self.position_size_long = 0
        self.position_size_remaining_long = 0
        self.position_size_short = 0
        self.position_size_remaining_short = 0
        self.trade_status_long = None
        self.trade_status_short = None
        self.entry_time_long = None
        self.entry_time_short = None
        self.trailing_stop_active = False
        self.trailing_stop_level = None
        self.entry_executed_long = False
        self.entry_executed_short = False
        self.executed_take_profits = []
        self.max_r_multiple = 0.0

    def _is_new_day(self, current_time: datetime) -> bool:
        """Check if we're processing a new day."""
        current_time_day_format = current_time.strftime("%d-%m-%Y")
        if not hasattr(self, '_last_processed_day'):
            self._last_processed_day = current_time_day_format
            return True
            
        is_new_day = current_time_day_format != self._last_processed_day
        if is_new_day:
            self._last_processed_day = current_time_day_format
            self._reset_day_state()
        return is_new_day

    def _update_morning_range(self):
        """Update morning range values."""
        # Update high and low during morning range period
        if self.mr_high is None or self.data.high[0] > self.mr_high:
            self.mr_high = self.data.high[0]
            logger.info(f"{self._get_log_prefix()} - Updated morning range high: {self.mr_high:.2f}")
        if self.mr_low is None or self.data.low[0] < self.mr_low:
            self.mr_low = self.data.low[0]
            logger.info(f"{self._get_log_prefix()} - Updated morning range low: {self.mr_low:.2f}")
            
        self.mr_size = self.mr_high - self.mr_low
        # Calculate entry prices using nearest price method
        self.mr_high_long_entry_price = get_closest_limit_price(self.mr_high, above=True)
        self.mr_low_short_entry_price = get_closest_limit_price(self.mr_low, above=False)
        self.mr_established = True
        logger.info(f"{self._get_log_prefix()} - Morning Range Established - High: {self.mr_high:.2f}, Low: {self.mr_low:.2f}, Size: {self.mr_size:.2f}, Long Entry: {self.mr_high_long_entry_price:.2f}, Short Entry: {self.mr_low_short_entry_price:.2f}")

    def _validate_mr_range(self) -> bool:
        """Validate morning range size and conditions."""
        if not self.mr_established:
            return False
            
        # Calculate MR size as percentage of average price
        avg_price = (self.mr_high + self.mr_low) / 2
        mr_size_pct = self.mr_size / avg_price

        # fetch the daily atr from the data
        daily_atr = self.data.tick_DAILY_ATR_14
        mr_size = abs(self.mr_high - self.mr_low)
        mr_size_pct = daily_atr / mr_size if mr_size > 0 else 0

        if mr_size_pct < 3:
            logger.info(f"{self._get_log_prefix()} - MR size {mr_size_pct:.4f} Less than 3, Skipping day")
            return False
            
        self.mr_valid = True
        logger.info(f"{self._get_log_prefix()} - MR range validated - Size: {mr_size_pct:.4f}, High: {self.mr_high:.2f}, Low: {self.mr_low:.2f}")

        return True

    def _detect_breakouts(self):
        """Detect breakouts from the morning range."""
        if not self.mr_established or not self.mr_valid:
            return
            
        # Long breakout detection
        if not self.long_breakout_detected and self.data.high[0] > self.mr_high:
            move_pct = ((self.data.high[0] - self.mr_high) / self.mr_high) * 100
            if move_pct >= self.p.min_breakout_move:
                self.long_breakout_detected = True
                self.long_breakout_price = self.data.high[0]
                self.long_max_move = self.data.high[0]
                logger.info(f"{self._get_log_prefix()} - Long breakout detected at {self.long_breakout_price:.2f} ({move_pct:.2f}%)")
                
        # Update maximum move after long breakout
        if self.long_breakout_detected and self.data.high[0] > self.long_max_move:
            self.long_max_move = self.data.high[0]
            
        # Short breakout detection
        if not self.short_breakout_detected and self.data.low[0] < self.mr_low:
            move_pct = ((self.mr_low - self.data.low[0]) / self.mr_low) * 100
            if move_pct >= self.p.min_breakout_move:
                self.short_breakout_detected = True
                self.short_breakout_price = self.data.low[0]
                self.short_max_move = self.data.low[0]
                logger.info(f"{self._get_log_prefix()} - Short breakout detected at {self.short_breakout_price:.2f} ({move_pct:.2f}%)")
                
        # Update maximum move after short breakout
        if self.short_breakout_detected and self.data.low[0] < self.short_max_move:
            self.short_max_move = self.data.low[0]

    def _detect_retests(self):
        """Detect retests of the morning range levels."""
        if not self.mr_established or not self.mr_valid:
            return
            
        # Long retest detection (retest of MR high)
        if self.long_breakout_detected and not self.long_retest_detected:
            # Check if there was a meaningful breakout first
            move_pct = ((self.long_max_move - self.mr_high) / self.mr_high) * 100
            if move_pct >= self.p.min_breakout_move:
                # Check if price has pulled back to MR high level (within tolerance)
                high_retest_tolerance = self.mr_high * (1 + self.p.max_retest_distance / 100)
                low_retest_tolerance = self.mr_high * (1 - self.p.max_retest_distance / 100)
                
                if self.data.low[0] <= high_retest_tolerance and self.data.low[0] >= low_retest_tolerance:
                    self.long_retest_detected = True
                    self.long_retest_price = min(self.data.high[0], self.mr_high)
                    
                    # Qualify the retest by checking if it's a pullback (not a failed breakout)
                    if self.data.high[0] > self.mr_high:
                        self.long_retest_qualified = True
                        logger.info(f"{self._get_log_prefix()} - Long retest qualified at {self.long_retest_price:.2f}")
                    else:
                        logger.info(f"{self._get_log_prefix()} - Long retest detected but not qualified")
                        
        # Short retest detection (retest of MR low)
        if self.short_breakout_detected and not self.short_retest_detected:
            # Check if there was a meaningful breakout first
            move_pct = ((self.mr_low - self.short_max_move) / self.mr_low) * 100
            if move_pct >= self.p.min_breakout_move:
                # Check if price has pulled back to MR low level (within tolerance)
                high_retest_tolerance = self.mr_low * (1 + self.p.max_retest_distance / 100)
                low_retest_tolerance = self.mr_low * (1 - self.p.max_retest_distance / 100)
                
                if self.data.high[0] >= low_retest_tolerance and self.data.high[0] <= high_retest_tolerance:
                    self.short_retest_detected = True
                    self.short_retest_price = max(self.data.low[0], self.mr_low)
                    
                    # Qualify the retest by checking if it's a pullback (not a failed breakout)
                    if self.data.low[0] < self.mr_low:
                        self.short_retest_qualified = True
                        logger.info(f"{self._get_log_prefix()} - Short retest qualified at {self.short_retest_price:.2f}")
                    else:
                        logger.info(f"{self._get_log_prefix()} - Short retest detected but not qualified")

    def _validate_trade_setup(self, entry_price: float, position_type: str) -> bool:
        """Validate trade setup before entry."""
        # Calculate risk-reward ratio
        levels = self._calculate_trade_levels(entry_price, position_type)
        risk_amount = levels["risk_amount"]
        reward_amount = abs(entry_price - levels["take_profit_levels"][-1]["price"])
        risk_reward = reward_amount / risk_amount if risk_amount > 0 else 0
        
        if risk_reward < self.p.min_risk_reward:
            logger.info(f"{self._get_log_prefix()} - Risk-reward {risk_reward:.2f} below minimum {self.p.min_risk_reward}")
            return False
            
        # Validate stop loss distance
        min_stop_distance = entry_price * 0.001  # Minimum 0.1% stop distance
        if risk_amount < min_stop_distance:
            logger.info(f"{self._get_log_prefix()} - Stop distance {risk_amount:.2f} below minimum {min_stop_distance:.2f}")
            return False
            
        logger.info(f"{self._get_log_prefix()} - Trade setup valid - Risk: {risk_amount:.2f}, Reward: {reward_amount:.2f}, R:R: {risk_reward:.2f}")
        return True

    def _calculate_trade_levels(self, entry_price: float, position_type: str) -> Dict[str, float]:
        """Calculate trade levels including take profit targets."""
        if position_type == "LONG":
            stop_loss = entry_price * (1 - self.p.stop_loss_pct/100)
            risk_amount = entry_price - stop_loss
            breakeven_level = entry_price + (risk_amount * self.p.breakeven_r)
            
            # Calculate take profit levels
            take_profit_levels = [
                {
                    "price": entry_price + (risk_amount * tp["r_multiple"]),
                    "size_percentage": tp["size_percentage"],
                    "r_multiple": tp["r_multiple"],
                    "trail_activation": tp["trail_activation"],
                    "move_sl_to_be": tp["move_sl_to_be"]
                }
                for tp in self.take_profit_levels_long
            ]
        else:  # SHORT
            stop_loss = entry_price * (1 + self.p.stop_loss_pct/100)
            risk_amount = stop_loss - entry_price
            breakeven_level = entry_price - (risk_amount * self.p.breakeven_r)
            
            # Calculate take profit levels
            take_profit_levels = [
                {
                    "price": entry_price - (risk_amount * tp["r_multiple"]),
                    "size_percentage": tp["size_percentage"],
                    "r_multiple": tp["r_multiple"],
                    "trail_activation": tp["trail_activation"],
                    "move_sl_to_be": tp["move_sl_to_be"]
                }
                for tp in self.take_profit_levels_short
            ]
        
        return {
            "stop_loss": stop_loss,
            "breakeven_level": breakeven_level,
            "risk_amount": risk_amount,
            "take_profit_levels": take_profit_levels
        }

    def _add_long_retest_order(self):
        """Add a long order on retest of MR high."""
        if self._validate_trade_setup(self.long_retest_price, "LONG"):
            self.entry_price_long = self.long_retest_price
            self.entry_time_long = self.data.datetime.datetime(0)
            self.trade_status_long = TradeStatus.ACTIVE
            
            # Calculate trade levels
            levels = self._calculate_trade_levels(self.entry_price_long, "LONG")
            self.stop_loss_original_long = levels["stop_loss"]
            self.stop_loss_long = levels["stop_loss"]
            self.breakeven_level_long = levels["breakeven_level"]
            self.take_profit_levels_long = levels["take_profit_levels"]
            self.executed_take_profits = []

            sl_points = self.entry_price_long - self.stop_loss_long
            self.position_size_long = math.floor(self.p.risk_per_trade / sl_points)
            self.position_size_remaining_long = math.floor(self.position_size_long)
            
            # Execute entry order
            self.buy(size=self.position_size_long, exectype=bt.Order.StopLimit, 
                    price=self.long_retest_price, plimit=self.long_retest_price)
            logger.info(f"{self._get_log_prefix()} - Long retest entry order placed at {self.long_retest_price:.2f}")

    def _add_short_retest_order(self):
        """Add a short order on retest of MR low."""
        if self._validate_trade_setup(self.short_retest_price, "SHORT"):
            self.entry_price_short = self.short_retest_price
            self.entry_time_short = self.data.datetime.datetime(0)
            self.trade_status_short = TradeStatus.ACTIVE
            
            # Calculate trade levels
            levels = self._calculate_trade_levels(self.entry_price_short, "SHORT")
            self.stop_loss_original_short = levels["stop_loss"]
            self.stop_loss_short = levels["stop_loss"]
            self.breakeven_level_short = levels["breakeven_level"]
            self.take_profit_levels_short = levels["take_profit_levels"]
            self.executed_take_profits = []

            sl_points = self.stop_loss_short - self.entry_price_short
            self.position_size_short = math.floor(self.p.risk_per_trade / sl_points)
            self.position_size_remaining_short = math.floor(self.position_size_short)
            
            # Execute entry order
            self.sell(size=self.position_size_short, exectype=bt.Order.StopLimit, 
                     price=self.short_retest_price, plimit=self.short_retest_price)
            logger.info(f"{self._get_log_prefix()} - Short retest entry order placed at {self.short_retest_price:.2f}")

    def _manage_position(self):
        """Manage open position."""
        current_price = self.data.close[0]
        current_time = self.data.datetime.datetime(0)
        current_time_ist = convert_utc_to_ist(current_time)
        
        logger.info(f"{self._get_log_prefix()} - Position State - Size: {self.position.size}, Remaining Long: {self.position_size_remaining_long}, Remaining Short: {self.position_size_remaining_short}")
        
        # Check if we're in a long position
        if self.position.size > 0:
            # Check stop loss
            if current_price <= self.stop_loss_long:
                self.close()
                logger.info(f"{self._get_log_prefix()} - Long position stopped out at {current_price:.2f}")
                return
                
            # Move to breakeven after 1R
            if not self._long_2_r_multiple_achieved and current_price >= self.breakeven_level_long:
                self._long_2_r_multiple_achieved = True
                self.stop_loss_long = self.entry_price_long
                logger.info(f"{self._get_log_prefix()} - Long position moved to breakeven at {self.entry_price_long:.2f}")
                
            # Check take profit levels
            for i, tp in enumerate(self.take_profit_levels_long):
                if tp in self.executed_take_profits:
                    continue
                    
                if current_price >= tp["price"]:
                    # Execute partial exit
                    exit_size = math.floor(self.position_size_remaining_long * (tp["size_percentage"] / 100))
                    if exit_size > 0:
                        self.sell(size=exit_size)
                        self.executed_take_profits.append(tp)
                        self.position_size_remaining_long -= exit_size
                        
                        if self.max_r_multiple < tp["r_multiple"]:
                            self.max_r_multiple = tp["r_multiple"]
                            
                        logger.info(f"{self._get_log_prefix()} - Long take profit {tp['r_multiple']}R hit at {current_price:.2f}, exited {exit_size} shares")
                        
                        # If remaining size is zero, close the position
                        if self.position_size_remaining_long <= 0:
                            logger.info(f"{self._get_log_prefix()} - Long position fully closed")
                            return
                            
        # Check if we're in a short position
        elif self.position.size < 0:
            # Check stop loss
            if current_price >= self.stop_loss_short:
                self.close()
                logger.info(f"{self._get_log_prefix()} - Short position stopped out at {current_price:.2f}")
                return
                
            # Move to breakeven after 1R
            if not self._short_2_r_multiple_achieved and current_price <= self.breakeven_level_short:
                self._short_2_r_multiple_achieved = True
                self.stop_loss_short = self.entry_price_short
                logger.info(f"{self._get_log_prefix()} - Short position moved to breakeven at {self.entry_price_short:.2f}")
                
            # Check take profit levels
            for i, tp in enumerate(self.take_profit_levels_short):
                if tp in self.executed_take_profits:
                    continue
                    
                if current_price <= tp["price"]:
                    # Execute partial exit
                    exit_size = math.floor(self.position_size_remaining_short * (tp["size_percentage"] / 100))
                    if exit_size > 0:
                        self.buy(size=exit_size)
                        self.executed_take_profits.append(tp)
                        self.position_size_remaining_short -= exit_size
                        
                        if self.max_r_multiple < tp["r_multiple"]:
                            self.max_r_multiple = tp["r_multiple"]
                            
                        logger.info(f"{self._get_log_prefix()} - Short take profit {tp['r_multiple']}R hit at {current_price:.2f}, exited {exit_size} shares")
                        
                        # If remaining size is zero, close the position
                        if self.position_size_remaining_short <= 0:
                            logger.info(f"{self._get_log_prefix()} - Short position fully closed")
                            return

    def next(self):
        """Called for each new candle."""
        try:
            # Skip if we don't have enough data
            if len(self) < 50:
                return
                
            # Get current time in IST
            current_time = self.data.datetime.datetime(0)
            current_time_ist = convert_utc_to_ist(current_time)
            
            # Check if we're starting a new day
            if self._is_new_day(current_time_ist):
                logger.info(f"{self._get_log_prefix()} - Starting new trading day")
                
            # If we're skipping this day, return early
            if self.day_skipped:
                return
            
            # Check if we're in trading hours
            if current_time_ist.time() > self.p.market_close_time:
                logger.info(f"{self._get_log_prefix()} - Market closed timing reached, skipping rest of the day")
                return
                
            candle_info = f"Time: {current_time_ist}, Open: {self.data.open[0]:.2f}, High: {self.data.high[0]:.2f}, Low: {self.data.low[0]:.2f}, Close: {self.data.close[0]:.2f}, Volume: {self.data.volume[0]}"
            
            # Morning range establishment
            if not self.mr_established and self.p.mr_start_time <= current_time_ist.time() < self.p.mr_end_time:
                logger.info(f"{self._get_log_prefix()} - {candle_info} - Updating morning range")
                self._update_morning_range()
                
            # Breakout and retest detection after morning range is established
            if self.mr_established and not self.position:
                self._detect_breakouts()
                self._detect_retests()
                
                # Check for qualified retests and place orders
                if self.long_retest_qualified and not self.entry_executed_long:
                    self._add_long_retest_order()
                elif self.short_retest_qualified and not self.entry_executed_short:
                    self._add_short_retest_order()
                
            # Position management
            if self.position:
                self._manage_position()
                
        except Exception as e:
            # Log the error with full context
            logging.exception("message")
            error_context = {
                "current_time": current_time_ist if 'current_time_ist' in locals() else None,
                "candle_info": candle_info if 'candle_info' in locals() else None,
                "mr_established": self.mr_established if hasattr(self, 'mr_established') else None,
                "position": self.position.size if hasattr(self, 'position') else None,
                "error": str(e)
            }
            logger.error(f"{self._get_log_prefix()} - Error in next() method: {error_context}")
            
            # If we have an open position, close it to prevent further losses
            if hasattr(self, 'position') and self.position:
                try:
                    logger.warning(f"{self._get_log_prefix()} - Closing position due to error")
                    self.close()
                except Exception as close_error:
                    logger.error(f"{self._get_log_prefix()} - Error while closing position: {str(close_error)}")
            
            # Re-raise the exception to ensure proper error handling by Backtrader
            raise

    def notify_order(self, order):
        """Handle order notifications."""
        if order.status in [order.Submitted, order.Accepted]:
            logger.info(f"{self._get_log_prefix()} - Order {order.getstatusname()} - Size: {order.size}, Price: {order.price}, Position Size: {self.position.size}, Remaining Size: {self.position_size_remaining_long}")
            return
            
        if order.status in [order.Completed]:
            if order.isbuy():
                logger.info(f"{self._get_log_prefix()} - Buy Executed - Price: {order.executed.price:.2f}, Cost: {order.executed.value:.2f}, Comm: {order.executed.comm:.2f}, Size: {order.executed.size}")
                self.entry_executed_long = True
            else:
                logger.info(f"{self._get_log_prefix()} - Sell Executed - Price: {order.executed.price:.2f}, Cost: {order.executed.value:.2f}, Comm: {order.executed.comm:.2f}, Size: {order.executed.size}")
                self.entry_executed_short = True
        elif order.status in [order.Canceled, order.Margin, order.Rejected]:
            cash = self.broker.get_cash()
            value = self.broker.get_value()
            price = order.price
            size = order.size if order.size is not None else "N/A"
            
            cash_str = f"{cash:.2f}" if cash is not None else "N/A"
            value_str = f"{value:.2f}" if value is not None else "N/A"
            price_str = f"{price:.2f}" if price is not None else "N/A"
            logger.warning(
                f"{self._get_log_prefix()} - Order Canceled/Margin/Rejected - "
                f"Status: {order.getstatusname()}, Size: {size}, "
                f"Price: {price_str}, Current Cash: {cash_str}, Portfolio Value: {value_str}"
            )

    def notify_trade(self, trade):
        """Handle trade notifications."""
        if not trade.isclosed:
            # Update max R multiple during the trade
            if self.current_trade:
                current_r = abs(trade.price - self.current_trade.entry_price) / self.current_trade.risk_amount
                self.max_r_multiple = max(self.max_r_multiple, current_r)
            return
        
        if self.entry_price_long > self.stop_loss_original_long:
            direction = "LONG"
        elif self.entry_price_short < self.stop_loss_original_short:
            direction = "SHORT"
        else:
            direction = "INVALID"
            
        # Create trade record for closed trade
        trade_record = TradeRecord(
            date=self.data.datetime.date(0).strftime('%Y-%m-%d'),
            name=self.data._name,
            pnl=trade.pnl,
            status=TradeStatus.CLOSED.value,
            direction=direction,
            trade_type=TradeType.RETEST_ENTRY.value,
            max_r_multiple=self.max_r_multiple,
            entry_time=trade.dtopen,
            exit_time=trade.dtclose,
            entry_price=trade.price,
            exit_price=trade.pnl / trade.size + trade.price if trade.size != 0 else 0,
            stop_loss=self.stop_loss_long if direction == "LONG" else self.stop_loss_short,
            risk_amount=abs(trade.price - self.stop_loss_long) if direction == "LONG" else abs(trade.price - self.stop_loss_short)
        )
        
        # Add to trade records
        self.trade_records.append(trade_record)
        self.trade_closed_today = True
        self.current_trade = None
        self.max_r_multiple = 0.0

        stop_loss = self.stop_loss_long if direction == "LONG" else self.stop_loss_short
        
        logger.info(f"{self._get_log_prefix()} - Trade Closed - PnL: {trade.pnl:.2f}, Gross: {trade.pnlcomm:.2f}, Entry: {trade.price:.2f}, Exit: {trade.pnl / trade.size + trade.price if trade.size != 0 else 0:.2f}, Stop loss: {stop_loss:.2f}, Size: {trade.size}")
        logger.info(f"{self._get_log_prefix()} - Trade closed, no new trades allowed for the day")

    def stop(self):
        """Called at the end of the backtest."""
        csv_path = results_dir / "trade_summary.csv"
        fieldnames = ['Date', 'Name', 'P&L', 'Status', 'Direction', 'Trade Type', 'Max R Multiple']
        
        # Read existing trades if file exists
        existing_trades = {}
        if csv_path.exists():
            with open(csv_path, 'r', newline='') as csvfile:
                reader = csv.DictReader(csvfile)
                for row in reader:
                    key = (row['Date'], row['Name'])
                    existing_trades[key] = row
        
        # Update or add new trades
        for record in self.trade_records:
            key = (record.date, record.name)
            trade_data = {
                'Date': record.date,
                'Name': record.name,
                'P&L': round(record.pnl, 2),
                'Status': "PROFIT" if record.pnl > 0 else "LOSS",
                'Direction': record.direction,
                'Trade Type': record.trade_type,
                'Max R Multiple': round(record.max_r_multiple, 2)
            }
            existing_trades[key] = trade_data
        
        # Write all trades back to CSV
        with open(csv_path, 'w', newline='') as csvfile:
            writer = csv.DictWriter(csvfile, fieldnames=fieldnames)
            writer.writeheader()
            for trade in existing_trades.values():
                writer.writerow(trade)
                
        logger.info(f"{self._get_log_prefix()} - Trade summary updated in {csv_path}")
