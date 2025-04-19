"""
Backtrader strategy implementation for Morning Range strategy.

This module implements the Morning Range strategy using Backtrader's framework,
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
from utils.utils import convert_utc_to_ist, get_nearest_price

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
        log_dir / f'mr_strategy_{current_date}.log',
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
    IMMEDIATE_BREAKOUT = "1ST_ENTRY"
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

class MorningRangeStrategy(bt.Strategy):
    """
    Backtrader implementation of the Morning Range strategy.
    
    This strategy:
    1. Identifies the morning range (9:15-9:20)
    2. Validates MR range size and conditions
    3. Generates signals based on range breakouts and technical indicators
    4. Manages positions with advanced features from trade_manager.py
    """
    
    params = (
        ('mr_start_time', time(9, 15)),  # Morning range start time
        ('mr_end_time', time(9, 20)),    # Morning range end time
        ('market_close_time', time(15, 20)),  # Market close time
        ('stop_loss_pct', 0.005),         # Stop loss percentage
        ('target_pct', 0.02),            # Target percentage
        ('position_size', 1),            # Position size in units
        ('risk_per_trade', 50),          # Risk per trade in percentage
        ('use_daily_indicators', True),  # Whether to use daily indicators
        ('min_mr_size_pct', 0.002),     # Minimum MR size as percentage of price
        ('max_mr_size_pct', 0.01),      # Maximum MR size as percentage of price
        ('min_risk_reward', 2.0),       # Minimum risk-reward ratio
        ('max_trade_duration', 600),    # Maximum trade duration in minutes
        ('breakeven_r', 1.0),           # R multiple to move to breakeven
        ('trail_activation_r', 2.0),    # R multiple to activate trailing stop
        ('trail_step_pct', 0.002),      # Trailing stop step size
    )
    
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
        self.day_skipped = False  # Flag to track if we're skipping the current day
        self.trade_closed_today = False  # Flag to track if a trade was closed today
        
        # Position tracking
        self.entry_price = None
        self.stop_loss = None
        self.stop_loss_original = None
        self.target = None
        self.position_size = self.p.position_size
        self.trade_status = None
        self.entry_time = None
        self.trailing_stop_active = False
        self.trailing_stop_level = None
        
        # Take profit levels as dictionaries
        self.take_profit_levels = [
            {
                "r_multiple": 3.0,
                "size_percentage": 30,
                "move_sl_to_be": True,
                "trail_activation": True
            },
            {
                "r_multiple": 5.0,
                "size_percentage": 50,
                "move_sl_to_be": False,
                "trail_activation": True
            },
            {
                "r_multiple": 7.0,
                "size_percentage": 100,
                "move_sl_to_be": False,
                "trail_activation": False
            }
        ]
        self.executed_take_profits = []
        
        # Indicator tracking
        self.ema_50 = bt.indicators.EMA(self.data.close, period=50)
        self.rsi = bt.indicators.RSI(self.data.close, period=14)
        self.atr = bt.indicators.ATR(self.data, period=14)
        
        logger.info(f"{self._get_log_prefix()} - Initialized MorningRangeStrategy with params: {self.p}")
        
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
        self.mr_high = None
        self.mr_low = None
        self.mr_high_long_entry_price = None
        self.mr_low_short_entry_price = None
        self.mr_size = None
        self.mr_established = False
        self.mr_valid = False
        self.day_skipped = False
        self.trade_closed_today = False  # Reset trade closed flag
        self.entry_price = None
        self.stop_loss = None
        self.target = None
        self.trade_status = None
        self.entry_time = None
        self.trailing_stop_active = False
        self.trailing_stop_level = None
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
        
    def _calculate_trade_levels(self, entry_price: float, position_type: str) -> Dict[str, float]:
        """Calculate trade levels including take profit targets."""
        if position_type == "LONG":
            stop_loss = entry_price * (1 - self.p.stop_loss_pct)
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
                for tp in self.take_profit_levels
            ]
        else:  # SHORT
            stop_loss = entry_price * (1 + self.p.stop_loss_pct)
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
                for tp in self.take_profit_levels
            ]
        
        return {
            "stop_loss": stop_loss,
            "breakeven_level": breakeven_level,
            "risk_amount": risk_amount,
            "take_profit_levels": take_profit_levels
        }
        
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
            
            if current_time_ist.time() > self.p.market_close_time:
                logger.info(f"{self._get_log_prefix()} - Market closed timing reached, skipping rest of the day")
                return
                
            candle_info = f"Time: {current_time_ist}, Open: {self.data.open[0]:.2f}, High: {self.data.high[0]:.2f}, Low: {self.data.low[0]:.2f}, Close: {self.data.close[0]:.2f}, Volume: {self.data.volume[0]}"
            
            # Morning range establishment
            if not self.mr_established and self.p.mr_start_time <= current_time_ist.time() < self.p.mr_end_time:
                logger.info(f"{self._get_log_prefix()} - {candle_info} - Updating morning range")
                self._update_morning_range()
                
            # Trading logic after morning range is established
            if self.mr_established and not self.position:
                logger.info(f"{self._get_log_prefix()} - {candle_info} - Checking trading signals")
                self._check_trading_signals()
                
            # Position management
            if self.position:
                logger.info(f"{self._get_log_prefix()} - {candle_info} - Managing position")
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
        self.mr_high_long_entry_price = self.mr_high
        self.mr_low_short_entry_price = self.mr_low
        self.mr_established = True
        logger.info(f"{self._get_log_prefix()} - Morning Range Established - High: {self.mr_high:.2f}, Low: {self.mr_low:.2f}, Size: {self.mr_size:.2f}, Long Entry: {self.mr_high_long_entry_price:.2f}, Short Entry: {self.mr_low_short_entry_price:.2f}")
            
            
    def _check_trading_signals(self):
        """Check for trading signals based on morning range and indicators."""
        # Validate MR range first
        if not self._validate_mr_range():
            self.day_skipped = True
            logger.info(f"{self._get_log_prefix()} - MR validation failed, skipping rest of the day")
            return
            
        # Check if a trade was already closed today
        if self.trade_closed_today:
            logger.info(f"{self._get_log_prefix()} - Trade already closed today, no new trades allowed")
            return
            
        # Check for breakout signals
        if self.data.high[0] > self.mr_high_long_entry_price:
            # Bullish breakout
            logger.info(f"{self._get_log_prefix()} - Potential bullish breakout detected - Price: {self.data.high[0]:.2f}, MR long entry price: {self.mr_high_long_entry_price:.2f}")
            if self._validate_bullish_signal():
                self._enter_long()
                
        elif self.data.low[0] < self.mr_low_short_entry_price:
            # Bearish breakout
            logger.info(f"{self._get_log_prefix()} - Potential bearish breakout detected - Price: {self.data.low[0]:.2f}, MR short entry price: {self.mr_low_short_entry_price:.2f}")
            if self._validate_bearish_signal():
                self._enter_short()
                
    def _validate_bullish_signal(self) -> bool:
        """Validate bullish breakout signal."""
        # Check RSI is not overbought
        # if self.rsi[0] > 70:
        #     logger.info(f"{self._get_log_prefix()} - Bullish signal rejected - RSI {self.rsi[0]:.2f} > 70")
        #     return False
            
        # Check price is above EMA
        if self.data.close[0] < self.ema_50[0]:
            logger.info(f"{self._get_log_prefix()} - Bullish signal rejected - Price {self.data.close[0]:.2f} < EMA {self.ema_50[0]:.2f}")
            return False
            
        # Check volume confirmation
        # if self.data.volume[0] < self.data.volume[-1]:
        #     logger.info(f"{self._get_log_prefix()} - Bullish signal rejected - Volume {self.data.volume[0]} < Previous {self.data.volume[-1]}")
        #     return False
            
        # Validate trade setup
        if not self._validate_trade_setup(self.mr_high_long_entry_price, "LONG"):
            return False
            
        logger.info(f"{self._get_log_prefix()} - Bullish signal validated - RSI: {self.rsi[0]:.2f}, Price: {self.data.close[0]:.2f}, EMA: {self.ema_50[0]:.2f}")
        return True
        
    def _validate_bearish_signal(self) -> bool:
        """Validate bearish breakout signal."""
        # Check RSI is not oversold
        # if self.rsi[0] < 30:
        #     logger.info(f"{self._get_log_prefix()} - Bearish signal rejected - RSI {self.rsi[0]:.2f} < 30")
        #     return False
            
        # Check price is below EMA
        if self.data.close[0] > self.ema_50[0]:
            logger.info(f"{self._get_log_prefix()} - Bearish signal rejected - Price {self.data.close[0]:.2f} > EMA {self.ema_50[0]:.2f}")
            return False
            
        # Check volume confirmation
        # if self.data.volume[0] < self.data.volume[-1]:
        #     logger.info(f"{self._get_log_prefix()} - Bearish signal rejected - Volume {self.data.volume[0]} < Previous {self.data.volume[-1]}")
        #     return False
            
        # Validate trade setup
        if not self._validate_trade_setup(self.mr_low_short_entry_price, "SHORT"):
            return False
            
        logger.info(f"{self._get_log_prefix()} - Bearish signal validated - RSI: {self.rsi[0]:.2f}, Price: {self.data.close[0]:.2f}, EMA: {self.ema_50[0]:.2f}")
        return True
        
    def _enter_long(self):
        """Enter long position."""
        self.entry_price = self.mr_high_long_entry_price
        self.entry_time = self.data.datetime.datetime(0)
        self.trade_status = TradeStatus.ACTIVE
        
        # Calculate trade levels
        levels = self._calculate_trade_levels(self.entry_price, "LONG")
        self.stop_loss_original = levels["stop_loss"]
        self.stop_loss = levels["stop_loss"]
        self.breakeven_level = levels["breakeven_level"]
        self.take_profit_levels = levels["take_profit_levels"]
        self.executed_take_profits = []

        sl_points = self.entry_price - self.stop_loss
        self.position_size = self.p.risk_per_trade / sl_points
        self.buy(size=self.position_size)
        
        logger.info(f"{self._get_log_prefix()} - Entered Long - Price: {self.entry_price:.2f}, SL: {self.stop_loss:.2f}, Size: {self.position_size}")
        
    def _enter_short(self):
        """Enter short position."""
        self.entry_price = self.mr_low_short_entry_price
        self.entry_time = self.data.datetime.datetime(0)
        self.trade_status = TradeStatus.ACTIVE
        
        # Calculate trade levels
        levels = self._calculate_trade_levels(self.entry_price, "SHORT")
        self.stop_loss_original = levels["stop_loss"]
        self.stop_loss = levels["stop_loss"]
        self.breakeven_level = levels["breakeven_level"]
        self.take_profit_levels = levels["take_profit_levels"]
        self.executed_take_profits = []

        sl_points = self.stop_loss - self.entry_price
        self.position_size = math.floor(self.p.risk_per_trade / sl_points)
        self.sell(size=self.position_size)
        
        logger.info(f"{self._get_log_prefix()} - Entered Short - Price: {self.entry_price:.2f}, SL: {self.stop_loss:.2f}, Size: {self.position_size}")
        
    def _manage_position(self):
        """Manage open position."""
        current_price = self.data.close[0]
        current_time = self.data.datetime.datetime(0)
        current_time_ist = convert_utc_to_ist(current_time)
        
        if self.entry_time is not None:
            trade_duration = (current_time - self.entry_time).total_seconds() / 60
            if trade_duration > self.p.max_trade_duration:
                logger.info(f"{self._get_log_prefix()} - Trade duration exceeded {self.p.max_trade_duration} minutes")
                self.close()
                return
        else:
            logger.warning(f"{self._get_log_prefix()} - Skipping trade duration check: entry_time is None")
            
        # Check stop loss
        if self.position.size > 0:  # Long position
            # Check for gap down
            if self.data.open[0] <= self.stop_loss:
                logger.info(f"{self._get_log_prefix()} - Long Position Stopped Out (Gap Down) - Open: {self.data.open[0]:.2f}, SL: {self.stop_loss:.2f}")
                self.close(price=self.data.open[0])
                return
                
            # Check for stop loss hit during candle
            if self.data.low[0] <= self.stop_loss:
                # Calculate the exact price where stop was hit
                # This is an approximation - in reality, we don't know the exact price
                # We'll use the stop loss price as it's the worst case
                exit_price = self.stop_loss
                logger.info(f"{self._get_log_prefix()} - Long Position Stopped Out (Mid-Candle) - Price: {exit_price:.2f}, SL: {self.stop_loss:.2f}")
                self.close(price=exit_price)
                return
            
            if current_price > (self.entry_price + 2*(self.entry_price - self.stop_loss)):
                # Trail SL
                self.stop_loss = self.entry_price + (self.entry_price - self.stop_loss)
                logger.info(f"{self._get_log_prefix()} - Trailing stop loss to: {self.stop_loss:.2f}")
            # Check take profit levels
            for i, tp in enumerate(self.take_profit_levels):
                if tp in self.executed_take_profits:
                    continue
                    
                if current_price >= tp["price"]:
                    # Execute partial exit
                    exit_size = self.position_size * (tp["size_percentage"] / 100)
                    self.sell(size=exit_size)
                    self.executed_take_profits.append(tp)
                    
                    if self.max_r_multiple < tp["r_multiple"]:
                        self.max_r_multiple = tp["r_multiple"]
                    
                    # Move stop loss based on take profit level
                    if i == 1:  # Second take profit level (R=5)
                        # Move SL to first take profit level (R=3)
                        self.stop_loss = self.take_profit_levels[0]["price"]
                        logger.info(f"{self._get_log_prefix()} - Moved stop loss to TP1: {self.stop_loss:.2f}")
                    elif i == 2:  # Third take profit level (R=7)
                        # Move SL to second take profit level (R=5)
                        self.stop_loss = self.take_profit_levels[1]["price"]
                        logger.info(f"{self._get_log_prefix()} - Moved stop loss to TP2: {self.stop_loss:.2f}")
                    # For first take profit level (R=3), don't move SL
                        
                    logger.info(f"{self._get_log_prefix()} - Executed take profit {tp['r_multiple']}R - Price: {current_price:.2f}, Size: {exit_size}")
                    
        else:  # Short position
            # Check for gap up
            if self.data.open[0] >= self.stop_loss:
                logger.info(f"{self._get_log_prefix()} - Short Position Stopped Out (Gap Up) - Open: {self.data.open[0]:.2f}, SL: {self.stop_loss:.2f}")
                self.close(price=self.data.open[0])
                return
                
            # Check for stop loss hit during candle
            if self.data.high[0] >= self.stop_loss:
                # Calculate the exact price where stop was hit
                # This is an approximation - in reality, we don't know the exact price
                # We'll use the stop loss price as it's the worst case
                exit_price = self.stop_loss
                logger.info(f"{self._get_log_prefix()} - Short Position Stopped Out (Mid-Candle) - Price: {exit_price:.2f}, SL: {self.stop_loss:.2f}")
                self.close(price=exit_price)
                return
            
            if current_price < (self.entry_price - 2*(self.stop_loss - self.entry_price)):
                # Trail SL
                self.stop_loss = self.entry_price - (self.stop_loss - self.entry_price)
                logger.info(f"{self._get_log_prefix()} - Trailing stop loss to: {self.stop_loss:.2f}")
                
            # Check take profit levels
            for i, tp in enumerate(self.take_profit_levels):
                if tp in self.executed_take_profits:
                    continue
                    
                if current_price <= tp["price"]:
                    # Execute partial exit
                    exit_size = self.position_size * (tp["size_percentage"] / 100)
                    self.buy(size=exit_size)
                    self.executed_take_profits.append(tp)
                    
                    if self.max_r_multiple < tp["r_multiple"]:
                        self.max_r_multiple = tp["r_multiple"]
                    
                    # Move stop loss based on take profit level
                    if i == 1:  # Second take profit level (R=5)
                        # Move SL to first take profit level (R=3)
                        self.stop_loss = self.take_profit_levels[0]["price"]
                        logger.info(f"{self._get_log_prefix()} - Moved stop loss to TP1: {self.stop_loss:.2f}")
                    elif i == 2:  # Third take profit level (R=7)
                        # Move SL to second take profit level (R=5)
                        self.stop_loss = self.take_profit_levels[1]["price"]
                        logger.info(f"{self._get_log_prefix()} - Moved stop loss to TP2: {self.stop_loss:.2f}")
                    # For first take profit level (R=3), don't move SL
                        
                    logger.info(f"{self._get_log_prefix()} - Executed take profit {tp['r_multiple']}R - Price: {current_price:.2f}, Size: {exit_size}")

        # Check market close
        if current_time_ist.time() >= self.p.market_close_time:
            logger.info(f"{self._get_log_prefix()} - Position Closed at Market Close - Price: {current_price:.2f}")
            self.close()
            
    def notify_order(self, order):
        """Handle order notifications."""
        if order.status in [order.Submitted, order.Accepted]:
            return
            
        if order.status in [order.Completed]:
            if order.isbuy():
                logger.info(f"{self._get_log_prefix()} - Buy Executed - Price: {order.executed.price:.2f}, Cost: {order.executed.value:.2f}, Comm: {order.executed.comm:.2f}, Size: {order.executed.size}")
            else:
                logger.info(f"{self._get_log_prefix()} - Sell Executed - Price: {order.executed.price:.2f}, Cost: {order.executed.value:.2f}, Comm: {order.executed.comm:.2f}, Size: {order.executed.size}")
                
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
        
        if self.entry_price > self.stop_loss_original:
            direction = "LONG"
        else:
            direction = "SHORT"
            
        # Create trade record for closed trade
        trade_record = TradeRecord(
            date=self.data.datetime.date(0).strftime('%Y-%m-%d'),
            name=self.data._name,
            pnl=trade.pnl,
            status=TradeStatus.CLOSED.value,
            direction=direction,
            trade_type=TradeType.IMMEDIATE_BREAKOUT.value,
            max_r_multiple=self.max_r_multiple,
            entry_time=trade.dtopen,
            exit_time=trade.dtclose,
            entry_price=trade.price,
            exit_price=trade.pnl / trade.size + trade.price if trade.size != 0 else 0,
            stop_loss=self.stop_loss,
            risk_amount=abs(trade.price - self.stop_loss)
        )
        
        # Add to trade records
        self.trade_records.append(trade_record)
        self.trade_closed_today = True
        self.current_trade = None
        self.max_r_multiple = 0.0
        
        logger.info(f"{self._get_log_prefix()} - Trade Closed - PnL: {trade.pnl:.2f}, Gross: {trade.pnlcomm:.2f}, Entry: {trade.price:.2f}, Exit: {trade.pnl / trade.size + trade.price if trade.size != 0 else 0:.2f}, Stop loss: {self.stop_loss:.2f}, Size: {trade.size}")
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