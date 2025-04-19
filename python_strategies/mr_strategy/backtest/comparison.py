"""
Entry Type Comparison Module

This module provides functionality to compare different entry types
for the Morning Range strategy using the Backtrader engine.
"""

import asyncio
import logging
from typing import List, Dict, Any
import pandas as pd
import numpy as np
from datetime import datetime

from .runner import BacktestRunner

logger = logging.getLogger(__name__)

async def run_entry_type_comparison(
    instrument_configs: List[Dict[str, Any]],
    start_date: str = "2024-04-01T09:15:00+05:30",
    end_date: str = "2024-04-11T15:25:00+05:30",
    initial_capital: float = 100000.0,
    entry_types: List[str] = ["1ST_ENTRY", "2ND_ENTRY"]
) -> Dict[str, Any]:
    """
    Run backtest comparison for different entry types.
    
    Args:
        instrument_configs: List of instrument configurations
        start_date: Start date for backtest
        end_date: End date for backtest
        initial_capital: Initial capital for backtest
        entry_types: List of entry types to compare
        
    Returns:
        Dictionary containing comparison results
    """
    try:
        logger.info("Starting entry type comparison")
        
        # Initialize results dictionary
        results = {
            "metrics": {},
            "trades": {},
            "entry_types": entry_types
        }
        
        # Run backtest for each entry type
        for entry_type in entry_types:
            logger.info(f"Running backtest for entry type: {entry_type}")
            
            # Create backtest runner
            runner = BacktestRunner({
                "initial_capital": initial_capital,
                "commission": 0.001,
                "slippage": 0.001,
                "strategy_params": {
                    "range_type": "5MR",
                    "entry_type": entry_type,
                    "sl_percentage": 0.75,
                    "target_r": 7.0
                }
            })
            
            # Run parallel backtests
            backtest_results = await runner.run_parallel_backtests(
                instruments=instrument_configs,
                start_date=start_date,
                end_date=end_date,
                timeframe="5minute"
            )
            
            # Aggregate results
            aggregated = runner.aggregate_results(backtest_results)
            
            # Store results
            results["metrics"][entry_type] = aggregated
            results["trades"][entry_type] = backtest_results
            
            logger.info(f"Completed backtest for entry type: {entry_type}")
        
        logger.info("Entry type comparison completed successfully")
        return results
        
    except Exception as e:
        logger.error(f"Error in entry type comparison: {str(e)}")
        raise 