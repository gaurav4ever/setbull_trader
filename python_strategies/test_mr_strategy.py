"""
Morning Range Strategy Backtest Runner

This script tests the Morning Range strategy with different entry types
and compares their performance using the Backtrader-based engine.
"""

import asyncio
import pandas as pd
import matplotlib.pyplot as plt
import pytz
from datetime import datetime, time, timedelta
import logging
import os
import pytest
import pytest_asyncio
from pathlib import Path
import numpy as np

from mr_strategy.backtest.runner import BacktestRunner
from mr_strategy.backtest.engine import BacktestEngine

# Register the asyncio mark
pytestmark = pytest.mark.asyncio

print(">> Script Started")

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler("backtest_run.log"),
        logging.StreamHandler()
    ]
)

logger = logging.getLogger(__name__)

# Test parameters
INSTRUMENT_CONFIGS = [
    {
        "name": "OLAELEC",
        "direction": "BEARISH",
        "key": "NSE_EQ|INE0LXG01040"
    },
    {
        "name": "GPTINFRA",
        "direction": "BULLISH",
        "key": "NSE_EQ|INE390G01014"
    },
    {
        "name": "BALUFORGE",
        "direction": "BULLISH",
        "key": "NSE_EQ|INE011E01029"
    }
]

START_DATE = "2024-04-01T09:15:00+05:30"
END_DATE = "2024-04-11T15:25:00+05:30"
INITIAL_CAPITAL = 100000.0

# Strategy parameters to test
STRATEGY_PARAMS = [
    {
        "range_type": "5MR",
        "entry_type": "1ST_ENTRY",
        "sl_percentage": 0.75,
        "target_r": 7.0
    },
    {
        "range_type": "5MR",
        "entry_type": "1ST_ENTRY",
        "sl_percentage": 1.0,
        "target_r": 5.0
    }
]

@pytest_asyncio.fixture
async def backtest_runner():
    """Create a backtest runner instance for testing."""
    runner = BacktestRunner({
        "initial_capital": INITIAL_CAPITAL,
        "commission": 0.001,  # 0.1% commission
        "slippage": 0.001,    # 0.1% slippage
        "strategy_params": STRATEGY_PARAMS[0]
    })
    return runner

async def test_single_backtest(backtest_runner):
    """Test running a single backtest."""
    # Run backtest for a single instrument
    results = await backtest_runner.run_single_backtest(
        instrument_key=INSTRUMENT_CONFIGS[0]["key"],
        start_date=START_DATE,
        end_date=END_DATE,
        timeframe="5minute"
    )
    
    # Verify basic results structure
    assert isinstance(results, dict)
    assert "total_return" in results
    assert "total_trades" in results
    assert "win_rate" in results
    assert "max_drawdown" in results
    
    # Verify metrics are within expected ranges
    assert -100 <= results["total_return"] <= 100
    assert results["total_trades"] >= 0
    assert 0 <= results["win_rate"] <= 100
    assert 0 <= results["max_drawdown"] <= 100

# async def test_parallel_backtests(backtest_runner):
#     """Test running multiple backtests in parallel."""
#     # Run backtests for all instruments
#     results = await backtest_runner.run_parallel_backtests(
#         instruments=INSTRUMENT_CONFIGS,
#         start_date=START_DATE,
#         end_date=END_DATE,
#         timeframe="5minute"
#     )
    
#     # Verify results structure
#     assert isinstance(results, dict)
#     assert len(results) == len(INSTRUMENT_CONFIGS)
    
#     # Verify each instrument's results
#     for inst_key, inst_results in results.items():
#         assert isinstance(inst_results, dict)
#         assert "total_return" in inst_results
#         assert "total_trades" in inst_results
#         assert "win_rate" in inst_results

# async def test_strategy_parameters(backtest_runner):
#     """Test the strategy with different parameters."""
#     for params in STRATEGY_PARAMS:
#         # Update runner configuration
#         backtest_runner.config["strategy_params"] = params
        
#         # Run backtest
#         results = await backtest_runner.run_single_backtest(
#             instrument_key=INSTRUMENT_CONFIGS[0]["key"],
#             start_date=START_DATE,
#             end_date=END_DATE,
#             timeframe="5minute"
#         )
        
#         # Verify results
#         assert isinstance(results, dict)
#         assert "max_drawdown" in results
#         assert results["max_drawdown"] <= params["sl_percentage"] * 100

# async def test_market_hours(backtest_runner):
#     """Test strategy behavior during market hours."""
#     # Test with different time ranges
#     time_ranges = [
#         ("2024-04-01T09:15:00+05:30", "2024-04-01T09:20:00+05:30"),  # Morning range
#         ("2024-04-01T09:20:00+05:30", "2024-04-01T15:30:00+05:30"),  # Trading hours
#         ("2024-04-01T15:30:00+05:30", "2024-04-01T15:35:00+05:30")   # After market close
#     ]
    
#     for start_time, end_time in time_ranges:
#         results = await backtest_runner.run_single_backtest(
#             instrument_key=INSTRUMENT_CONFIGS[0]["key"],
#             start_date=start_time,
#             end_date=end_time,
#             timeframe="5minute"
#         )
        
#         assert isinstance(results, dict)
#         assert "total_trades" in results
        
#         # Verify trade timing
#         if "trades" in results:
#             for trade in results["trades"]:
#                 trade_time = pd.to_datetime(trade["timestamp"]).time()
#                 if start_time.endswith("09:20:00+05:30"):
#                     assert trade_time >= time(9, 20)  # Trades should be after morning range
#                 elif start_time.endswith("15:30:00+05:30"):
#                     assert trade_time <= time(15, 30)  # No trades after market close

# async def test_results_aggregation(backtest_runner):
    """Test results aggregation functionality."""
    # Run parallel backtests
    results = await backtest_runner.run_parallel_backtests(
        instruments=INSTRUMENT_CONFIGS,
        start_date=START_DATE,
        end_date=END_DATE,
        timeframe="5minute"
    )
    
    # Aggregate results
    aggregated = backtest_runner.aggregate_results(results)
    
    # Verify aggregated metrics
    assert isinstance(aggregated, dict)
    assert "total_return" in aggregated
    assert "total_trades" in aggregated
    assert "win_rate" in aggregated
    assert "profit_factor" in aggregated
    assert "max_drawdown" in aggregated
    
    # Verify aggregation calculations
    total_trades = sum(r["total_trades"] for r in results.values())
    assert aggregated["total_trades"] == total_trades
    
    # Verify report generation
    report = backtest_runner.generate_report(results)
    assert isinstance(report, str)
    assert "Backtest Results Report" in report
    assert "Aggregated Results" in report

if __name__ == "__main__":
    # Run all tests
    pytest.main([__file__, "-v"])
