"""
 Trade Manager for Morning Range Strategy.

This module enhances the basic trade manager with advanced features like
multiple take profit levels, dynamic stop loss, breakeven, and trailing stops.
"""

from dataclasses import dataclass
from enum import Enum
from typing import Dict, Optional, Union, List, Tuple
import logging
from datetime import datetime, time
import numpy as np
import pandas as pd

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
class TradeConfig:
    """ configuration for trade management."""
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
    initial_capital: float  # Initial capital for profit percentage calculation

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

class TradeManager:
    """Enhanced manager for advanced trade management."""
    
    def __init__(self, trade_config: TradeConfig):
        """
        Initialize the  Trade Manager.
        
        Args:
            trade_config:  trade management configuration
        """
        self.config = trade_config
        self.active_trades: Dict[str, Dict] = {}
        self.trade_history: List[Dict] = []
        self.trade_executed_counter = 0
        self.winning_trades = 0
        self.losing_trades = 0
        logger.info(f"Initialized TradeManager with {len(trade_config.take_profit_levels)} TP levels")

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

    def calculate_trade_levels(self, 
                             entry_price: float, 
                             position_type: str,
                             sl_percentage: Optional[float] = None,
                             candle_data: Optional[Dict] = None) -> Dict[str, float]:
        """Calculate multiple trade levels including all take profit targets."""
        candle_info = self._format_candle_info(candle_data) if candle_data else ""
        
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
        
        logger.info(f"{candle_info}Calculated trade levels: {levels}")
        return levels

    def validate_trade_setup(self, 
                           entry_price: float,
                           current_price: float,
                           position_type: str,
                           levels: Dict[str, float],
                           candle_data: Optional[Dict] = None) -> Tuple[bool, str]:
        """
        Validate trade setup before entry.
        
        Args:
            entry_price: Planned entry price
            current_price: Current market price
            position_type: Type of position
            levels: Calculated trade levels
            candle_data: Current candle data (time, open, high, low, close)
            
        Returns:
            Tuple[bool, str]: (Is valid, Reason if not valid)
        """
        candle_info = self._format_candle_info(candle_data) if candle_data else ""
        
        # Check if risk-reward meets minimum requirement
        if levels["risk_reward_ratio"] < self.config.min_risk_reward:
            logger.info(f"{candle_info}Risk-reward {levels['risk_reward_ratio']} below minimum {self.config.min_risk_reward}")
            return False, f"Risk-reward {levels['risk_reward_ratio']} below minimum {self.config.min_risk_reward}"
        
        # Validate entry price slippage
        max_slippage = levels["risk_amount"] * 0.1  # 10% of risk
        price_diff = abs(current_price - entry_price)
        if price_diff > max_slippage:
            logger.info(f"{candle_info}Price slippage {price_diff} exceeds maximum {max_slippage}")
            return False, f"Price slippage {price_diff} exceeds maximum {max_slippage}"
        
        # Validate stop loss distance
        min_stop_distance = entry_price * 0.001  # Minimum 0.1% stop distance
        if levels["risk_amount"] < min_stop_distance:
            logger.info(f"{candle_info}Stop distance {levels['risk_amount']} below minimum {min_stop_distance}")
            return False, f"Stop distance {levels['risk_amount']} below minimum {min_stop_distance}"
        
        logger.info(f"{candle_info}Trade setup valid for entry price {entry_price}")
        return True, "Trade setup valid"

    def create_trade(self,
                   instrument_key: str,
                   entry_price: float,
                   position_size: float,
                   position_type: str,
                   trade_type: TradeType,
                   sl_percentage: Optional[float] = None,
                   candle_data: Optional[Dict] = None) -> Dict:
        """Create a new trade with features."""
        candle_info = self._format_candle_info(candle_data) if candle_data else ""

        # add a trade executed counter
        self.trade_executed_counter += 1

        # Calculate trade levels
        levels = self.calculate_trade_levels(entry_price, position_type, sl_percentage, candle_data)
        levels["initial_position_size"] = position_size
        
        # Create trade object with features
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
            "partial_exits": [],
            "entry_candle": candle_data
        }
        
        self.active_trades[instrument_key] = trade
        logger.info(f"{candle_info}Created new trade: {trade}")
        return trade

    def update_trailing_stop(self, trade: Dict, current_price: float, candle_data: Optional[Dict] = None) -> float:
        """
        Update trailing stop level based on take profit levels hit.
        If no TP hit, use breakeven.
        If TP1 hit, use breakeven.
        If TP2 hit, use TP1 level.
        If TP3 hit, use TP2 level.
        """
        candle_info = self._format_candle_info(candle_data) if candle_data else ""
        
        # Get executed take profits
        executed_tps = trade.get("executed_take_profits", [])
    
        if not executed_tps:
            # No TP hit, use breakeven
            return trade["entry_price"]
        
        # Get the highest TP level hit
        highest_tp = max(executed_tps, key=lambda x: x["r_multiple"])
        
        if highest_tp["r_multiple"] >= 7.0:  # TP3 hit
            # Find TP2 level
            tp2 = next((tp for tp in trade["take_profit_levels"] 
                       if tp["r_multiple"] == 5.0), None)
            if tp2:
                logger.info(f"{candle_info}TP3 hit, moving stop to TP2 level: {tp2['price']}")
                return tp2["price"]
            return trade["entry_price"]
        
        elif highest_tp["r_multiple"] >= 5.0:  # TP2 hit
            # Find TP1 level
            tp1 = next((tp for tp in trade["take_profit_levels"] 
                       if tp["r_multiple"] == 3.0), None)
            if tp1:
                logger.info(f"{candle_info}TP2 hit, moving stop to TP1 level: {tp1['price']}")
                return tp1["price"]
            return trade["entry_price"]
        
        else:  # TP1 hit
            logger.info(f"{candle_info}TP1 hit, moving stop to breakeven: {trade['entry_price']}")
            return trade["entry_price"]  # Use breakeven

    def check_take_profit_levels(self, trade: Dict, current_price: float, candle_data: Optional[Dict] = None) -> Optional[Dict]:
        """Check if any take profit level is hit."""
        candle_info = self._format_candle_info(candle_data) if candle_data else ""
        
        for tp_level in trade["take_profit_levels"]:
            if tp_level in trade["executed_take_profits"]:
                continue
                
            price_hit = False
            if trade["position_type"] == "LONG":
                price_hit = current_price >= tp_level["price"]
            else:  # SHORT
                price_hit = current_price <= tp_level["price"]
                
            if price_hit:
                logger.info(f"{candle_info}Take profit level detected: {tp_level['r_multiple']}R at price {tp_level['price']}")
                self.update_trailing_stop(trade, current_price, candle_data)
                return tp_level
        
        return None

    def execute_partial_exit(self, 
                           trade: Dict, 
                           tp_level: Dict,
                           current_price: float,
                           candle_data: Optional[Dict] = None) -> Dict:
        """Execute a partial position exit at take profit level."""
        candle_info = self._format_candle_info(candle_data) if candle_data else ""
        
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
            "exit_time": datetime.now(),
            "exit_candle": candle_data
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
            logger.info(f"{candle_info}Moving stop loss to breakeven after partial exit")
        
        logger.info(f"{candle_info}Executed partial exit: {partial_exit}")
        return trade

    def update_trade(self, 
                    instrument_key: str, 
                    current_price: float,
                    current_time: datetime,
                    candle_data: Dict,
                    entry_candle: Dict = None) -> Dict:
        """Update trade with features."""
        candle_info = self._format_candle_info(candle_data)
        
        if instrument_key not in self.active_trades:
            logger.warning(f"{candle_info}No active trade found for {instrument_key}")
            return {}
        
        trade = self.active_trades[instrument_key]
        trade["current_price"] = current_price
        trade["current_time"] = current_time.strftime('%Y-%m-%d %H:%M:%S')
        trade["current_candle"] = candle_data

        # Check stop loss
        if self.check_stop_loss(trade, current_price, candle_data, entry_candle):
            logger.info(f"{candle_info}Stop loss hit for trade {instrument_key}, closing trade")
            return self.close_trade(
                instrument_key,
                current_price,
                TradeStatus.STOPPED_OUT,
                candle_data
            )
        
        # Check if it's time to close (15:15 PM or later)
        candle_time = current_time.time()
        if candle_time >= time(15, 15):
            logger.info(f"{candle_info}Market close time reached for trade {instrument_key}, closing trade")
            return self.close_trade(
                instrument_key=instrument_key,
                exit_price=current_price,
                status=TradeStatus.CLOSED,
                candle_data=candle_data
            )
        
        one_two_size_price = 0
        one_two_ratio_done = False
        # Update unrealized P&L
        if trade["position_type"] == "LONG":
            trade["unrealized_pnl"] = (current_price - trade["entry_price"]) * trade["current_position_size"]
            one_two_size_price = trade["entry_price"] + ((trade["entry_price"] - trade["stop_loss"]) * 2)
            if current_price >= one_two_size_price:
                one_two_ratio_done = True   
        else:  # SHORT
            trade["unrealized_pnl"] = (trade["entry_price"] - current_price) * trade["current_position_size"]
            one_two_size_price = trade["entry_price"] - ((trade["stop_loss"] - trade["entry_price"]) * 2)
            if current_price <= one_two_size_price:
                one_two_ratio_done = True
        # Update MAE/MFE
        trade["max_favorable_excursion"] = max(
            trade["max_favorable_excursion"],
            trade["unrealized_pnl"]
        )
        trade["max_adverse_excursion"] = min(
            trade["max_adverse_excursion"],
            trade["unrealized_pnl"]
        )
        
        # If price reaches 1:1R, move SL to breakeven
        if trade["trailing_stop_active"] == False and one_two_ratio_done == True:
            logger.info(f"{candle_info}Price reached 1:2R, moving SL to breakeven")
            trade["trailing_stop_active"] = True
            trade["status"] = TradeStatus.TRAILING.value
            trade["moved_to_one_two_size"] = False
            new_stop = trade["entry_price"]
            trade["stop_loss"] = new_stop
            logger.info(f"{candle_info}Updated trailing stop to Breakeven: {new_stop}")
        
        # Check take profit levels
        tp_level = self.check_take_profit_levels(trade, current_price, candle_data)
        if tp_level is not None:
            logger.info(f"{candle_info}Take profit level hit: {tp_level} at candle price: {current_price}")
            trade = self.execute_partial_exit(trade, tp_level, current_price, candle_data)
            
            # Check if position is fully closed
            if trade["current_position_size"] == 0:
                return self.close_trade(
                    instrument_key,
                    current_price,
                    TradeStatus.TAKE_PROFIT,
                    candle_data
                )
        
        # Check trade duration
        trade_duration = (datetime.now() - trade["entry_time"]).total_seconds() / 60
        if trade_duration > self.config.max_trade_duration:
            logger.info(f"{candle_info}Trade duration exceeded maximum {self.config.max_trade_duration} minutes")
            return self.close_trade(
                instrument_key,
                current_price,
                TradeStatus.EXPIRED,
                candle_data
            )
        
        return trade

    def close_trade(self,
                  instrument_key: str,
                  exit_price: float,
                  status: TradeStatus,
                  candle_data: Optional[Dict] = None) -> Dict:
        """Close trade with metrics."""
        candle_info = self._format_candle_info(candle_data) if candle_data else ""
        
        if instrument_key not in self.active_trades:
            logger.warning(f"{candle_info}No active trade found for {instrument_key}")
            return {}
        
        trade = self.active_trades.pop(instrument_key)
        
        # Calculate final realized P&L including any remaining position
        if trade["current_position_size"] > 0:
            if trade["position_type"] == "LONG":
                final_pnl = (exit_price - trade["entry_price"]) * trade["current_position_size"]
            else:  # SHORT
                final_pnl = (trade["entry_price"] - exit_price) * trade["current_position_size"]
            
            trade["realized_pnl"] += final_pnl
            if final_pnl > 0:
                self.winning_trades += 1
            else:
                self.losing_trades += 1
        
        # Update trade metrics
        trade["exit_price"] = exit_price
        trade["exit_time"] = datetime.now()
        trade["exit_candle"] = candle_data
        trade["status"] = status.value
        trade["duration"] = (trade["exit_time"] - trade["entry_time"]).total_seconds() / 60
        
        # Calculate overall R-multiple
        if trade["risk_amount"] > 0:
            trade["r_multiple"] = trade["realized_pnl"] / (trade["risk_amount"] * trade["initial_position_size"])
        
        # Add exit reason
        if status == TradeStatus.CLOSED:
            trade["exit_reason"] = "market_close"
        elif status == TradeStatus.STOPPED_OUT:
            trade["exit_reason"] = "stop_loss"
        elif status == TradeStatus.TAKE_PROFIT:
            trade["exit_reason"] = "take_profit"
        elif status == TradeStatus.EXPIRED:
            trade["exit_reason"] = "duration_expired"
        
        # Add to trade history
        # if same current_time trade already exists in the trade history, dont add it
        same_day_trade = False
        for t in self.trade_history:
            # t['current_time'] to date
            t_current_time = pd.to_datetime(t['current_time']).date()
            trade_current_time = pd.to_datetime(trade['current_time']).date()
            if t['instrument_key'] == instrument_key and t_current_time == trade_current_time:
                same_day_trade = True
                break
        if not same_day_trade:
            self.trade_history.append(trade)
        
        logger.info(f"{candle_info}Closed trade: {trade}")
        return trade

    def get_trade_metrics(self, instrument_key: str, candle_data: Optional[Dict] = None) -> Dict:
        """Get detailed metrics for a trade."""
        candle_info = self._format_candle_info(candle_data) if candle_data else ""
        
        if instrument_key not in self.active_trades:
            logger.warning(f"{candle_info}No active trade found for {instrument_key}")
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
            "remaining_tp_levels": len(trade["take_profit_levels"]) - len(trade["executed_take_profits"]),
            "exit_reason": trade.get("exit_reason", ""),
            "r_multiple": trade.get("r_multiple", 0.0),
            "current_candle": candle_data
        }
        
        logger.debug(f"{candle_info}Trade metrics: {metrics}")
        return metrics

    def check_stop_loss(self, trade: Dict, current_price: float, candle_data: Optional[Dict] = None, entry_candle: Optional[Dict] = None) -> bool:
        """Check if stop loss is hit."""
        candle_info = self._format_candle_info(candle_data) if candle_data else ""
        
        stop_hit = False
        # if the current candle is the same as the entry candle, return False
        if candle_data["timestamp"] == entry_candle["timestamp"]:
            return False

        if trade["position_type"] == "LONG":
            stop_hit = candle_data['low'] <= trade["stop_loss"]
        else:  # SHORT
            stop_hit = candle_data['high'] >= trade["stop_loss"]
            
        if stop_hit:
            logger.info(f"{candle_info}Stop loss hit at price {current_price}, stop level: {trade['stop_loss']}")
            
        return stop_hit

    def get_trade_statistics(self, candle_data: Optional[Dict] = None) -> Dict:
        """Get overall trade statistics."""
        candle_info = self._format_candle_info(candle_data) if candle_data else ""
        
        if not self.trade_history:
            return {
                "total_trades": 0,
                "winning_trades": 0,
                "losing_trades": 0,
                "win_rate": 0.0,
                "average_r": 0.0,
                "profit_factor": 0.0,
                "total_profit": 0.0,
                "total_loss": 0.0,
                "average_win": 0.0,
                "average_loss": 0.0,
                "overall_pnl": 0.0,
                "profit_percentage": 0.0,
                "loss_percentage": 0.0,
                "expectancy": 0.0
            }
        
        # Separate winning and losing trades
        winning_trades = [t for t in self.trade_history if t["realized_pnl"] > 0]
        losing_trades = [t for t in self.trade_history if t["realized_pnl"] <= 0]
        
        # Calculate basic statistics
        total_trades = len(self.trade_history)
        winning_count = len(winning_trades)
        losing_count = len(losing_trades)
        
        # Calculate profit/loss metrics
        total_profit = sum(t["realized_pnl"] for t in winning_trades)
        total_loss = abs(sum(t["realized_pnl"] for t in losing_trades))
        
        # Calculate averages
        avg_win = total_profit / winning_count if winning_count > 0 else 0
        avg_loss = total_loss / losing_count if losing_count > 0 else 0
        
        # Calculate percentages
        profit_percentage = (total_profit / self.config.initial_capital * 100) if self.config.initial_capital > 0 else 0
        loss_percentage = (total_loss / self.config.initial_capital * 100) if self.config.initial_capital > 0 else 0
        
        # Calculate expectancy
        win_rate = winning_count / total_trades if total_trades > 0 else 0
        loss_rate = 1 - win_rate
        expectancy = (win_rate * avg_win) - (loss_rate * avg_loss)
        
        # Calculate other metrics
        profit_factor = total_profit / total_loss if total_loss > 0 else float('inf')
        avg_r = np.mean([t["r_multiple"] for t in self.trade_history]) if self.trade_history else 0
        overall_pnl = total_profit - total_loss
        
        stats = {
            "total_trades": total_trades,
            "winning_trades": winning_count,
            "losing_trades": losing_count,
            "win_rate": win_rate,  # Convert to percentage
            "average_r": avg_r,
            "profit_factor": profit_factor,
            "total_profit": total_profit,
            "total_loss": total_loss,
            "average_win": avg_win,
            "average_loss": avg_loss,
            "overall_pnl": overall_pnl,
            "profit_percentage": profit_percentage,
            "loss_percentage": loss_percentage,
            "expectancy": expectancy
        }
        
        logger.info(f"{candle_info}Trade statistics: {stats}")
        return stats 