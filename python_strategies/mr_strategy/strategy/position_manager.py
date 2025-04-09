"""
Extended Position Manager for Morning Range Strategy.

This module enhances the basic position manager with advanced risk management,
position tracking, and multi-position handling capabilities.
"""

from dataclasses import dataclass
from enum import Enum
from typing import Dict, Optional, Union, List, Tuple
import logging
from decimal import Decimal
import pandas as pd
from datetime import datetime, time

logger = logging.getLogger(__name__)

class PositionSizeType(Enum):
    """Types of position sizing strategies available."""
    FIXED = "fixed"
    RISK_PERCENTAGE = "risk_percentage"
    ACCOUNT_PERCENTAGE = "account_percentage"

class PositionStatus(Enum):
    """Status of a trading position."""
    PENDING = "pending"
    ACTIVE = "active"
    SCALING = "scaling"
    BREAKEVEN = "breakeven"
    TRAILING = "trailing"
    CLOSED = "closed"
    STOPPED = "stopped"

@dataclass
class AccountInfo:
    """Account information for position calculations."""
    total_capital: float
    available_capital: float
    max_position_size: float
    risk_per_trade: float  # Percentage
    max_risk_per_trade: float  # Absolute value
    currency: str = "INR"

@dataclass
class PositionSizeConfig:
    """Configuration for position sizing."""
    size_type: PositionSizeType
    value: float  # Either fixed size, risk %, or account %
    min_size: float = 1.0
    max_size: float = float('inf')
    round_to: int = 0  # Number of decimal places to round to

@dataclass
class RiskLimits:
    """Risk limits configuration."""
    max_daily_loss: float  # Maximum daily loss allowed
    max_position_loss: float  # Maximum loss per position
    max_open_positions: int  # Maximum number of concurrent positions
    max_daily_trades: int  # Maximum trades per day
    max_risk_multiplier: float  # Maximum risk multiplier for scaling
    position_size_limit: float  # Maximum position size as % of account
    max_correlated_positions: int  # Maximum correlated positions allowed

@dataclass
class PositionMetrics:
    """Metrics for position tracking."""
    max_favorable_excursion: float = 0.0  # Maximum profit reached
    max_adverse_excursion: float = 0.0  # Maximum drawdown reached
    time_in_trade: int = 0  # Duration in minutes
    scaling_count: int = 0  # Number of times position was scaled
    entry_efficiency: float = 0.0  # Entry timing efficiency
    exit_efficiency: float = 0.0  # Exit timing efficiency
    r_multiple: float = 0.0  # R-multiple achieved

class ExtendedPositionManager:
    """Enhanced Position Manager with advanced risk management."""
    
    def __init__(self, 
                 account_info: AccountInfo, 
                 position_config: PositionSizeConfig,
                 risk_limits: RiskLimits):
        """
        Initialize the Extended Position Manager.
        
        Args:
            account_info: Account information
            position_config: Position sizing configuration
            risk_limits: Risk management limits
        """
        self.account_info = account_info
        self.position_config = position_config
        self.risk_limits = risk_limits
        
        # Enhanced position tracking
        self.positions: Dict[str, Dict] = {}  # Active positions
        self.position_history: List[Dict] = []  # Historical positions
        self.daily_stats: Dict = self._init_daily_stats()
        self.correlation_matrix: Dict[str, List[str]] = {}  # Track correlated instruments
        
        logger.info("Initialized Extended Position Manager")
        logger.info(f"Risk Limits: {risk_limits}")

    def _init_daily_stats(self) -> Dict:
        """Initialize daily trading statistics."""
        return {
            "total_trades": 0,
            "winning_trades": 0,
            "losing_trades": 0,
            "daily_pnl": 0.0,
            "max_drawdown": 0.0,
            "open_positions": 0,
            "risk_exposure": 0.0,
            "start_balance": self.account_info.total_capital,
            "current_balance": self.account_info.total_capital,
            "trades": []
        }

    def can_take_new_position(self, instrument_key: str, position_type: str) -> Tuple[bool, str]:
        """
        Check if a new position can be taken based on risk limits.
        
        Args:
            instrument_key: Instrument identifier
            position_type: Type of position (LONG/SHORT)
            
        Returns:
            Tuple[bool, str]: (Can take position, Reason if not)
        """
        # Check daily loss limit
        if abs(self.daily_stats["daily_pnl"]) >= self.risk_limits.max_daily_loss:
            return False, "Daily loss limit reached"
        
        # Check maximum trades per day
        if self.daily_stats["total_trades"] >= self.risk_limits.max_daily_trades:
            return False, "Daily trade limit reached"
        
        # Check open positions limit
        if len(self.positions) >= self.risk_limits.max_open_positions:
            return False, "Maximum open positions reached"
        
        # Check correlated positions
        if self._check_correlation_limit(instrument_key):
            return False, "Maximum correlated positions reached"
        
        # Check risk exposure
        if not self._check_risk_exposure():
            return False, "Risk exposure limit reached"
        
        return True, "Position allowed"

    def _check_correlation_limit(self, instrument_key: str) -> bool:
        """Check if adding a position would exceed correlation limits."""
        if instrument_key in self.correlation_matrix:
            correlated_instruments = self.correlation_matrix[instrument_key]
            current_correlated = sum(1 for inst in correlated_instruments 
                                   if inst in self.positions)
            return current_correlated >= self.risk_limits.max_correlated_positions
        return False

    def _check_risk_exposure(self) -> bool:
        """Check if current risk exposure is within limits."""
        total_risk = sum(pos["risk_amount"] for pos in self.positions.values())
        max_risk = self.account_info.total_capital * (self.risk_limits.position_size_limit / 100)
        return total_risk < max_risk

    def calculate_position_metrics(self, position: Dict) -> PositionMetrics:
        """Calculate performance metrics for a position."""
        metrics = PositionMetrics()
        
        entry_price = position["entry_price"]
        current_price = position["current_price"]
        stop_loss = position["stop_loss"]
        position_type = position["position_type"]
        
        # Calculate R-multiple
        risk_per_share = abs(entry_price - stop_loss)
        if risk_per_share > 0:
            if position_type == "LONG":
                profit_per_share = current_price - entry_price
            else:
                profit_per_share = entry_price - current_price
            metrics.r_multiple = profit_per_share / risk_per_share
        
        # Calculate time in trade
        if "entry_time" in position:
            metrics.time_in_trade = int((datetime.now() - position["entry_time"]).total_seconds() / 60)
        
        # Update MAE/MFE
        metrics.max_adverse_excursion = position.get("max_adverse_excursion", 0.0)
        metrics.max_favorable_excursion = position.get("max_favorable_excursion", 0.0)
        
        return metrics

    def scale_position(self, 
                      instrument_key: str, 
                      scale_percentage: float,
                      new_sl_percentage: Optional[float] = None) -> Dict:
        """
        Scale into or out of a position.
        
        Args:
            instrument_key: Instrument identifier
            scale_percentage: Percentage to scale (positive for scaling in, negative for out)
            new_sl_percentage: New stop loss percentage (optional)
            
        Returns:
            Dict: Updated position information
        """
        if instrument_key not in self.positions:
            logger.warning(f"No position found for {instrument_key}")
            return {}
        
        position = self.positions[instrument_key]
        current_size = position["size"]
        
        # Calculate new position size
        scale_factor = 1 + (scale_percentage / 100)
        new_size = current_size * scale_factor
        
        # Validate new size
        if not self.validate_position_size(new_size, position["current_price"]):
            logger.warning(f"Invalid scaled size {new_size}")
            return position
        
        # Update position
        position["size"] = new_size
        position["scaling_count"] += 1
        
        # Update stop loss if provided
        if new_sl_percentage is not None:
            position["sl_percentage"] = new_sl_percentage
            position["stop_loss"] = self.calculate_stop_loss_price(
                position["current_price"],
                new_sl_percentage,
                position["position_type"]
            )
        
        # Recalculate risk and metrics
        position["risk_amount"] = abs(position["current_price"] - position["stop_loss"]) * new_size
        position["metrics"] = self.calculate_position_metrics(position)
        
        logger.info(f"Scaled position for {instrument_key}: {position}")
        return position

    def move_to_breakeven(self, instrument_key: str) -> Dict:
        """Move stop loss to breakeven level."""
        if instrument_key not in self.positions:
            logger.warning(f"No position found for {instrument_key}")
            return {}
        
        position = self.positions[instrument_key]
        position["stop_loss"] = position["entry_price"]
        position["status"] = PositionStatus.BREAKEVEN.value
        
        logger.info(f"Moved position to breakeven for {instrument_key}")
        return position

    def update_trailing_stop(self, 
                           instrument_key: str, 
                           trail_percentage: float) -> Dict:
        """
        Update trailing stop loss.
        
        Args:
            instrument_key: Instrument identifier
            trail_percentage: Trailing stop percentage
            
        Returns:
            Dict: Updated position information
        """
        if instrument_key not in self.positions:
            logger.warning(f"No position found for {instrument_key}")
            return {}
        
        position = self.positions[instrument_key]
        current_price = position["current_price"]
        position_type = position["position_type"]
        
        # Calculate new stop loss based on trailing percentage
        if position_type == "LONG":
            new_stop_loss = current_price * (1 - trail_percentage/100)
            # Only update if new stop loss is higher
            if new_stop_loss > position["stop_loss"]:
                position["stop_loss"] = new_stop_loss
        else:  # SHORT
            new_stop_loss = current_price * (1 + trail_percentage/100)
            # Only update if new stop loss is lower
            if new_stop_loss < position["stop_loss"]:
                position["stop_loss"] = new_stop_loss
        
        position["status"] = PositionStatus.TRAILING.value
        logger.info(f"Updated trailing stop for {instrument_key}: {position}")
        return position

    def update_daily_stats(self, trade_result: Dict):
        """Update daily trading statistics."""
        self.daily_stats["total_trades"] += 1
        self.daily_stats["daily_pnl"] += trade_result["realized_pnl"]
        
        if trade_result["realized_pnl"] > 0:
            self.daily_stats["winning_trades"] += 1
        else:
            self.daily_stats["losing_trades"] += 1
        
        # Update maximum drawdown
        current_drawdown = min(0, self.daily_stats["daily_pnl"])
        self.daily_stats["max_drawdown"] = min(
            self.daily_stats["max_drawdown"],
            current_drawdown
        )
        
        # Update current balance
        self.daily_stats["current_balance"] = (
            self.daily_stats["start_balance"] + 
            self.daily_stats["daily_pnl"]
        )
        
        # Add trade to history
        self.daily_stats["trades"].append(trade_result)
        
        logger.info(f"Updated daily stats: {self.daily_stats}")

    def get_risk_metrics(self) -> Dict:
        """Get current risk metrics."""
        return {
            "total_risk_exposure": sum(pos["risk_amount"] for pos in self.positions.values()),
            "risk_per_position": {k: v["risk_amount"] for k, v in self.positions.items()},
            "account_risk_percentage": (sum(pos["risk_amount"] for pos in self.positions.values()) / 
                                      self.account_info.total_capital * 100),
            "daily_loss_percentage": (self.daily_stats["daily_pnl"] / 
                                    self.daily_stats["start_balance"] * 100),
            "max_drawdown_percentage": (self.daily_stats["max_drawdown"] / 
                                      self.daily_stats["start_balance"] * 100),
            "win_rate": (self.daily_stats["winning_trades"] / 
                        max(1, self.daily_stats["total_trades"]) * 100)
        }

    def get_position_performance(self, instrument_key: str) -> Dict:
        """Get detailed performance metrics for a position."""
        if instrument_key not in self.positions:
            logger.warning(f"No position found for {instrument_key}")
            return {}
        
        position = self.positions[instrument_key]
        metrics = self.calculate_position_metrics(position)
        
        return {
            "instrument_key": instrument_key,
            "position_type": position["position_type"],
            "entry_price": position["entry_price"],
            "current_price": position["current_price"],
            "size": position["size"],
            "unrealized_pnl": position["unrealized_pnl"],
            "risk_amount": position["risk_amount"],
            "r_multiple": metrics.r_multiple,
            "time_in_trade": metrics.time_in_trade,
            "max_favorable_excursion": metrics.max_favorable_excursion,
            "max_adverse_excursion": metrics.max_adverse_excursion,
            "scaling_count": metrics.scaling_count,
            "status": position.get("status", PositionStatus.ACTIVE.value)
        }

    def validate_position_size(self, size: float, price: float) -> bool:
        """
        Validate if a position size is within acceptable limits.
        
        Args:
            size: Position size to validate
            price: Current price of the instrument
            
        Returns:
            bool: True if position size is valid
        """
        position_value = size * price
        
        # Check minimum size
        if size < self.position_config.min_size:
            logger.warning(f"Position size {size} below minimum {self.position_config.min_size}")
            return False
        
        # Check maximum size
        if size > self.position_config.max_size:
            logger.warning(f"Position size {size} above maximum {self.position_config.max_size}")
            return False
        
        # Check against account limits
        if position_value > self.account_info.available_capital:
            logger.warning(f"Position value {position_value} exceeds available capital {self.account_info.available_capital}")
            return False
        
        # Check against max position size
        if position_value > self.account_info.max_position_size:
            logger.warning(f"Position value {position_value} exceeds max position size {self.account_info.max_position_size}")
            return False
        
        return True

    def calculate_fixed_size(self, price: float) -> float:
        """Calculate position size based on fixed quantity."""
        size = self.position_config.value
        if self.validate_position_size(size, price):
            return round(size, self.position_config.round_to)
        return 0.0

    def calculate_stop_loss_price(self, entry_price: float, sl_percentage: float, position_type: str) -> float:
        """
        Calculate stop loss price based on percentage and position type.
        
        Args:
            entry_price: Entry price of the position
            sl_percentage: Stop loss percentage (e.g., 0.5 for 0.5%)
            position_type: Type of position ("LONG" or "SHORT")
            
        Returns:
            float: Calculated stop loss price
        """
        sl_decimal = sl_percentage / 100  # Convert percentage to decimal
        
        if position_type == "LONG":
            sl_price = entry_price - (entry_price * sl_decimal)
        else:  # SHORT
            sl_price = entry_price + (entry_price * sl_decimal)
            
        logger.info(f"Calculated SL price: {sl_price} for {position_type} position at entry {entry_price} with {sl_percentage}% SL")
        return round(sl_price, 2)

    def calculate_risk_based_size(self, price: float, sl_percentage: float, position_type: str) -> float:
        """
        Calculate position size based on risk percentage.
        
        Args:
            price: Entry price
            sl_percentage: Stop loss percentage
            position_type: Type of position ("LONG" or "SHORT")
            
        Returns:
            float: Position size based on risk
        """
        # Calculate stop loss price
        stop_loss = self.calculate_stop_loss_price(price, sl_percentage, position_type)
        
        # Calculate risk per share
        risk_per_share = abs(price - stop_loss)
        if risk_per_share == 0:
            logger.warning("Risk per share is zero")
            return 0.0
            
        # Calculate risk amount based on account risk percentage
        risk_amount = self.account_info.total_capital * (self.position_config.value / 100)
        
        # Calculate position size
        size = risk_amount / risk_per_share
        
        # Round to specified decimal places
        size = round(size, self.position_config.round_to)
        
        if self.validate_position_size(size, price):
            logger.info(f"Calculated risk-based size: {size} shares at price {price} with {sl_percentage}% SL")
            return size
        return 0.0

    def calculate_account_based_size(self, price: float) -> float:
        """Calculate position size based on account percentage."""
        position_value = self.account_info.total_capital * (self.position_config.value / 100)
        size = position_value / price
        
        # Round to specified decimal places
        size = round(size, self.position_config.round_to)
        
        if self.validate_position_size(size, price):
            return size
        return 0.0

    def calculate_position_size(self, price: float, sl_percentage: Optional[float] = None, position_type: str = "LONG") -> float:
        """
        Calculate position size based on configured sizing strategy.
        
        Args:
            price: Current price of the instrument
            sl_percentage: Stop loss percentage (required for risk-based sizing)
            position_type: Type of position ("LONG" or "SHORT")
            
        Returns:
            float: Calculated position size
        """
        logger.info(f"Calculating position size for price {price} using {self.position_config.size_type.value} strategy")
        
        if self.position_config.size_type == PositionSizeType.FIXED:
            size = self.calculate_fixed_size(price)
            
        elif self.position_config.size_type == PositionSizeType.RISK_PERCENTAGE:
            if sl_percentage is None:
                logger.error("Stop loss percentage required for risk-based position sizing")
                return 0.0
            size = self.calculate_risk_based_size(price, sl_percentage, position_type)
            
        elif self.position_config.size_type == PositionSizeType.ACCOUNT_PERCENTAGE:
            size = self.calculate_account_based_size(price)
            
        else:
            logger.error(f"Unknown position size type: {self.position_config.size_type}")
            return 0.0
        
        logger.info(f"Calculated position size: {size}")
        return size

    def add_position(self, 
                    instrument_key: str, 
                    size: float, 
                    entry_price: float,
                    sl_percentage: float,
                    position_type: str = "LONG") -> Dict:
        """
        Add a new position to tracking.
        
        Args:
            instrument_key: Unique identifier for the instrument
            size: Position size
            entry_price: Entry price
            sl_percentage: Stop loss percentage
            position_type: Type of position ("LONG" or "SHORT")
            
        Returns:
            Dict: Position information
        """
        if instrument_key in self.positions:
            logger.warning(f"Position already exists for {instrument_key}")
            return {}
            
        # Calculate stop loss price
        stop_loss = self.calculate_stop_loss_price(entry_price, sl_percentage, position_type)
            
        position = {
            "instrument_key": instrument_key,
            "size": size,
            "entry_price": entry_price,
            "current_price": entry_price,
            "stop_loss": stop_loss,
            "sl_percentage": sl_percentage,
            "position_type": position_type,
            "unrealized_pnl": 0.0,
            "risk_amount": abs(entry_price - stop_loss) * size
        }
        
        self.positions[instrument_key] = position
        logger.info(f"Added new position for {instrument_key}: {position}")
        
        return position

    def update_position(self, 
                       instrument_key: str, 
                       current_price: float,
                       new_sl_percentage: Optional[float] = None) -> Dict:
        """
        Update an existing position with new price and optional stop loss.
        
        Args:
            instrument_key: Unique identifier for the instrument
            current_price: Current market price
            new_sl_percentage: New stop loss percentage (optional)
            
        Returns:
            Dict: Updated position information
        """
        if instrument_key not in self.positions:
            logger.warning(f"No position found for {instrument_key}")
            return {}
            
        position = self.positions[instrument_key]
        position["current_price"] = current_price
        
        # Update stop loss if provided
        if new_sl_percentage is not None:
            position["sl_percentage"] = new_sl_percentage
            position["stop_loss"] = self.calculate_stop_loss_price(
                current_price, 
                new_sl_percentage, 
                position["position_type"]
            )
            position["risk_amount"] = abs(current_price - position["stop_loss"]) * position["size"]
        
        # Calculate unrealized P&L
        if position["position_type"] == "LONG":
            position["unrealized_pnl"] = (current_price - position["entry_price"]) * position["size"]
        else:  # SHORT
            position["unrealized_pnl"] = (position["entry_price"] - current_price) * position["size"]
        
        logger.info(f"Updated position for {instrument_key}: {position}")
        return position

    def close_position(self, instrument_key: str, exit_price: float) -> Dict:
        """
        Close an existing position and calculate final P&L.
        
        Args:
            instrument_key: Unique identifier for the instrument
            exit_price: Exit price
            
        Returns:
            Dict: Closed position information with realized P&L
        """
        if instrument_key not in self.positions:
            logger.warning(f"No position found for {instrument_key}")
            return {}
            
        position = self.positions.pop(instrument_key)
        
        # Calculate realized P&L
        if position["position_type"] == "LONG":
            realized_pnl = (exit_price - position["entry_price"]) * position["size"]
        else:  # SHORT
            realized_pnl = (position["entry_price"] - exit_price) * position["size"]
        
        position["exit_price"] = exit_price
        position["realized_pnl"] = realized_pnl
        
        logger.info(f"Closed position for {instrument_key} with P&L: {realized_pnl}")
        return position

    def get_position_summary(self) -> Dict:
        """
        Get summary of all current positions.
        
        Returns:
            Dict: Summary of all positions
        """
        total_positions = len(self.positions)
        total_exposure = sum(pos["size"] * pos["current_price"] for pos in self.positions.values())
        total_unrealized_pnl = sum(pos["unrealized_pnl"] for pos in self.positions.values())
        total_risk = sum(pos["risk_amount"] for pos in self.positions.values())
        
        summary = {
            "total_positions": total_positions,
            "total_exposure": total_exposure,
            "total_unrealized_pnl": total_unrealized_pnl,
            "total_risk": total_risk,
            "available_capital": self.account_info.available_capital,
            "positions": self.positions
        }
        
        logger.info(f"Position summary: {summary}")
        return summary 