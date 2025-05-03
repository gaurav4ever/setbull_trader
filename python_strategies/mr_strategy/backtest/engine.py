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
    instruments: List[Dict[str, str]]  # Changed from List[str] to List[Dict]
    strategies: List[MRStrategyConfig]
    initial_capital: float
    position_size_type: str
    max_positions: int
    enable_parallel: bool = True
    cache_data: bool = True
    trading_hours: Optional[Dict[str, time]] = None
    excluded_dates: Optional[List[datetime]] = None

    def __post_init__(self):
        """Validate instrument configurations."""
        for instrument in self.instruments:
            if not isinstance(instrument, dict):
                raise ValueError("Each instrument must be a dictionary with 'key' and 'direction'")
            if 'key' not in instrument or 'direction' not in instrument:
                raise ValueError("Instrument configuration must contain 'key' and 'direction'")
            if instrument['direction'] not in ['BULLISH', 'BEARISH']:
                raise ValueError("Direction must be either 'BULLISH' or 'BEARISH'")

class BacktestEngine:
    """Core backtesting engine for Morning Range strategy."""
    
    def __init__(self, 
                config: Dict[str, Any],
                data_processor: Optional[CandleProcessor] = None,
                position_manager: Optional[PositionManager] = None,
                trade_manager: Optional[TradeManager] = None,
                risk_calculator: Optional[RiskCalculator] = None):
        """
        Initialize the backtest engine.
        
        Args:
            config: Strategy configuration
            data_processor: Optional data processor instance
            position_manager: Optional position manager instance
            trade_manager: Optional trade manager instance
            risk_calculator: Optional risk calculator instance
        """
        self.config = config
        self.data_processor = data_processor or CandleProcessor()
        
        # # Create morning range calculator
        # self.mr_calculator = MorningRangeCalculator(
        #     range_type=config.get('range_type', '5MR'),
        #     respect_trend=config.get('respect_trend', True)
        # )
        
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
            value=0.05,  # 1% risk or account size
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
        
        logger.info(f"Initialized BacktestEngine with config: {config}")

    async def run_backtest(self, all_data_feed: Dict[str, pd.DataFrame]) -> Dict[str, Any]:
        """
        Run backtest on the provided intraday data.
        
        Args:
            intraday_data: Dictionary of instrument keys to their respective DataFrames with OHLCV data
            
        Returns:
            Dictionary containing backtest results:
                - signals: List of all generated signals
                - trades: List of all executed trades
                - metrics: Dictionary of strategy-specific metrics
                - instruments: Dictionary of instrument-specific results
                - portfolio: Dictionary containing portfolio-level results
        """
        logger.info("Starting backtest run")
        
        all_results = {
            'instruments': {},
            'signals': [],
            'trades': [],
            'metrics': {},
            'equity_curve': pd.Series(),
            'portfolio': {
                'equity_curve': pd.Series(),
                'metrics': {}
            }
        }
        
        # Process each instrument sequentially
        for instrument_key, instrument_data_feed in all_data_feed.items():
            # Find instrument config
            instrument_config = next(
                (inst for inst in self.config.instruments 
                 if inst['key'] == instrument_key),
                None
            )
            
            if not instrument_config:
                logger.warning(f"No configuration found for instrument {instrument_key}")
                continue
            
            logger.info(f"Processing instrument {instrument_key} ({instrument_config['direction']})")
            
            # Process candles using CandleProcessor
            logger.info(f"Processed {len(instrument_data_feed)} candles for {instrument_key}")
            # Run backtest for each strategy
            
            for strategy_config in self.config.strategies:
                if strategy_config.instrument_key.get('key') == instrument_key:
                    self.signal_generator = SignalGenerator(strategy_config, entry_type=strategy_config.entry_type)
                    # ---------------------------------------------
                    # Finally run the backtest for the strategy
                    # BACKTEST IMPLEMENTATION STARTS HERE
                    # ---------------------------------------------
                    strategy_results = await self._run_single_strategy(instrument_data_feed, instrument_config, strategy_config)
                    
                    # Store instrument-specific results
                    all_results['instruments'][instrument_key] = {
                        'direction': instrument_config['direction'],
                        'signals': strategy_results['signals'],
                        'trades': strategy_results['trades'],
                        'metrics': strategy_results['metrics'],
                        'equity_curve': self._build_equity_curve(strategy_results['trades'])
                    }
                    
                    # don't combine results, just store them
                    all_results['signals'].extend(strategy_results['signals'])
                    all_results['trades'].extend(strategy_results['trades'])
                    
                    # Store metrics by strategy
                    strategy_id = f"{strategy_config.instrument_key}_{strategy_config.range_type}_{strategy_config.entry_type}"
                    all_results['metrics'][strategy_id] = strategy_results['metrics']

        # Calculate portfolio-level metrics
        portfolio_metrics = self._calculate_portfolio_metrics(all_results)
        all_results['portfolio']['metrics'] = portfolio_metrics
        
        # Build portfolio equity curve
        portfolio_equity = self._build_portfolio_equity_curve(all_results)
        all_results['portfolio']['equity_curve'] = portfolio_equity
        
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

    async def _run_single_strategy(
            self, 
            data: pd.DataFrame, 
            instrument_config: Dict[str, Any], 
            strategy_config: MRStrategyConfig) -> Dict[str, Any]:
        """
        Run backtest for a single strategy.
        
        Args:
            data: DataFrame containing processed candle data
            instrument_config: Dictionary containing instrument configuration including direction
            
        Returns:
            Dictionary containing strategy results:
                - signals: List of generated signals
                - trades: List of executed trades
                - metrics: Performance metrics
        """
        signals: List[Signal] = []
        trades: List[Dict[str, Any]] = []
        
        # Process candles using CandleProcessor
        processed_data = data.copy()
        logger.info(f"Processed {len(processed_data)} candles for strategy {strategy_config.instrument_key}")
        
        # Create MorningRangeStrategy instance
        mr_strategy = MorningRangeStrategy(
            config=strategy_config,
            position_manager=self.position_manager,
            trade_manager=self.trade_manager,
            risk_calculator=self.risk_calculator
        )
        
        # Get unique dates from timestamp column
        unique_dates = processed_data['timestamp'].dt.date.unique()
        logger.info(f"Found {len(unique_dates)} unique trading days")
        
        # Dictionary to store valid dates and their MR values
        valid_dates_range: Dict[date, Dict] = {}
        
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
            range_values = await mr_strategy.calculate_morning_range(day_candles)
            logger.info(f"MR values: {range_values}")
            
            # Validate MR values
            if strategy_config.entry_type != '1ST_ENTRY':
                valid_dates_range[date] = range_values
            elif range_values.get('is_valid', False):
                logger.info(f"Valid MR for {date}: High={range_values['mr_high']}, Low={range_values['mr_low']}")
                logger.debug(f"MR validation details: {range_values.get('validation_details', {})}")
                valid_dates_range[date] = range_values
            else:
                logger.warning(f"Invalid MR for {date}: {range_values.get('validation_reason', 'Unknown reason')}")
                logger.debug(f"MR validation details: {range_values.get('validation_details', {})}")
                continue
                
            # Calculate entry levels for valid dates if MR is valid
            entry_levels = mr_strategy.calculate_entry_levels()
            logger.debug(f"Calculated entry levels for {date}: {entry_levels}")
        
        logger.info(f"MR validation complete. Found {len(valid_dates_range)} valid trading days")
        
        # PHASE 2: Signal Generation for Valid Dates
        logger.info(f"#########################################")
        logger.info(f"Starting signal generation phase for valid dates: {valid_dates_range}")
        for date, range_values in valid_dates_range.items():
            # Get candles for this valid date
            day_candles = processed_data[processed_data['timestamp'].dt.date == date]
            logger.debug(f"Processing {len(day_candles)} candles for valid date {date} with mr_values: {range_values}")
            entry_candle = None
            
            # Generate signals for each candle
            for idx, candle in day_candles.iterrows():
                candle_dict = candle.to_dict()
                candle_info = self._format_candle_info(candle_dict)
                
                # --------------------------
                # SIGNAL GENERATION PHASE
                # --------------------------
                strategy_signals = await self.signal_generator.process_candle(candle_dict, range_values)
                
                if strategy_signals:
                    # Filter signals based on instrument direction
                    filtered_signals = []
                    for signal in strategy_signals:
                        if signal.direction.value == "LONG":
                            filtered_signals.append(signal)
                            logger.info(f"{candle_info}Accepted LONG signal for BULLISH instrument")
                        elif signal.direction.value == "SHORT":
                            filtered_signals.append(signal)
                            logger.info(f"{candle_info}Accepted SHORT signal for BEARISH instrument")
                        else:
                            logger.info(f"{candle_info}Skipping signal {signal.type} at price {signal.price} because it does not match instrument direction {instrument_config['direction']}")
                    
                    strategy_signals = filtered_signals
                    
                    if strategy_signals:
                        signals.extend(strategy_signals)
                        logger.info(f"{candle_info}Generated {len(strategy_signals)} filtered signals")
                        
                        # Process each signal using trade manager
                        for signal in strategy_signals:
                            # Calculate position size
                            position_size = self.position_manager.calculate_position_size(
                                signal.price,
                                self.config.strategies[0].sl_percentage,
                                signal.direction.value
                            )
                            
                            # --------------------------
                            # TRADE CREATION PHASE
                            # --------------------------
                            entry_candle = candle_dict
                            trade = self.trade_manager.create_trade(
                                instrument_key=instrument_config.get('key'),
                                entry_price=signal.price,
                                position_size=position_size,
                                position_type=signal.direction.value,
                                entry_type=signal.metadata.get('entry_type'),
                                entry_time_string=signal.metadata.get('entry_time'),
                                trade_type=TradeType.IMMEDIATE_BREAKOUT if signal.type == SignalType.IMMEDIATE_BREAKOUT else TradeType.RETEST_ENTRY,
                                sl_percentage=self.config.strategies[0].sl_percentage,
                                candle_data=candle_dict
                            )
                            
                            
                            if trade:
                                logger.info(f"{candle_info}Created new trade for signal: {signal.type} at price {signal.price}")
                
                # --------------------------
                # TRADE MANAGEMENT PHASE
                # --------------------------
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
                                logger.info(f"{candle_info}Trade {instrument_key} closed with status {updated_trade['status']} and pnl {updated_trade['realized_pnl']}")
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

            # Day is over, reset the signal generator state
            self.signal_generator.reset_signal_and_state()
        
        # Calculate metrics for this strategy
        metrics = self._calculate_metrics(trades)
        
        return {
            'signals': signals,
            'trades': trades,
            'metrics': metrics
        }

    def _calculate_metrics(self, trades: List[Dict[str, Any]]) -> Dict[str, Any]:
        """
        Calculate backtest performance metrics using trade manager's statistics.
        
        Args:
            trades: List of trade dictionaries
            
        Returns:
            Dictionary containing performance metrics
        """
        # Get trade statistics from trade manager
        trade_stats = self.trade_manager.get_trade_statistics()
        
        # Calculate equity curve for drawdown calculation
        equity_curve = self._build_equity_curve(trades)
        
        # Calculate max drawdown from equity curve
        if not equity_curve.empty:
            rolling_max = equity_curve.expanding().max()
            drawdowns = (equity_curve - rolling_max) / rolling_max
            max_drawdown = abs(drawdowns.min())
        else:
            max_drawdown = 0.0
        
        # Calculate Sharpe ratio
        if trade_stats['total_trades'] > 0:
            returns = pd.Series([trade.get('realized_pnl', 0) for trade in trades])
            excess_returns = returns.mean() - (0.02 / 252)  # Daily risk-free rate
            sharpe_ratio = excess_returns / returns.std() if returns.std() != 0 else 0.0
        else:
            sharpe_ratio = 0.0
        
        import math

        # Get instrument direction
        # direction = instrument_config['direction'] if instrument_config else 'UNKNOWN'

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
            'max_drawdown': round(float(max_drawdown), 2),
            # 'direction': direction,  # Add direction to metrics
            # make sharpe ratio 0 if nan
            'sharpe_ratio': round(float(sharpe_ratio), 2) if not math.isnan(sharpe_ratio) else 0.0
        }

        logger.info(f"Calculated performance metrics: {metrics}")
        return metrics

    def _build_equity_curve(self, trades: List[Dict[str, Any]]) -> pd.Series:
        """
        Build equity curve from trade history.
        
        Args:
            trades: List of trade dictionaries
            
        Returns:
            Series representing equity curve
        """
        if not trades:
            return pd.Series([])
        
        # Create DataFrame from trade history
        trades_df = pd.DataFrame(trades)
        
        # Sort trades by exit time
        trades_df['exit_time'] = pd.to_datetime(trades_df['exit_time'])
        trades_df = trades_df.sort_values('exit_time')
        
        # Calculate cumulative P&L
        cumulative_pnl = trades_df['realized_pnl'].cumsum()
        
        # Add initial capital
        equity_curve = self.config.initial_capital + cumulative_pnl
        
        # Add direction information
        instrument_config = next(
            (inst for inst in self.config.instruments 
             if inst['key'] == self.config.strategies[0].instrument_key),
            None
        )
        if instrument_config:
            trades_df['direction'] = instrument_config['direction']
        
        logger.debug(f"Built equity curve with {len(equity_curve)} points")
        return equity_curve

    def _calculate_portfolio_metrics(self, results: Dict[str, Any]) -> Dict[str, Any]:
        """
        Calculate portfolio-level metrics from all instrument results.
        
        Args:
            results: Dictionary containing all backtest results
            
        Returns:
            Dictionary containing portfolio-level metrics
        """
        all_trades = results['trades']
        if not all_trades:
            return {
                'total_trades': 0,
                'win_rate': 0.0,
                'profit_factor': 0.0,
                'total_return': 0.0,
                'max_drawdown': 0.0,
                'sharpe_ratio': 0.0
            }
        
        # Calculate basic metrics
        winning_trades = [t for t in all_trades if t['realized_pnl'] > 0]
        losing_trades = [t for t in all_trades if t['realized_pnl'] <= 0]
        
        total_trades = len(all_trades)
        win_rate = len(winning_trades) / total_trades if total_trades > 0 else 0
        
        total_profit = sum(t['realized_pnl'] for t in winning_trades)
        total_loss = abs(sum(t['realized_pnl'] for t in losing_trades))
        profit_factor = total_profit / total_loss if total_loss > 0 else float('inf')
        
        # Calculate returns and Sharpe ratio
        returns = pd.Series([t['realized_pnl'] for t in all_trades])
        total_return = returns.sum() / self.config.initial_capital
        excess_returns = returns.mean() - (0.02 / 252)  # Daily risk-free rate
        sharpe_ratio = excess_returns / returns.std() if returns.std() != 0 else 0.0
        
        # Calculate max drawdown
        equity_curve = self._build_portfolio_equity_curve(results)
        if not equity_curve.empty:
            rolling_max = equity_curve.expanding().max()
            drawdowns = (equity_curve - rolling_max) / rolling_max
            max_drawdown = abs(drawdowns.min())
        else:
            max_drawdown = 0.0
        
        return {
            'total_trades': total_trades,
            'win_rate': win_rate,
            'profit_factor': profit_factor,
            'total_return': total_return,
            'max_drawdown': max_drawdown,
            'sharpe_ratio': sharpe_ratio,
            'total_profit': total_profit,
            'total_loss': total_loss,
            'average_win': total_profit / len(winning_trades) if winning_trades else 0,
            'average_loss': total_loss / len(losing_trades) if losing_trades else 0
        }

    def _build_portfolio_equity_curve(self, results: Dict[str, Any]) -> pd.Series:
        """
        Build portfolio equity curve from all instrument trades.
        
        Args:
            results: Dictionary containing all backtest results
            
        Returns:
            Series representing portfolio equity curve
        """
        all_trades = results['trades']
        if not all_trades:
            return pd.Series()
        
        # Create DataFrame from all trades
        trades_df = pd.DataFrame(all_trades)
        
        # Sort trades by exit time
        trades_df['exit_time'] = pd.to_datetime(trades_df['exit_time'])
        trades_df = trades_df.sort_values('exit_time')
        
        # Calculate cumulative P&L
        cumulative_pnl = trades_df['realized_pnl'].cumsum()
        
        # Add initial capital
        equity_curve = self.config.initial_capital + cumulative_pnl
        
        # Set index to exit times
        equity_curve.index = trades_df['exit_time']
        
        return equity_curve