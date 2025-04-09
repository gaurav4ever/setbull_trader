"""
Extended Trade Manager for Morning Range Strategy.

This module enhances the basic trade manager with advanced features like
multiple take profit levels, dynamic stop loss, breakeven, and trailing stops.
"""

from dataclasses import dataclass
from enum import Enum
from typing import Dict, Optional, Union, List, Tuple
import logging
from datetime import datetime, time
import numpy as np

logger = logging.getLogger(__name__)

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
class TakeProfitLevel:
    """Configuration for a take profit level."""
    r_multiple: float  # Target in R multiples
    size_percentage: float  # Percentage of position to close
    trail_activation: bool = False  # Whether to activate trailing stop
    move_sl_to_be: bool = False  # Whether to move stop loss to breakeven

@dataclass
class ExtendedTradeConfig:
    """Extended configuration for trade management."""
    sl_percentage: float  # Stop loss percentage
    take_profit_levels: List[TakeProfitLevel]  # Multiple TP levels
    breakeven_r: float  # R multiple to move to breakeven
    trail_activation_r: float  # R multiple to activate trailing stop
    trail_step_percentage: float  # Trailing stop step size
    partial_exit_adjustment: bool  # Whether to adjust stops after partial exits
    max_trade_duration: int  # Maximum trade duration in minutes
    entry_timeout: int  # Entry timeout in minutes
    reentry_times: int  # Number of allowed re-entries
    min_risk_reward: float  # Minimum risk-reward ratio
    dynamic_sl_adjustment: bool  # Whether to enable dynamic SL adjustment

@dataclass
class TradeMetrics:
    """Metrics for trade tracking."""
    entry_price: float
    current_price: float
    stop_loss: float
    take_profit: float
    position_size: float
    unrealized_pnl: float = 0.0
    realized_pnl: float = 0.0
    risk_amount: float = 0.0
    reward_amount: float = 0.0
    risk_reward_ratio: float = 0.0
    r_multiple: float = 0.0
    duration: int = 0  # minutes
    max_favorable_excursion: float = 0.0
    max_adverse_excursion: float = 0.0

class ExtendedTradeManager:
    """Enhanced manager for advanced trade management."""
    
    def __init__(self, trade_config: ExtendedTradeConfig):
        """
        Initialize the Extended Trade Manager.
        
        Args:
            trade_config: Extended trade management configuration
        """
        self.config = trade_config
        self.active_trades: Dict[str, Dict] = {}
        self.trade_history: List[Dict] = []
        
        logger.info(f"Initialized ExtendedTradeManager with {len(trade_config.take_profit_levels)} TP levels")

    def calculate_trade_levels(self, 
                             entry_price: float, 
                             position_type: str,
                             sl_percentage: Optional[float] = None) -> Dict[str, float]:
        """Calculate multiple trade levels including all take profit targets."""
        sl_pct = sl_percentage if sl_percentage is not None else self.config.sl_percentage
        
        # Calculate base levels
        if position_type == "LONG":
            stop_loss = entry_price * (1 - sl_pct/100)
            risk_amount = entry_price - stop_loss
            breakeven_level = entry_price + (risk_amount * self.config.breakeven_r)
            
            # Calculate multiple take profit levels
            take_profit_levels = [
                {
                    "price": entry_price + (risk_amount * tp.r_multiple),
                    "size_percentage": tp.size_percentage,
                    "r_multiple": tp.r_multiple,
                    "trail_activation": tp.trail_activation,
                    "move_sl_to_be": tp.move_sl_to_be
                }
                for tp in self.config.take_profit_levels
            ]
        else:  # SHORT
            stop_loss = entry_price * (1 + sl_pct/100)
            risk_amount = stop_loss - entry_price
            breakeven_level = entry_price - (risk_amount * self.config.breakeven_r)
            
            # Calculate multiple take profit levels
            take_profit_levels = [
                {
                    "price": entry_price - (risk_amount * tp.r_multiple),
                    "size_percentage": tp.size_percentage,
                    "r_multiple": tp.r_multiple,
                    "trail_activation": tp.trail_activation,
                    "move_sl_to_be": tp.move_sl_to_be
                }
                for tp in self.config.take_profit_levels
            ]
        
        levels = {
            "entry_price": entry_price,
            "stop_loss": round(stop_loss, 2),
            "breakeven_level": round(breakeven_level, 2),
            "risk_amount": round(risk_amount, 2),
            "take_profit_levels": take_profit_levels,
            "initial_position_size": None  # To be set when creating trade
        }
        
        logger.info(f"Calculated trade levels: {levels}")
        return levels

    def validate_trade_setup(self, 
                           entry_price: float,
                           current_price: float,
                           position_type: str,
                           levels: Dict[str, float]) -> Tuple[bool, str]:
        """
        Validate trade setup before entry.
        
        Args:
            entry_price: Planned entry price
            current_price: Current market price
            position_type: Type of position
            levels: Calculated trade levels
            
        Returns:
            Tuple[bool, str]: (Is valid, Reason if not valid)
        """
        # Check if risk-reward meets minimum requirement
        if levels["risk_reward_ratio"] < self.config.min_risk_reward:
            return False, f"Risk-reward {levels['risk_reward_ratio']} below minimum {self.config.min_risk_reward}"
        
        # Validate entry price slippage
        max_slippage = levels["risk_amount"] * 0.1  # 10% of risk
        price_diff = abs(current_price - entry_price)
        if price_diff > max_slippage:
            return False, f"Price slippage {price_diff} exceeds maximum {max_slippage}"
        
        # Validate stop loss distance
        min_stop_distance = entry_price * 0.001  # Minimum 0.1% stop distance
        if levels["risk_amount"] < min_stop_distance:
            return False, f"Stop distance {levels['risk_amount']} below minimum {min_stop_distance}"
        
        return True, "Trade setup valid"

    def create_extended_trade(self,
                            instrument_key: str,
                            entry_price: float,
                            position_size: float,
                            position_type: str,
                            trade_type: TradeType,
                            sl_percentage: Optional[float] = None) -> Dict:
        """Create a new trade with extended features."""
        # Calculate trade levels
        levels = self.calculate_trade_levels(entry_price, position_type, sl_percentage)
        levels["initial_position_size"] = position_size
        
        # Create trade object with extended features
        trade = {
            "instrument_key": instrument_key,
            "entry_price": entry_price,
            "current_price": entry_price,
            "initial_position_size": position_size,
            "current_position_size": position_size,
            "position_type": position_type,
            "trade_type": trade_type.value,
            "status": TradeStatus.ACTIVE.value,
            "entry_time": datetime.now(),
            "stop_loss": levels["stop_loss"],
            "breakeven_level": levels["breakeven_level"],
            "risk_amount": levels["risk_amount"],
            "take_profit_levels": levels["take_profit_levels"],
            "executed_take_profits": [],
            "unrealized_pnl": 0.0,
            "realized_pnl": 0.0,
            "max_favorable_excursion": 0.0,
            "max_adverse_excursion": 0.0,
            "trailing_stop_active": False,
            "trailing_stop_level": None,
            "re_entries": 0,
            "partial_exits": []
        }
        
        self.active_trades[instrument_key] = trade
        logger.info(f"Created new extended trade: {trade}")
        return trade

    def update_trailing_stop(self, trade: Dict, current_price: float) -> float:
        """Update trailing stop level based on price movement."""
        if not trade["trailing_stop_active"]:
            return trade["stop_loss"]
        
        trail_step = current_price * (self.config.trail_step_percentage / 100)
        
        if trade["position_type"] == "LONG":
            new_stop = current_price - trail_step
            if new_stop > trade["stop_loss"]:
                return round(new_stop, 2)
        else:  # SHORT
            new_stop = current_price + trail_step
            if new_stop < trade["stop_loss"]:
                return round(new_stop, 2)
        
        return trade["stop_loss"]

    def check_take_profit_levels(self, trade: Dict, current_price: float) -> Optional[Dict]:
        """Check if any take profit level is hit."""
        for tp_level in trade["take_profit_levels"]:
            if tp_level in trade["executed_take_profits"]:
                continue
                
            price_hit = False
            if trade["position_type"] == "LONG":
                price_hit = current_price >= tp_level["price"]
            else:  # SHORT
                price_hit = current_price <= tp_level["price"]
                
            if price_hit:
                return tp_level
        
        return None

    def execute_partial_exit(self, 
                           trade: Dict, 
                           tp_level: Dict,
                           current_price: float) -> Dict:
        """Execute a partial position exit at take profit level."""
        exit_size = trade["initial_position_size"] * (tp_level["size_percentage"] / 100)
        remaining_size = trade["current_position_size"] - exit_size
        
        # Calculate realized P&L for this partial exit
        if trade["position_type"] == "LONG":
            realized_pnl = (current_price - trade["entry_price"]) * exit_size
        else:  # SHORT
            realized_pnl = (trade["entry_price"] - current_price) * exit_size
        
        partial_exit = {
            "exit_price": current_price,
            "exit_size": exit_size,
            "realized_pnl": realized_pnl,
            "r_multiple": tp_level["r_multiple"],
            "exit_time": datetime.now()
        }
        
        # Update trade
        trade["current_position_size"] = remaining_size
        trade["realized_pnl"] += realized_pnl
        trade["executed_take_profits"].append(tp_level)
        trade["partial_exits"].append(partial_exit)
        
        # Check if we should move to breakeven
        if tp_level["move_sl_to_be"]:
            trade["stop_loss"] = trade["entry_price"]
            trade["status"] = TradeStatus.BREAKEVEN.value
        
        # Check if we should activate trailing stop
        if tp_level["trail_activation"]:
            trade["trailing_stop_active"] = True
            trade["status"] = TradeStatus.TRAILING.value
        
        logger.info(f"Executed partial exit: {partial_exit}")
        return trade

    def update_extended_trade(self, 
                            instrument_key: str, 
                            current_price: float) -> Dict:
        """Update trade with extended features."""
        if instrument_key not in self.active_trades:
            logger.warning(f"No active trade found for {instrument_key}")
            return {}
        
        trade = self.active_trades[instrument_key]
        trade["current_price"] = current_price
        
        # Update unrealized P&L
        if trade["position_type"] == "LONG":
            trade["unrealized_pnl"] = (current_price - trade["entry_price"]) * trade["current_position_size"]
        else:  # SHORT
            trade["unrealized_pnl"] = (trade["entry_price"] - current_price) * trade["current_position_size"]
        
        # Update MAE/MFE
        trade["max_favorable_excursion"] = max(
            trade["max_favorable_excursion"],
            trade["unrealized_pnl"]
        )
        trade["max_adverse_excursion"] = min(
            trade["max_adverse_excursion"],
            trade["unrealized_pnl"]
        )
        
        # Update trailing stop if active
        if trade["trailing_stop_active"]:
            new_stop = self.update_trailing_stop(trade, current_price)
            if new_stop != trade["stop_loss"]:
                trade["stop_loss"] = new_stop
                logger.info(f"Updated trailing stop to {new_stop}")
        
        # Check take profit levels
        tp_level = self.check_take_profit_levels(trade, current_price)
        if tp_level is not None:
            trade = self.execute_partial_exit(trade, tp_level, current_price)
            
            # Check if position is fully closed
            if trade["current_position_size"] == 0:
                return self.close_extended_trade(
                    instrument_key,
                    current_price,
                    TradeStatus.TAKE_PROFIT
                )
        
        # Check stop loss
        if self.check_stop_loss(trade, current_price):
            return self.close_extended_trade(
                instrument_key,
                current_price,
                TradeStatus.STOPPED_OUT
            )
        
        # Check trade duration
        trade_duration = (datetime.now() - trade["entry_time"]).total_seconds() / 60
        if trade_duration > self.config.max_trade_duration:
            return self.close_extended_trade(
                instrument_key,
                current_price,
                TradeStatus.EXPIRED
            )
        
        return trade

    def close_extended_trade(self,
                           instrument_key: str,
                           exit_price: float,
                           status: TradeStatus) -> Dict:
        """Close trade with extended metrics."""
        if instrument_key not in self.active_trades:
            logger.warning(f"No active trade found for {instrument_key}")
            return {}
        
        trade = self.active_trades.pop(instrument_key)
        
        # Calculate final realized P&L including any remaining position
        if trade["current_position_size"] > 0:
            if trade["position_type"] == "LONG":
                final_pnl = (exit_price - trade["entry_price"]) * trade["current_position_size"]
            else:  # SHORT
                final_pnl = (trade["entry_price"] - exit_price) * trade["current_position_size"]
            
            trade["realized_pnl"] += final_pnl
        
        trade["exit_price"] = exit_price
        trade["exit_time"] = datetime.now()
        trade["status"] = status.value
        trade["duration"] = (trade["exit_time"] - trade["entry_time"]).total_seconds() / 60
        
        # Calculate overall R-multiple
        if trade["risk_amount"] > 0:
            trade["r_multiple"] = trade["realized_pnl"] / (trade["risk_amount"] * trade["initial_position_size"])
        
        # Add to trade history
        self.trade_history.append(trade)
        
        logger.info(f"Closed extended trade: {trade}")
        return trade

    def get_extended_trade_metrics(self, instrument_key: str) -> Dict:
        """Get detailed metrics for an extended trade."""
        if instrument_key not in self.active_trades:
            logger.warning(f"No active trade found for {instrument_key}")
            return None
        
        trade = self.active_trades[instrument_key]
        
        metrics = {
            "entry_price": trade["entry_price"],
            "current_price": trade["current_price"],
            "initial_position_size": trade["initial_position_size"],
            "current_position_size": trade["current_position_size"],
            "stop_loss": trade["stop_loss"],
            "unrealized_pnl": trade["unrealized_pnl"],
            "realized_pnl": trade["realized_pnl"],
            "total_pnl": trade["unrealized_pnl"] + trade["realized_pnl"],
            "risk_amount": trade["risk_amount"],
            "max_favorable_excursion": trade["max_favorable_excursion"],
            "max_adverse_excursion": trade["max_adverse_excursion"],
            "duration": (datetime.now() - trade["entry_time"]).total_seconds() / 60,
            "status": trade["status"],
            "trailing_active": trade["trailing_stop_active"],
            "partial_exits": len(trade["partial_exits"]),
            "remaining_tp_levels": len(trade["take_profit_levels"]) - len(trade["executed_take_profits"])
        }
        
        logger.debug(f"Trade metrics: {metrics}")
        return metrics

    def check_stop_loss(self, trade: Dict, current_price: float) -> bool:
        """Check if stop loss is hit."""
        if trade["position_type"] == "LONG":
            return current_price <= trade["stop_loss"]
        else:  # SHORT
            return current_price >= trade["stop_loss"]

    def get_trade_statistics(self) -> Dict:
        """Get overall trade statistics."""
        if not self.trade_history:
            return {
                "total_trades": 0,
                "win_rate": 0.0,
                "average_r": 0.0,
                "profit_factor": 0.0,
                "average_duration": 0.0
            }
        
        winning_trades = [t for t in self.trade_history if t["realized_pnl"] > 0]
        losing_trades = [t for t in self.trade_history if t["realized_pnl"] <= 0]
        
        total_profit = sum(t["realized_pnl"] for t in winning_trades)
        total_loss = abs(sum(t["realized_pnl"] for t in losing_trades))
        
        stats = {
            "total_trades": len(self.trade_history),
            "winning_trades": len(winning_trades),
            "losing_trades": len(losing_trades),
            "win_rate": len(winning_trades) / len(self.trade_history) * 100,
            "total_profit": total_profit,
            "total_loss": total_loss,
            "profit_factor": total_profit / total_loss if total_loss > 0 else float('inf'),
            "average_r": np.mean([t["r_multiple"] for t in self.trade_history]),
            "average_duration": np.mean([t["duration"] for t in self.trade_history])
        }
        
        logger.info(f"Trade statistics: {stats}")
        return stats 