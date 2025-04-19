"""
Test module for the backtest engine.

This module contains tests for the Backtrader-based backtest engine.
"""

import pytest
import pandas as pd
import numpy as np
from datetime import datetime, time, timedelta
import logging
import asyncio

from ..backtest.engine import BacktestEngine

logger = logging.getLogger(__name__)

@pytest.fixture
def backtest_engine():
    """Create a backtest engine instance for testing."""
    return BacktestEngine()

@pytest.fixture
def sample_data():
    """Create sample data for testing."""
    # Create a date range
    dates = pd.date_range(start='2024-01-01 09:15:00', end='2024-01-01 15:30:00', freq='5min')
    
    # Create sample OHLCV data
    data = {
        'timestamp': dates,
        'open': np.random.uniform(100, 200, len(dates)),
        'high': np.random.uniform(200, 300, len(dates)),
        'low': np.random.uniform(50, 100, len(dates)),
        'close': np.random.uniform(100, 200, len(dates)),
        'volume': np.random.randint(1000, 10000, len(dates))
    }
    
    return pd.DataFrame(data)

@pytest.mark.asyncio
async def test_run_backtest(backtest_engine, sample_data):
    """Test running a backtest."""
    # Run backtest
    results = await backtest_engine.run_backtest(
        instrument_key='NSE:RELIANCE',
        start_date='2024-01-01',
        end_date='2024-01-01',
        timeframe='5minute'
    )
    
    # Verify results
    assert isinstance(results, dict)
    assert 'total_return' in results
    assert 'sharpe_ratio' in results
    assert 'max_drawdown' in results
    assert 'total_trades' in results
    assert 'win_rate' in results
    
    # Verify metrics are within expected ranges
    assert -100 <= results['total_return'] <= 100  # Return should be between -100% and 100%
    assert results['max_drawdown'] <= 100  # Drawdown should be less than 100%
    assert 0 <= results['win_rate'] <= 100  # Win rate should be between 0% and 100%

@pytest.mark.asyncio
async def test_empty_data(backtest_engine):
    """Test running backtest with empty data."""
    results = await backtest_engine.run_backtest(
        instrument_key='NSE:RELIANCE',
        start_date='2024-01-01',
        end_date='2024-01-01',
        timeframe='5minute'
    )
    
    assert results == {}

@pytest.mark.asyncio
async def test_invalid_dates(backtest_engine):
    """Test running backtest with invalid dates."""
    with pytest.raises(Exception):
        await backtest_engine.run_backtest(
            instrument_key='NSE:RELIANCE',
            start_date='invalid_date',
            end_date='2024-01-01',
            timeframe='5minute'
        )

@pytest.mark.asyncio
async def test_different_timeframes(backtest_engine):
    """Test running backtest with different timeframes."""
    timeframes = ['5minute', '15minute', '30minute', '1hour']
    
    for timeframe in timeframes:
        results = await backtest_engine.run_backtest(
            instrument_key='NSE:RELIANCE',
            start_date='2024-01-01',
            end_date='2024-01-01',
            timeframe=timeframe
        )
        
        assert isinstance(results, dict)
        assert 'total_return' in results
        assert 'total_trades' in results 