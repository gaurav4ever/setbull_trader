"""
Core Backtesting Engine for Morning Range Strategy.

This module provides the core backtesting functionality with event-driven
architecture and multi-strategy support.
"""

from dataclasses import dataclass
from enum import Enum
from typing import Dict, List, Optional, Union, Tuple
import pandas as pd
import numpy as np
from datetime import datetime, time, timedelta
import logging
from concurrent.futures import ThreadPoolExecutor, as_completed
import pytz

from ..strategy.base_strategy import BaseStrategy, StrategyConfig
from ..data.data_processor import CandleProcessor
from ..strategy.position_manager import PositionManager
from ..strategy.trade_manager import TradeManager
from ..strategy.risk_calculator import RiskCalculator

logger = logging.getLogger(__name__)

class BacktestEvent(Enum):
    """Backtest event types."""
    MARKET_DATA = "market_data"
    RANGE_CALCULATED = "range_calculated"
    ENTRY_SIGNAL = "entry_signal"
    EXIT_SIGNAL = "exit_signal"
    POSITION_OPENED = "position_opened"
    POSITION_CLOSED = "position_closed"
    TRADE_UPDATE = "trade_update"
    ERROR = "error"

@dataclass
class BacktestConfig:
    """Configuration for backtest execution."""
    start_date: datetime
    end_date: datetime
    instruments: List[str]
    strategies: List[StrategyConfig]
    initial_capital: float
    position_size_type: str
    max_positions: int
    enable_parallel: bool = True
    cache_data: bool = True
    trading_hours: Dict[str, time] = None
    excluded_dates: List[datetime] = None

class BacktestEngine:
    """Core backtesting engine."""
    
    def __init__(self, config: BacktestConfig):
        """
        Initialize the Backtest Engine.
        
        Args:
            config: Backtest configuration
        """
        self.config = config
        self.data_processor = CandleProcessor()
        self.candle_cache: Dict[str, pd.DataFrame] = {}
        self.daily_cache: Dict[str, pd.DataFrame] = {}
        self.results: Dict[str, List] = {}
        self.events: List[Dict] = []
        
        # Initialize managers for each strategy
        self.strategy_managers: Dict[str, Dict] = {}
        self._initialize_strategy_managers()
        
        logger.info(f"Initialized BacktestEngine for {len(config.instruments)} instruments")

    def _initialize_strategy_managers(self):
        """Initialize managers for each strategy configuration."""
        for strategy_config in self.config.strategies:
            strategy_id = f"{strategy_config.instrument_key}_{strategy_config.range_type}_{strategy_config.entry_type}"
            
            # Create managers
            position_manager = PositionManager(
                account_info=self._create_account_info(),
                position_config=self._create_position_config()
            )
            
            trade_manager = TradeManager(
                trade_config=self._create_trade_config()
            )
            
            risk_calculator = RiskCalculator(
                risk_config=self._create_risk_config()
            )
            
            self.strategy_managers[strategy_id] = {
                'config': strategy_config,
                'position_manager': position_manager,
                'trade_manager': trade_manager,
                'risk_calculator': risk_calculator,
                'strategy': None  # Will be initialized during backtest
            }
            
        logger.info(f"Initialized {len(self.strategy_managers)} strategy managers")

    def _create_account_info(self) -> AccountInfo:
        """Create account information for backtesting."""
        return AccountInfo(
            total_capital=self.config.initial_capital,
            available_capital=self.config.initial_capital,
            max_position_size=self.config.initial_capital * 0.1,  # 10% max position size
            risk_per_trade=1.0,  # 1% risk per trade
            max_risk_per_trade=self.config.initial_capital * 0.01,
            currency="INR"
        )

    def _create_position_config(self) -> PositionSizeConfig:
        """Create position sizing configuration."""
        return PositionSizeConfig(
            size_type=PositionSizeType[self.config.position_size_type],
            value=1.0,  # 1% risk or account size
            min_size=1.0,
            max_size=float('inf'),
            round_to=0
        )

    def _create_trade_config(self) -> TradeConfig:
        """Create trade management configuration."""
        return TradeConfig(
            sl_percentage=0.5,  # 0.5% stop loss
            initial_target_r=2.0,  # 2R initial target
            breakeven_r=1.0,
            max_trade_duration=360,  # 6 hours
            entry_timeout=5,
            reentry_times=1,
            min_risk_reward=1.5
        )

    def _create_risk_config(self) -> RiskConfig:
        """Create risk management configuration."""
        return RiskConfig(
            max_risk_per_trade=1.0,
            max_daily_risk=3.0,
            max_correlated_risk=2.0,
            position_size_limit=5.0,
            max_drawdown_limit=10.0
        )

    async def load_data(self, instrument_key: str) -> Tuple[pd.DataFrame, pd.DataFrame]:
        """
        Load and prepare data for backtesting.
        
        Args:
            instrument_key: Instrument identifier
            
        Returns:
            Tuple[pd.DataFrame, pd.DataFrame]: (Intraday candles, Daily candles)
        """
        # Check cache first
        if self.config.cache_data and instrument_key in self.candle_cache:
            logger.info(f"Using cached data for {instrument_key}")
            return self.candle_cache[instrument_key], self.daily_cache[instrument_key]
        
        try:
            # Load intraday data
            intraday_candles = await self.data_processor.load_intraday_data(
                instrument_key,
                self.config.start_date,
                self.config.end_date
            )
            
            # Load daily data
            daily_candles = await self.data_processor.load_daily_data(
                instrument_key,
                self.config.start_date - timedelta(days=30),  # Extra days for indicators
                self.config.end_date
            )
            
            # Process and validate data
            intraday_candles = self.data_processor.process_candles(intraday_candles)
            daily_candles = self.data_processor.process_candles(daily_candles)
            
            # Cache if enabled
            if self.config.cache_data:
                self.candle_cache[instrument_key] = intraday_candles
                self.daily_cache[instrument_key] = daily_candles
            
            logger.info(f"Loaded data for {instrument_key}: {len(intraday_candles)} intraday candles, {len(daily_candles)} daily candles")
            return intraday_candles, daily_candles
            
        except Exception as e:
            logger.error(f"Error loading data for {instrument_key}: {str(e)}")
            return pd.DataFrame(), pd.DataFrame()

    def _filter_trading_day_candles(self, 
                                  candles: pd.DataFrame,
                                  date: datetime) -> pd.DataFrame:
        """Filter candles for a specific trading day."""
        if candles.empty:
            return pd.DataFrame()
        
        # Convert to datetime if timestamp is string
        if isinstance(candles['timestamp'].iloc[0], str):
            candles['timestamp'] = pd.to_datetime(candles['timestamp'])
        
        # Filter by date
        day_candles = candles[candles['timestamp'].dt.date == date.date()]
        
        # Apply trading hours filter if configured
        if self.config.trading_hours:
            market_start = self.config.trading_hours['start']
            market_end = self.config.trading_hours['end']
            
            day_candles = day_candles[
                (day_candles['timestamp'].dt.time >= market_start) &
                (day_candles['timestamp'].dt.time <= market_end)
            ]
        
        return day_candles

    def _is_valid_trading_day(self, date: datetime) -> bool:
        """Check if date is a valid trading day."""
        # Check if weekend
        if date.weekday() in [5, 6]:  # Saturday, Sunday
            return False
        
        # Check if excluded date
        if self.config.excluded_dates and date.date() in self.config.excluded_dates:
            return False
        
        return True

    async def run_strategy_backtest(self,
                                  strategy_id: str,
                                  candles: pd.DataFrame,
                                  daily_candles: pd.DataFrame) -> List[Dict]:
        """Run backtest for a single strategy."""
        if candles.empty:
            logger.warning(f"No data available for {strategy_id}")
            return []
        
        strategy_config = self.strategy_managers[strategy_id]['config']
        position_manager = self.strategy_managers[strategy_id]['position_manager']
        trade_manager = self.strategy_managers[strategy_id]['trade_manager']
        risk_calculator = self.strategy_managers[strategy_id]['risk_calculator']
        
        # Create strategy instance
        strategy = MorningRangeStrategy(
            config=strategy_config,
            position_manager=position_manager,
            trade_manager=trade_manager,
            risk_calculator=risk_calculator
        )
        
        trades = []
        current_date = self.config.start_date
        
        while current_date <= self.config.end_date:
            if not self._is_valid_trading_day(current_date):
                current_date += timedelta(days=1)
                continue
            
            # Get candles for the day
            day_candles = self._filter_trading_day_candles(candles, current_date)
            if day_candles.empty:
                current_date += timedelta(days=1)
                continue
            
            # Calculate morning range
            mr_values = strategy.calculate_morning_range(day_candles)
            if mr_values:
                self.events.append({
                    'timestamp': current_date,
                    'type': BacktestEvent.RANGE_CALCULATED.value,
                    'strategy_id': strategy_id,
                    'data': mr_values
                })
                
                # Calculate entry levels
                entry_levels = strategy.calculate_entry_levels()
                
                # Process each candle
                for _, candle in day_candles.iterrows():
                    result = strategy.process_candle(candle.to_dict())
                    
                    if result['action'] in ['entry', 'exit']:
                        trades.append(result)
                        self.events.append({
                            'timestamp': candle['timestamp'],
                            'type': BacktestEvent.TRADE_UPDATE.value,
                            'strategy_id': strategy_id,
                            'data': result
                        })
            
            current_date += timedelta(days=1)
        
        logger.info(f"Completed backtest for {strategy_id}: {len(trades)} trades executed")
        return trades

    async def run_backtest(self) -> Dict:
        """Run backtest for all configured strategies."""
        all_results = {}
        
        if self.config.enable_parallel:
            # Run parallel backtests
            with ThreadPoolExecutor() as executor:
                futures = []
                
                for instrument_key in self.config.instruments:
                    # Load data
                    candles, daily_candles = await self.load_data(instrument_key)
                    
                    # Create futures for each strategy
                    for strategy_id in self.strategy_managers:
                        if strategy_id.startswith(instrument_key):
                            future = executor.submit(
                                self.run_strategy_backtest,
                                strategy_id,
                                candles,
                                daily_candles
                            )
                            futures.append((strategy_id, future))
                
                # Collect results
                for strategy_id, future in futures:
                    try:
                        trades = future.result()
                        all_results[strategy_id] = trades
                    except Exception as e:
                        logger.error(f"Error in backtest for {strategy_id}: {str(e)}")
                        all_results[strategy_id] = []
        else:
            # Run sequential backtests
            for instrument_key in self.config.instruments:
                # Load data
                candles, daily_candles = await self.load_data(instrument_key)
                
                # Run each strategy
                for strategy_id in self.strategy_managers:
                    if strategy_id.startswith(instrument_key):
                        trades = await self.run_strategy_backtest(
                            strategy_id,
                            candles,
                            daily_candles
                        )
                        all_results[strategy_id] = trades
        
        self.results = all_results
        return self.generate_backtest_report()

    def generate_backtest_report(self) -> Dict:
        """Generate comprehensive backtest report."""
        report = {
            'summary': self._generate_summary(),
            'strategy_results': self._generate_strategy_results(),
            'risk_metrics': self._generate_risk_metrics(),
            'equity_curve': self._generate_equity_curve(),
            'trade_list': self._generate_trade_list(),
            'events': self.events
        }
        
        logger.info("Generated backtest report")
        return report

    def _generate_summary(self) -> Dict:
        """Generate overall backtest summary."""
        total_trades = sum(len(trades) for trades in self.results.values())
        total_profit = sum(
            sum(t['result']['realized_pnl'] for t in trades if t['action'] == 'exit')
            for trades in self.results.values()
        )
        
        return {
            'period_start': self.config.start_date,
            'period_end': self.config.end_date,
            'instruments': len(self.config.instruments),
            'strategies': len(self.strategy_managers),
            'total_trades': total_trades,
            'total_profit': total_profit,
            'initial_capital': self.config.initial_capital,
            'final_capital': self.config.initial_capital + total_profit
        }

    def _generate_strategy_results(self) -> Dict:
        """Generate results for each strategy."""
        strategy_results = {}
        
        for strategy_id, trades in self.results.items():
            # Calculate strategy metrics
            winning_trades = [t for t in trades if t['action'] == 'exit' and t['result']['realized_pnl'] > 0]
            losing_trades = [t for t in trades if t['action'] == 'exit' and t['result']['realized_pnl'] <= 0]
            
            strategy_results[strategy_id] = {
                'total_trades': len(trades),
                'winning_trades': len(winning_trades),
                'losing_trades': len(losing_trades),
                'win_rate': len(winning_trades) / len(trades) if trades else 0,
                'avg_profit': np.mean([t['result']['realized_pnl'] for t in winning_trades]) if winning_trades else 0,
                'avg_loss': np.mean([t['result']['realized_pnl'] for t in losing_trades]) if losing_trades else 0,
                'largest_win': max([t['result']['realized_pnl'] for t in winning_trades], default=0),
                'largest_loss': min([t['result']['realized_pnl'] for t in losing_trades], default=0)
            }
        
        return strategy_results

    def _generate_risk_metrics(self) -> Dict:
        """Generate risk metrics for the backtest."""
        all_trades = []
        for trades in self.results.values():
            all_trades.extend([t for t in trades if t['action'] == 'exit'])
        
        if not all_trades:
            return {}
        
        returns = [t['result']['realized_pnl'] for t in all_trades]
        
        return {
            'sharpe_ratio': self._calculate_sharpe_ratio(returns),
            'sortino_ratio': self._calculate_sortino_ratio(returns),
            'max_drawdown': self._calculate_max_drawdown(returns),
            'profit_factor': self._calculate_profit_factor(returns),
            'risk_reward_ratio': self._calculate_risk_reward_ratio(returns)
        }

    def _generate_equity_curve(self) -> pd.DataFrame:
        """Generate equity curve data."""
        equity_data = []
        current_equity = self.config.initial_capital
        
        # Combine all trades and sort by timestamp
        all_trades = []
        for strategy_id, trades in self.results.items():
            for trade in trades:
                if trade['action'] == 'exit':
                    all_trades.append({
                        'timestamp': trade['result']['exit_time'],
                        'pnl': trade['result']['realized_pnl'],
                        'strategy_id': strategy_id
                    })
        
        all_trades.sort(key=lambda x: x['timestamp'])
        
        # Create equity curve
        for trade in all_trades:
            current_equity += trade['pnl']
            equity_data.append({
                'timestamp': trade['timestamp'],
                'equity': current_equity,
                'strategy_id': trade['strategy_id']
            })
        
        return pd.DataFrame(equity_data)

    def _generate_trade_list(self) -> List[Dict]:
        """Generate detailed trade list."""
        trade_list = []
        
        for strategy_id, trades in self.results.items():
            for trade in trades:
                if trade['action'] == 'exit':
                    trade_list.append({
                        'strategy_id': strategy_id,
                        'entry_time': trade['result']['entry_time'],
                        'exit_time': trade['result']['exit_time'],
                        'position_type': trade['result']['position_type'],
                        'entry_price': trade['result']['entry_price'],
                        'exit_price': trade['result']['exit_price'],
                        'position_size': trade['result']['position_size'],
                        'pnl': trade['result']['realized_pnl'],
                        'r_multiple': trade['result'].get('r_multiple', 0),
                        'exit_reason': trade['result']['status']
                    })
        
        return sorted(trade_list, key=lambda x: x['entry_time']) 