"""
Backtest engine for the Morning Range strategy.

This module implements the backtesting engine using Backtrader,
while maintaining compatibility with existing configuration and metrics.
"""

import backtrader as bt
import pandas as pd
import numpy as np
from typing import Dict, List, Optional, Any, Tuple
import logging
from datetime import datetime, time, timedelta
import pytz

from ..data.data_processor import CandleProcessor
from ..signals.signal_generator import SignalGenerator
from .mr_backtrader_strategy import MorningRangeStrategy
# from .mr_backtrader_data import MorningRangeDataFeed

logger = logging.getLogger(__name__)

class BacktestEngine:
    """
    Backtest engine that uses Backtrader for strategy execution.
    
    This engine:
    1. Loads and processes historical data
    2. Sets up Backtrader cerebro with the strategy
    3. Runs the backtest and collects results
    4. Calculates performance metrics
    """
    
    def __init__(self, config: Optional[Dict] = None):
        """
        Initialize the backtest engine.
        
        Args:
            config: Optional configuration dictionary
        """
        self.config = config or {}
        self.data_processor = CandleProcessor(config)
        self.cerebro = bt.Cerebro()
        
        # Set up cerebro
        self.cerebro.broker.setcash(100000.0)  # Initial cash
        self.cerebro.broker.setcommission(commission=0.001)  # 0.1% commission
        
        # Add analyzers
        self.cerebro.addanalyzer(bt.analyzers.SharpeRatio, _name='sharpe')
        self.cerebro.addanalyzer(bt.analyzers.DrawDown, _name='drawdown')
        self.cerebro.addanalyzer(bt.analyzers.Returns, _name='returns')
        self.cerebro.addanalyzer(bt.analyzers.TradeAnalyzer, _name='trades')
        
        logger.info("Initialized BacktestEngine")
        
    async def run_backtest(
        self,
        instrument_key: str,
        name: str,
        start_date: str,
        end_date: str,
        timeframe: str = '5minute'
    ) -> Dict[str, Any]:
        """
        Run a backtest for the given instrument and date range.
        
        Args:
            instrument_key: Instrument key
            start_date: Start date in ISO format
            end_date: End date in ISO format
            timeframe: Timeframe for candles (default: 5minute)
            
        Returns:
            Dictionary containing backtest results and metrics
        """
        try:
            # Load and process data
            processed_df, datafeed = await self.data_processor.load_and_process_candles(
                instrument_key=instrument_key,
                name=name,
                start_date=start_date,
                end_date=end_date,
                timeframe=timeframe
            )
            
            if processed_df.empty or datafeed is None:
                logger.error("No data available for backtest")
                return {}
            
            logger.info(f"Datafeed: {datafeed}, saving to csv")
            datafeed._dataname.to_csv('/Users/gaurav/setbull_projects/setbull_trader/python_strategies/results/datafeed/datafeed.csv')
                
            # Add data to cerebro
            self.cerebro.adddata(datafeed)
            
            # Add strategy with parameters
            self.cerebro.addstrategy(
                MorningRangeStrategy,
                mr_start_time=time(9, 15),
                mr_end_time=time(9, 20),
                market_close_time=time(15, 20),
                stop_loss_pct=0.005,
                target_pct=0.02,
                position_size=1,
                risk_per_trade=50,
                use_daily_indicators=True
            )
            
            # Run the backtest
            logger.info("Starting backtest...")
            results = self.cerebro.run()
            strategy = results[0]
            
            # Calculate metrics
            metrics = self._calculate_metrics(strategy, processed_df)
            
            logger.info("Backtest completed successfully")
            return metrics
            
        except Exception as e:
            logger.error(f"Error running backtest: {str(e)}")
            raise
            
    def _calculate_metrics(self, strategy: MorningRangeStrategy, df: pd.DataFrame) -> Dict[str, Any]:
        """
        Calculate performance metrics from the backtest results.
        
        Args:
            strategy: The strategy instance after backtest
            df: The processed DataFrame used in the backtest
            
        Returns:
            Dictionary containing performance metrics
        """
        # Get analyzer results
        sharpe = strategy.analyzers.sharpe.get_analysis()
        drawdown = strategy.analyzers.drawdown.get_analysis()
        returns = strategy.analyzers.returns.get_analysis()
        trades = strategy.analyzers.trades.get_analysis()
        logger.info(f"Trades analysis: {strategy.analyzers.trades.get_analysis()}")
        logger.info(f"Sharpe analysis: {strategy.analyzers.sharpe.get_analysis()}")
        logger.info(f"Drawdown analysis: {strategy.analyzers.drawdown.get_analysis()}")
        logger.info(f"Returns analysis: {strategy.analyzers.returns.get_analysis()}")
        
        
        # Calculate additional metrics
        total_trades = trades.total.closed
        winning_trades = trades.won.total
        losing_trades = trades.lost.total
        win_rate = winning_trades / total_trades if total_trades > 0 else 0
        
        # Calculate average trade metrics
        avg_win = trades.won.pnl.average if winning_trades > 0 else 0
        avg_loss = trades.lost.pnl.average if losing_trades > 0 else 0
        profit_factor = abs(avg_win / avg_loss) if avg_loss != 0 else float('inf')
        
        # Calculate time-based metrics
        start_date = df['timestamp'].min()
        end_date = df['timestamp'].max()
        total_days = (end_date - start_date).days
        
        # Compile metrics
        metrics = {
            'total_return': returns['rtot'] * 100,  # Convert to percentage
            'sharpe_ratio': sharpe['sharperatio'] if sharpe['sharperatio'] != None else 0,
            'max_drawdown': drawdown['max']['drawdown'] if drawdown['max']['drawdown'] != None else 0,
            'total_trades': total_trades,
            'winning_trades': winning_trades,
            'losing_trades': losing_trades,
            'win_rate': win_rate * 100,  # Convert to percentage
            'profit_factor': profit_factor if profit_factor != float('inf') else 0,
            'avg_win': avg_win,
            'avg_loss': avg_loss,
            'total_days': total_days,
            'start_date': start_date,
            'end_date': end_date
        }
        
        logger.info(f"Calculated metrics: {metrics}")
        return metrics