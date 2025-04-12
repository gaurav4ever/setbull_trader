"""
Backtest Engine for Morning Range strategy.

This module provides the core backtesting functionality for the Morning Range strategy,
handling data processing, signal generation, and trade execution simulation.
"""

import pandas as pd
import numpy as np
from typing import Dict, List, Optional, Union, Any
from datetime import datetime, time
import logging
import asyncio
from dataclasses import dataclass
from enum import Enum

from ..data.data_processor import CandleProcessor
from ..strategy.signal_generator import SignalGenerator
from ..strategy.config import MRStrategyConfig
from ..strategy.models import Signal, SignalType
from .simulator import BacktestSimulator, SimulationConfig, MarketImpactConfig
from ..strategy.position_manager import (
    PositionManager, 
    AccountInfo, 
    PositionSizeConfig, 
    RiskLimits,
    PositionSizeType
)
from ..strategy.trade_manager import (
    TradeManager,
    TradeConfig,
    TakeProfitLevel,
    TradeType,
    TradeStatus
)
from ..strategy.risk_calculator import RiskCalculator, RiskConfig
from ..strategy.mr_strategy_base import MorningRangeStrategy

# Configure logging
logger = logging.getLogger(__name__)
logger.setLevel(logging.DEBUG)

@dataclass
class BacktestConfig:
    """Configuration for backtest execution."""
    start_date: datetime
    end_date: datetime
    instruments: List[str]
    strategies: List[MRStrategyConfig]
    initial_capital: float
    position_size_type: str
    max_positions: int
    enable_parallel: bool = True
    cache_data: bool = True
    trading_hours: Optional[Dict[str, time]] = None
    excluded_dates: Optional[List[datetime]] = None

class BacktestEngine:
    """Core backtesting engine for Morning Range strategy."""
    
    def __init__(self, config: BacktestConfig):
        """
        Initialize the Backtest Engine.
        
        Args:
            config: Backtest configuration
        """
        self.config = config
        self.signal_generator = SignalGenerator(config.strategies[0])  # Use first strategy config
        self.data_processor = CandleProcessor(config={
            'instrument_key': config.strategies[0].instrument_key
        })
        
        # Create account info
        account_info = AccountInfo(
            total_capital=config.initial_capital,
            available_capital=config.initial_capital,
            max_position_size=config.initial_capital * 0.5,  # 10% max position size
            risk_per_trade=1.0,  # 1% risk per trade
            max_risk_per_trade=config.initial_capital * 0.01,
            currency="INR"
        )
        
        # Create position config
        position_config = PositionSizeConfig(
            size_type=PositionSizeType[config.position_size_type],
            value=0.1,  # 1% risk or account size
            min_size=1.0,
            max_size=float('inf'),
            round_to=0
        )
        
        # Create risk limits
        risk_limits = RiskLimits(
            max_daily_loss=config.initial_capital * 0.03,  # 3% max daily loss
            max_position_loss=config.initial_capital * 0.01,  # 1% max position loss
            max_open_positions=config.max_positions,
            max_daily_trades=10,  # Maximum 10 trades per day
            max_risk_multiplier=2.0,  # Maximum 2x risk for scaling
            position_size_limit=10.0,  # Maximum 10% position size
            max_correlated_positions=2  # Maximum 2 correlated positions
        )
        
        # Create trade config
        take_profit_levels = [
            TakeProfitLevel(
                r_multiple=3.0,
                size_percentage=0.3,
                trail_activation=False,
                move_sl_to_be=True
            ),
            TakeProfitLevel(
                r_multiple=5.0,
                size_percentage=0.5,
                trail_activation=True,
                move_sl_to_be=False
            ),
            TakeProfitLevel(
                r_multiple=7.0,
                size_percentage=1.0,
                trail_activation=True,
                move_sl_to_be=False
            )
        ]
        
        trade_config = TradeConfig(
            sl_percentage=0.5,  # 0.5% stop loss
            take_profit_levels=take_profit_levels,
            breakeven_r=1.0,
            trail_activation_r=1.5,
            trail_step_percentage=0.2,
            partial_exit_adjustment=True,
            max_trade_duration=360,  # 6 hours
            entry_timeout=5,
            reentry_times=1,
            min_risk_reward=1.5,
            dynamic_sl_adjustment=True,
            initial_capital=100000.0
        )
        
        # Create risk config
        risk_config = RiskConfig(
            max_risk_per_trade=1.0,  # 1% max risk per trade
            max_daily_risk=3.0,  # 3% max daily risk
            max_correlated_risk=2.0,  # 2% max correlated risk
            position_size_limit=10.0,  # 10% max position size
            max_drawdown_limit=5.0,  # 5% max drawdown
            risk_free_rate=0.05,  # 5% risk-free rate
            correlation_threshold=0.7  # 70% correlation threshold
        )
        
        # Initialize managers
        self.position_manager = PositionManager(
            account_info=account_info,
            position_config=position_config,
            risk_limits=risk_limits
        )
        self.trade_manager = TradeManager(trade_config=trade_config)
        self.risk_calculator = RiskCalculator(risk_config=risk_config)
        
        # Create simulation config with required arguments
        market_impact_config = MarketImpactConfig(
            slippage_percentage=0.01,
            volume_impact_factor=0.1,
            spread_percentage=0.02,
            tick_size=0.05,
            min_volume=100,
            max_position_volume=0.1
        )
        
        simulation_config = SimulationConfig(
            market_impact=market_impact_config,
            time_in_force=5  # 5 minutes time in force
        )
        
        self.simulator = BacktestSimulator(
            config=simulation_config,
            position_manager=self.position_manager,
            trade_manager=self.trade_manager,
            risk_calculator=self.risk_calculator
        )
        
        self.signals: List[Signal] = []
        self.trades: List[Dict[str, Any]] = []
        self.metrics: Dict[str, Any] = {}
        
        logger.info("Initialized BacktestEngine")
        logger.debug(f"Backtest config: {config}")

    async def run_backtest(self, data: Dict[str, pd.DataFrame]) -> Dict[str, Any]:
        """
        Run backtest on the provided data.
        
        Args:
            data: Dictionary of instrument keys to their respective DataFrames with OHLCV data
            
        Returns:
            Dictionary containing backtest results
        """
        logger.info("Starting backtest run")
        
        all_results = {
            'signals': [],
            'trades': [],
            'metrics': {}
        }
        
        # Process candles for each instrument
        for instrument_key, instrument_data in data.items():
            # Process candles using CandleProcessor
            async with self.data_processor as processor:
                processed_data = processor.process_candles(instrument_data)
                logger.info(f"Processed candle data for {instrument_key}")
                
                # Run backtest for each strategy
                for strategy_config in self.config.strategies:
                    if strategy_config.instrument_key == instrument_key:
                        self.signal_generator = SignalGenerator(strategy_config)
                        strategy_results = await self._run_single_strategy(processed_data)
                        
                        # Combine results
                        all_results['signals'].extend(strategy_results['signals'])
                        all_results['trades'].extend(strategy_results['trades'])
                        
                        # Store metrics by strategy
                        strategy_id = f"{strategy_config.instrument_key}_{strategy_config.range_type}_{strategy_config.entry_type}"
                        all_results['metrics'][strategy_id] = strategy_results['metrics']
        
        # Calculate overall metrics
        self.metrics = self._calculate_metrics()
        all_results['metrics']['overall'] = self.metrics
        
        logger.info("Completed backtest run")
        return all_results

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

    async def _run_single_strategy(self, data: pd.DataFrame) -> Dict[str, Any]:
        """
        Run backtest for a single strategy.
        
        Args:
            data: DataFrame containing processed candle data
            
        Returns:
            Dictionary containing strategy results:
                - signals: List of generated signals
                - trades: List of executed trades
                - metrics: Performance metrics
        """
        signals: List[Signal] = []
        trades: List[Dict[str, Any]] = []
        
        # Process candles using CandleProcessor
        async with self.data_processor as processor:
            processed_data = processor.process_candles(data)
            logger.info(f"Processed {len(processed_data)} candles for strategy")
            
            # Create MorningRangeStrategy instance
            mr_strategy = MorningRangeStrategy(
                config=self.config.strategies[0],
                position_manager=self.position_manager,
                trade_manager=self.trade_manager,
                risk_calculator=self.risk_calculator
            )
            
            # Get unique dates from timestamp column
            unique_dates = processed_data['timestamp'].dt.date.unique()
            logger.info(f"Found {len(unique_dates)} unique trading days")
            
            # Dictionary to store valid dates and their MR values
            valid_dates_mr: Dict[date, Dict] = {}
            
            # PHASE 1: MR Validation and Date Filtering
            logger.info(f"#########################################")
            logger.info("Starting MR validation phase")
            for date in unique_dates:
                # Filter candles for this date
                day_candles = processed_data[processed_data['timestamp'].dt.date == date]
                if day_candles.empty:
                    logger.debug(f"No candles found for {date}")
                    continue
                    
                # Calculate morning range values
                mr_values = await mr_strategy.calculate_morning_range(day_candles)
                logger.info(f"MR values: {mr_values}")
                
                # Validate MR values
                if mr_values.get('is_valid', False):
                    logger.info(f"Valid MR for {date}: High={mr_values['mr_high']}, Low={mr_values['mr_low']}")
                    logger.debug(f"MR validation details: {mr_values.get('validation_details', {})}")
                    valid_dates_mr[date] = mr_values
                else:
                    logger.warning(f"Invalid MR for {date}: {mr_values.get('validation_reason', 'Unknown reason')}")
                    logger.debug(f"MR validation details: {mr_values.get('validation_details', {})}")
                    continue
                    
                # Calculate entry levels for valid dates
                entry_levels = mr_strategy.calculate_entry_levels()
                logger.debug(f"Calculated entry levels for {date}: {entry_levels}")
            
            logger.info(f"MR validation complete. Found {len(valid_dates_mr)} valid trading days")
            
            # PHASE 2: Signal Generation for Valid Dates
            logger.info(f"#########################################")
            logger.info(f"Starting signal generation phase for valid dates: {valid_dates_mr}")
            for date, mr_values in valid_dates_mr.items():
                # Get candles for this valid date
                day_candles = processed_data[processed_data['timestamp'].dt.date == date]
                logger.debug(f"Processing {len(day_candles)} candles for valid date {date} with mr_values: {mr_values}")
                entry_candle = None
                
                # Generate signals for each candle
                for idx, candle in day_candles.iterrows():
                    candle_dict = candle.to_dict()
                    candle_info = self._format_candle_info(candle_dict)
                    
                    # Generate signals using morning range values
                    strategy_signals = await self.signal_generator.process_candle(candle_dict, mr_values)
                    
                    if strategy_signals:
                        signals.extend(strategy_signals)
                        logger.info(f"{candle_info}Generated {len(strategy_signals)} signals")
                        
                        # Process each signal using trade manager
                        for signal in strategy_signals:
                            # Calculate position size
                            position_size = self.position_manager.calculate_position_size(
                                signal.price,
                                self.config.strategies[0].sl_percentage,
                                signal.direction.value
                            )
                            
                            # Create trade using trade manager
                            trade = self.trade_manager.create_trade(
                                instrument_key=self.config.strategies[0].instrument_key,
                                entry_price=signal.price,
                                position_size=position_size,
                                position_type=signal.direction.value,
                                trade_type=TradeType.IMMEDIATE_BREAKOUT if signal.type == SignalType.IMMEDIATE_BREAKOUT else TradeType.RETEST_ENTRY,
                                sl_percentage=self.config.strategies[0].sl_percentage
                            )
                            entry_candle = candle_dict
                            
                            if trade:
                                logger.info(f"{candle_info}Created new trade for signal: {signal.type} at price {signal.price}")
                    
                    # Process all active trades for this candle
                    for instrument_key in list(self.trade_manager.active_trades.keys()):
                        try:
                            updated_trade = self.trade_manager.update_trade(
                                instrument_key=instrument_key,
                                current_price=candle_dict['close'],
                                current_time=candle_dict['timestamp'],
                                candle_data=candle_dict,
                                entry_candle=entry_candle
                            )
                            
                            if updated_trade:
                                if updated_trade['status'] in [TradeStatus.CLOSED.value, 
                                                             TradeStatus.STOPPED_OUT.value, 
                                                             TradeStatus.TAKE_PROFIT.value]:
                                    logger.info(f"{candle_info}Trade {instrument_key} closed with status {updated_trade['status']} and pnl {updated_trade['realized_pnl']} and updated_trade: {updated_trade}")
                                    # check if same day trade is already present in the trades list
                                    same_day_trade = False
                                    for trade in trades:
                                        trade_current_time = pd.to_datetime(trade['current_time'])
                                        updated_trade_current_time = pd.to_datetime(updated_trade['current_time'])
                                        if trade['instrument_key'] == instrument_key and trade_current_time.date() == updated_trade_current_time.date():
                                            same_day_trade = True
                                            break
                                    if not same_day_trade:
                                        trades.append(updated_trade)
                                    self.signal_generator.reset_signal_and_state()
                        except Exception as e:
                            logger.error(f"{candle_info}Error updating trade {instrument_key}: {str(e)}")
                            continue
        
        # Calculate metrics for this strategy
        metrics = self._calculate_metrics()
        
        return {
            'signals': signals,
            'trades': trades,
            'metrics': metrics
        }

    def _calculate_metrics(self) -> Dict[str, Any]:
        """
        Calculate backtest performance metrics using trade manager's statistics.
        
        Returns:
            Dictionary containing performance metrics
        """
        # Get trade statistics from trade manager
        trade_stats = self.trade_manager.get_trade_statistics()
        
        # Calculate equity curve for drawdown calculation
        equity_curve = self._build_equity_curve()
        
        # Calculate max drawdown from equity curve
        if not equity_curve.empty:
            rolling_max = equity_curve.expanding().max()
            drawdowns = (equity_curve - rolling_max) / rolling_max
            max_drawdown = abs(drawdowns.min())
        else:
            max_drawdown = 0.0
        
        # Calculate Sharpe ratio
        if trade_stats['total_trades'] > 0:
            returns = pd.Series([trade.get('realized_pnl', 0) for trade in self.trade_manager.trade_history])
            excess_returns = returns.mean() - (0.02 / 252)  # Daily risk-free rate
            sharpe_ratio = excess_returns / returns.std() if returns.std() != 0 else 0.0
        else:
            sharpe_ratio = 0.0
        
        import math

        metrics = {
            'total_signals': len(self.signals),
            'total_trades': int(trade_stats['total_trades']),
            'winning_trades': int(trade_stats['winning_trades']),
            'losing_trades': int(trade_stats['losing_trades']),
            'win_rate': round(float(trade_stats['win_rate']), 2),
            'profit_factor': 0 if math.isinf(float(trade_stats['profit_factor'])) else round(float(trade_stats['profit_factor']), 2),
            'total_profit': round(float(trade_stats['total_profit']), 2),
            'total_loss': round(float(trade_stats['total_loss']), 2),
            'average_win': round(float(trade_stats['average_win']), 2),
            'average_loss': round(float(trade_stats['average_loss']), 2),
            'overall_pnl': round(float(trade_stats['overall_pnl']), 2),
            'profit_percentage': round(float(trade_stats['profit_percentage']), 2),
            'loss_percentage': round(float(trade_stats['loss_percentage']), 2),
            'expectancy': round(float(trade_stats['expectancy']), 2),
            'net_pnl': round(float(trade_stats['total_profit'] - trade_stats['total_loss']), 2),
            'average_r': round(float(trade_stats['average_r']), 2),
            'max_drawdown': round(float(max_drawdown), 2)
        }



        logger.info(f"Calculated performance metrics: {metrics}")
        return metrics

    def _build_equity_curve(self) -> pd.Series:
        """
        Build equity curve from trade history.
        
        Returns:
            Series representing equity curve
        """
        if not self.trade_manager.trade_history:
            return pd.Series([])
        
        # Create DataFrame from trade history
        trades_df = pd.DataFrame(self.trade_manager.trade_history)
        
        # Sort trades by exit time
        trades_df['exit_time'] = pd.to_datetime(trades_df['exit_time'])
        trades_df = trades_df.sort_values('exit_time')
        
        # Calculate cumulative P&L
        cumulative_pnl = trades_df['realized_pnl'].cumsum()
        
        # Add initial capital
        equity_curve = self.config.initial_capital + cumulative_pnl
        
        logger.debug(f"Built equity curve with {len(equity_curve)} points")
        return equity_curve