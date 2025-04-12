"""
Backtest Simulator for Morning Range Strategy.

This module provides trade execution simulation with market impact modeling
and multi-timeframe support.
"""

from dataclasses import dataclass
from enum import Enum
from typing import Dict, List, Optional, Union, Tuple
import pandas as pd
import numpy as np
from datetime import datetime, time, timedelta
import logging

from ..strategy.position_manager import PositionManager, PositionStatus
from ..strategy.trade_manager import TradeManager, TradeStatus
from ..strategy.risk_calculator import RiskCalculator
from ..strategy.mr_strategy_base import MorningRangeStrategy

logger = logging.getLogger(__name__)

class SimulationEvent(Enum):
    """Simulation event types."""
    TICK = "tick"
    TRADE = "trade"
    RANGE_BREAK = "range_break"
    STOP_LOSS = "stop_loss"
    TAKE_PROFIT = "take_profit"
    MARKET_OPEN = "market_open"
    MARKET_CLOSE = "market_close"
    ERROR = "error"

@dataclass
class MarketImpactConfig:
    """Configuration for market impact simulation."""
    slippage_percentage: float = 0.01  # Base slippage
    volume_impact_factor: float = 0.1  # Impact based on volume
    spread_percentage: float = 0.02  # Bid-ask spread
    tick_size: float = 0.05  # Minimum price movement
    min_volume: int = 100  # Minimum volume for valid trades
    max_position_volume: float = 0.1  # Max % of candle volume

@dataclass
class SimulationConfig:
    """Configuration for backtest simulation."""
    market_impact: MarketImpactConfig
    time_in_force: int  # Minutes
    enable_partial_fills: bool = True
    enable_market_impact: bool = True
    enable_volume_validation: bool = True
    replay_speed: int = 1  # 1x, 2x, etc.

class BacktestSimulator:
    """Simulator for backtesting trade execution."""
    
    def __init__(self, 
                 config: SimulationConfig,
                 position_manager: PositionManager,
                 trade_manager: TradeManager,
                 risk_calculator: RiskCalculator):
        """Initialize the Backtest Simulator."""
        self.config = config
        self.position_manager = position_manager
        self.trade_manager = trade_manager
        self.risk_calculator = risk_calculator
        
        self.events: List[Dict] = []
        self.current_time: datetime = None
        self.market_state: Dict = {}
        
        logger.info("Initialized BacktestSimulator")

    def calculate_market_impact(self, 
                              order: Dict,
                              candle: Dict) -> Tuple[float, float]:
        """
        Calculate market impact and slippage for an order.
        
        Args:
            order: Order details
            candle: Current candle data
            
        Returns:
            Tuple[float, float]: (Adjusted price, Slippage amount)
        """
        if not self.config.enable_market_impact:
            return order["price"], 0.0

        base_slippage = order["price"] * (self.config.market_impact.slippage_percentage / 100)
        
        # Volume-based impact
        volume_ratio = order["size"] / candle["volume"]
        volume_impact = base_slippage * (volume_ratio / self.config.market_impact.volume_impact_factor)
        
        # Add spread cost
        spread_cost = order["price"] * (self.config.market_impact.spread_percentage / 100)
        
        total_impact = base_slippage + volume_impact + spread_cost
        
        # Round to tick size
        adjusted_price = round(
            order["price"] + (total_impact if order["side"] == "BUY" else -total_impact),
            2
        )
        
        logger.debug(f"Market impact calculation: base={base_slippage}, volume={volume_impact}, spread={spread_cost}")
        return adjusted_price, total_impact

    def validate_execution(self, 
                         order: Dict,
                         candle: Dict) -> Tuple[bool, str]:
        """
        Validate if an order can be executed.
        
        Args:
            order: Order details
            candle: Current candle data
            
        Returns:
            Tuple[bool, str]: (Is valid, Reason if not valid)
        """

        logger.info("###VALIDATE EXECUTION order: %s, candle: %s", order, candle)

        # Check minimum volume
        if candle["volume"] < self.config.market_impact.min_volume:
            return False, "Insufficient market volume"
        
        # Check maximum position volume
        max_allowed_volume = candle["volume"] * self.config.market_impact.max_position_volume
        if order["size"] > max_allowed_volume:
            return False, f"Order size exceeds {self.config.market_impact.max_position_volume*100}% of candle volume"
        
        # Price validation
        if order["side"] == "BUY":
            if order["price"] < candle["low"] or order["price"] > candle["high"]:
                return False, "Price outside candle range"
        else:  # SELL
            if order["price"] < candle["low"] or order["price"] > candle["high"]:
                return False, "Price outside candle range"
        
        return True, "Order valid"

    def simulate_execution(self, 
                         order: Dict,
                         candle: Dict) -> Dict:
        """
        Simulate order execution with market impact.
        
        Args:
            order: Order to execute
            candle: Current market data
            
        Returns:
            Dict: Execution result
        """
        # Validate execution
        is_valid, reason = self.validate_execution(order, candle)
        if not is_valid:
            logger.warning(f"Order validation failed: {reason}")
            return {
                "status": "rejected",
                "reason": reason,
                "order": order
            }
        
        # Calculate market impact
        executed_price, slippage = self.calculate_market_impact(order, candle)
        logger.info(f"Executed price: {executed_price}, Slippage: {slippage}")
        
        # Handle partial fills if enabled
        executed_size = order["size"]
        if self.config.enable_partial_fills:
            volume_ratio = order["size"] / candle["volume"]
            if volume_ratio > self.config.market_impact.max_position_volume:
                executed_size = candle["volume"] * self.config.market_impact.max_position_volume
                logger.info(f"Partial fill: {executed_size}/{order['size']} units")
        
        execution = {
            "order_id": order.get("order_id", ""),
            "status": "filled",
            "executed_price": executed_price,
            "executed_size": executed_size,
            "slippage": slippage,
            "execution_time": self.current_time,
            "original_order": order
        }
        
        self.events.append({
            "timestamp": self.current_time,
            "type": SimulationEvent.TRADE.value,
            "data": execution
        })
        
        return execution

    def process_candle(self, 
                      strategy: MorningRangeStrategy,
                      candle: Dict) -> List[Dict]:
        """
        Process a candle and simulate strategy execution.
        
        Args:
            strategy: Strategy instance
            candle: Candle data
            
        Returns:
            List[Dict]: List of executions
        """
        self.current_time = candle["timestamp"]
        executions = []
        
        # Update market state
        self.market_state = {
            "timestamp": candle["timestamp"],
            "open": candle["open"],
            "high": candle["high"],
            "low": candle["low"],
            "close": candle["close"],
            "volume": candle["volume"]
        }
        
        # Process strategy signals
        result = strategy.process_candle(candle)
        
        if result["action"] in ["entry", "exit"]:
            order = self._create_order_from_signal(result, candle)
            execution = self.simulate_execution(order, candle)
            executions.append(execution)
            
            # Update position and trade managers
            if execution["status"] == "filled":
                if result["action"] == "entry":
                    self._handle_entry_execution(execution, strategy)
                else:  # exit
                    self._handle_exit_execution(execution, strategy)
        
        return executions

    def _create_order_from_signal(self, 
                                signal: Dict,
                                candle: Dict) -> Dict:
        """Create order object from strategy signal."""
        is_entry = signal["action"] == "entry"
        
        order = {
            "order_id": f"order_{datetime.now().timestamp()}",
            "instrument_key": signal["result"].get("instrument_key", ""),
            "side": "BUY" if (is_entry and signal["result"].get("position_type") == "LONG") 
                   or (not is_entry and signal["result"].get("position_type") == "SHORT") else "SELL",
            "size": signal["result"].get("position_size", 0),
            "price": candle["close"],
            "order_type": "MARKET",
            "time_in_force": self.config.time_in_force
        }
        
        return order

    def _handle_entry_execution(self, 
                              execution: Dict,
                              strategy: MorningRangeStrategy):
        """Handle entry execution updates."""
        order = execution["original_order"]
        
        # Update position manager
        position = self.position_manager.add_position(
            instrument_key=order["instrument_key"],
            size=execution["executed_size"],
            entry_price=execution["executed_price"],
            sl_percentage=strategy.config.sl_percentage,
            position_type="LONG" if order["side"] == "BUY" else "SHORT"
        )
        
        # Update trade manager
        trade = self.trade_manager.create_extended_trade(
            instrument_key=order["instrument_key"],
            entry_price=execution["executed_price"],
            position_size=execution["executed_size"],
            position_type="LONG" if order["side"] == "BUY" else "SHORT",
            trade_type=strategy.config.entry_type
        )
        
        logger.info(f"Handled entry execution: {position}")

    def _handle_exit_execution(self, 
                             execution: Dict,
                             strategy: MorningRangeStrategy):
        """Handle exit execution updates."""
        order = execution["original_order"]
        
        # Update position manager
        position = self.position_manager.close_position(
            instrument_key=order["instrument_key"],
            exit_price=execution["executed_price"]
        )
        
        # Update trade manager
        trade = self.trade_manager.close_extended_trade(
            instrument_key=order["instrument_key"],
            exit_price=execution["executed_price"],
            status=TradeStatus.CLOSED
        )
        
        logger.info(f"Handled exit execution: {position}")

    def get_simulation_metrics(self) -> Dict:
        """Get simulation performance metrics."""
        return {
            "total_events": len(self.events),
            "execution_summary": self._generate_execution_summary(),
            "market_impact_analysis": self._analyze_market_impact(),
            "volume_profile": self._analyze_volume_profile()
        }

    def _generate_execution_summary(self) -> Dict:
        """Generate summary of executions."""
        trade_events = [e for e in self.events if e["type"] == SimulationEvent.TRADE.value]
        
        return {
            "total_trades": len(trade_events),
            "filled_trades": len([e for e in trade_events if e["data"]["status"] == "filled"]),
            "rejected_trades": len([e for e in trade_events if e["data"]["status"] == "rejected"]),
            "average_slippage": np.mean([e["data"]["slippage"] for e in trade_events 
                                       if e["data"]["status"] == "filled"]),
            "max_slippage": max([e["data"]["slippage"] for e in trade_events 
                               if e["data"]["status"] == "filled"], default=0)
        }

    def _analyze_market_impact(self) -> Dict:
        """Analyze market impact of trades."""
        trade_events = [e for e in self.events if e["type"] == SimulationEvent.TRADE.value
                       and e["data"]["status"] == "filled"]
        
        return {
            "average_price_impact": np.mean([e["data"]["slippage"] / e["data"]["executed_price"] * 100 
                                           for e in trade_events]),
            "volume_distribution": np.percentile([e["data"]["executed_size"] for e in trade_events],
                                               [25, 50, 75, 90, 95])
        }

    def _analyze_volume_profile(self) -> Dict:
        """Analyze volume profile of executions."""
        trade_events = [e for e in self.events if e["type"] == SimulationEvent.TRADE.value
                       and e["data"]["status"] == "filled"]
        
        return {
            "average_execution_size": np.mean([e["data"]["executed_size"] for e in trade_events]),
            "partial_fills_percentage": len([e for e in trade_events 
                                           if e["data"]["executed_size"] < e["data"]["original_order"]["size"]]) / 
                                      len(trade_events) * 100 if trade_events else 0
        }