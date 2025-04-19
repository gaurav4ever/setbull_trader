"""
Backtest runner for the Morning Range strategy.

This module provides a high-level interface for running backtests
using the Backtrader-based engine.
"""

import asyncio
import logging
from typing import Dict, List, Optional, Any
from datetime import datetime, time, timedelta
import pandas as pd
import numpy as np

from ..backtest.engine import BacktestEngine

logger = logging.getLogger(__name__)

class BacktestRunner:
    """
    Runner for executing backtests with the Morning Range strategy.
    
    This class provides methods for:
    1. Running single backtests
    2. Running multiple backtests in parallel
    3. Aggregating results
    4. Generating reports
    """
    
    def __init__(self, config: Optional[Dict] = None):
        """
        Initialize the backtest runner.
        
        Args:
            config: Optional configuration dictionary
        """
        self.config = config or {}
        self.engine = BacktestEngine(config)
        
    async def run_single_backtest(
        self,
        instrument_key: str,
        name: str,
        start_date: str,
        end_date: str,
        timeframe: str = '5minute'
    ) -> Dict[str, Any]:
        """
        Run a single backtest for the given instrument and date range.
        
        Args:
            instrument_key: Instrument key
            start_date: Start date in ISO format
            end_date: End date in ISO format
            timeframe: Timeframe for candles (default: 5minute)
            
        Returns:
            Dictionary containing backtest results and metrics
        """
        try:
            logger.info(f"Starting backtest for {instrument_key} from {start_date} to {end_date}")
            
            # Run the backtest
            results = await self.engine.run_backtest(
                instrument_key=instrument_key,
                name=name,
                start_date=start_date,
                end_date=end_date,
                timeframe=timeframe
            )
            
            if not results:
                logger.warning(f"No results for {name}")
                return {}
                
            logger.info(f"Completed backtest for {name}")
            return results
            
        except Exception as e:
            logger.exception(f"Error running backtest for {name}: {str(e)}")
            raise
            
    async def run_parallel_backtests(
        self,
        instruments: List[Dict[str, str]],
        start_date: str,
        end_date: str,
        timeframe: str = '5minute'
    ) -> Dict[str, Dict[str, Any]]:
        """
        Run multiple backtests in parallel.
        
        Args:
            instruments: List of instrument dictionaries with 'key' and 'direction'
            start_date: Start date in ISO format
            end_date: End date in ISO format
            timeframe: Timeframe for candles (default: 5minute)
            
        Returns:
            Dictionary mapping instrument keys to their backtest results
        """
        try:
            # Create tasks for each instrument
            tasks = [
                self.run_single_backtest(
                    instrument_key=inst['key'],
                    start_date=start_date,
                    end_date=end_date,
                    timeframe=timeframe
                )
                for inst in instruments
            ]
            
            # Run all tasks in parallel
            results = await asyncio.gather(*tasks, return_exceptions=True)
            
            # Process results
            processed_results = {}
            for inst, result in zip(instruments, results):
                if isinstance(result, Exception):
                    logger.error(f"Error in backtest for {inst['key']}: {str(result)}")
                    continue
                processed_results[inst['key']] = result
                
            return processed_results
            
        except Exception as e:
            logger.error(f"Error running parallel backtests: {str(e)}")
            raise
            
    def aggregate_results(self, results: Dict[str, Dict[str, Any]]) -> Dict[str, Any]:
        """
        Aggregate results from multiple backtests.
        
        Args:
            results: Dictionary mapping instrument keys to their backtest results
            
        Returns:
            Dictionary containing aggregated metrics
        """
        if not results:
            return {}
            
        # Initialize aggregated metrics
        aggregated = {
            'total_return': 0.0,
            'total_trades': 0,
            'winning_trades': 0,
            'losing_trades': 0,
            'total_profit': 0.0,
            'total_loss': 0.0,
            'max_drawdown': 0.0,
            'instruments': len(results)
        }
        
        # Aggregate metrics
        for inst_results in results.values():
            aggregated['total_return'] += inst_results.get('total_return', 0)
            aggregated['total_trades'] += inst_results.get('total_trades', 0)
            aggregated['winning_trades'] += inst_results.get('winning_trades', 0)
            aggregated['losing_trades'] += inst_results.get('losing_trades', 0)
            aggregated['total_profit'] += inst_results.get('avg_win', 0) * inst_results.get('winning_trades', 0)
            aggregated['total_loss'] += abs(inst_results.get('avg_loss', 0)) * inst_results.get('losing_trades', 0)
            aggregated['max_drawdown'] = max(aggregated['max_drawdown'], inst_results.get('max_drawdown', 0))
            
        # Calculate derived metrics
        if aggregated['total_trades'] > 0:
            aggregated['win_rate'] = (aggregated['winning_trades'] / aggregated['total_trades']) * 100
            aggregated['profit_factor'] = aggregated['total_profit'] / aggregated['total_loss'] if aggregated['total_loss'] > 0 else float('inf')
        else:
            aggregated['win_rate'] = 0.0
            aggregated['profit_factor'] = 0.0
            
        return aggregated
        
    def generate_report(self, results: Dict[str, Dict[str, Any]]) -> str:
        """
        Generate a human-readable report from backtest results.
        
        Args:
            results: Dictionary mapping instrument keys to their backtest results
            
        Returns:
            Formatted report string
        """
        if not results:
            return "No results to report"
            
        # Generate report header
        report = [
            "Backtest Results Report",
            "=====================",
            f"Total Instruments: {len(results)}",
            ""
        ]
        
        # Add results for each instrument
        for inst_key, inst_results in results.items():
            report.extend([
                f"Instrument: {inst_key}",
                "-" * 50,
                f"Total Return: {inst_results.get('total_return', 0):.2f}%",
                f"Total Trades: {inst_results.get('total_trades', 0)}",
                f"Win Rate: {inst_results.get('win_rate', 0):.2f}%",
                f"Profit Factor: {inst_results.get('profit_factor', 0):.2f}",
                f"Max Drawdown: {inst_results.get('max_drawdown', 0):.2f}%",
                ""
            ])
            
        # Add aggregated results
        aggregated = self.aggregate_results(results)
        report.extend([
            "Aggregated Results",
            "-" * 50,
            f"Total Return: {aggregated['total_return']:.2f}%",
            f"Total Trades: {aggregated['total_trades']}",
            f"Win Rate: {aggregated['win_rate']:.2f}%",
            f"Profit Factor: {aggregated['profit_factor']:.2f}",
            f"Max Drawdown: {aggregated['max_drawdown']:.2f}%"
        ])
        
        return "\n".join(report)
