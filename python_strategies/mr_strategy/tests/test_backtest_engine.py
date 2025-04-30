"""
Tests for backtest engine with different entry strategies.
"""

import pytest
from datetime import datetime, time
import pandas as pd
import numpy as np

from ..backtest.engine import BacktestEngine
from ..strategy.models import SignalType, SignalDirection
from ..strategy.config import MRStrategyConfig

@pytest.fixture
def config():
    """Create test configuration."""
    return {
        'buffer_ticks': 5,
        'tick_size': 0.05,
        'breakout_percentage': 0.003,
        'invalidation_percentage': 0.005,
        'range_type': '5MR',
        'respect_trend': True,
        'entry_type': '1ST_ENTRY',
        'initial_capital': 100000,
        'position_size_type': 'FIXED',
        'max_positions': 1
    }

@pytest.fixture
def sample_data():
    """Create sample data for testing."""
    # Create a DataFrame with multiple candles
    candles = pd.DataFrame([
        {
            'timestamp': pd.Timestamp('2024-04-15 09:15:00'),
            'open': 95.0,
            'high': 100.0,
            'low': 90.0,
            'close': 95.0,
            'DAILY_ATR_14': 5.0
        },
        {
            'timestamp': pd.Timestamp('2024-04-15 09:30:00'),
            'open': 95.0,
            'high': 96.0,
            'low': 94.0,
            'close': 95.5,
            'DAILY_ATR_14': 5.0
        },
        {
            'timestamp': pd.Timestamp('2024-04-15 09:35:00'),
            'open': 95.5,
            'high': 101.0,  # Breakout
            'low': 95.0,
            'close': 100.5,
            'DAILY_ATR_14': 5.0
        }
    ])
    
    return {'RELIANCE': candles}

@pytest.mark.asyncio
async def test_first_entry_backtest(config, sample_data):
    """Test backtest with first entry strategy."""
    # Configure for first entry
    config['entry_type'] = '1ST_ENTRY'
    engine = BacktestEngine(config)
    
    # Run backtest
    results = await engine.run_backtest(sample_data)
    
    # Verify results
    assert results['status'] == 'success'
    assert len(results['signals']) > 0
    
    # Check signal properties
    signal = results['signals'][0]
    assert signal.type == SignalType.IMMEDIATE_BREAKOUT
    assert signal.direction == SignalDirection.LONG
    
    # Verify trade creation
    assert len(results['trades']) > 0
    trade = results['trades'][0]
    assert trade['trade_type'] == 'IMMEDIATE_BREAKOUT'
    assert trade['position_type'] == 'LONG'
    
    # Verify signal group tracking
    signal_groups = engine.signal_generator.active_signal_groups
    assert len(signal_groups) > 0
    assert signal_groups[0].status == 'active'

@pytest.mark.asyncio
async def test_two_thirty_entry_backtest(config, sample_data):
    """Test backtest with 2:30 PM entry strategy."""
    # Configure for 2:30 entry
    config['entry_type'] = '2_30_ENTRY'
    engine = BacktestEngine(config)
    
    # Add 2:30 PM candle to data
    two_thirty_data = sample_data.copy()
    two_thirty_data['RELIANCE'] = pd.concat([
        two_thirty_data['RELIANCE'],
        pd.DataFrame([{
            'timestamp': pd.Timestamp('2024-04-15 14:30:00'),
            'open': 101.0,
            'high': 102.0,
            'low': 100.5,
            'close': 101.5,
            'DAILY_ATR_14': 5.0
        }])
    ]).reset_index(drop=True)
    
    # Run backtest
    results = await engine.run_backtest(two_thirty_data)
    
    # Verify results
    assert results['status'] == 'success'
    assert len(results['signals']) > 0
    
    # Check signal properties
    signal = results['signals'][0]
    assert signal.type == SignalType.TWO_THIRTY_ENTRY
    assert signal.direction == SignalDirection.LONG
    
    # Verify trade creation
    assert len(results['trades']) > 0
    trade = results['trades'][0]
    assert trade['trade_type'] == 'TWO_THIRTY_ENTRY'
    assert trade['position_type'] == 'LONG'
    
    # Verify signal group tracking
    signal_groups = engine.signal_generator.active_signal_groups
    assert len(signal_groups) > 0
    assert signal_groups[0].status == 'active'

@pytest.mark.asyncio
async def test_invalid_entry_type(config):
    """Test backtest with invalid entry type."""
    config['entry_type'] = 'INVALID_ENTRY'
    
    with pytest.raises(ValueError):
        BacktestEngine(config)

@pytest.mark.asyncio
async def test_no_signals_backtest(config, sample_data):
    """Test backtest with no signal conditions."""
    engine = BacktestEngine(config)
    
    # Modify data to have no breakouts
    no_signal_data = sample_data.copy()
    no_signal_data['RELIANCE']['high'] = 95.0
    no_signal_data['RELIANCE']['low'] = 94.0
    
    # Run backtest
    results = await engine.run_backtest(no_signal_data)
    
    # Verify results
    assert results['status'] == 'success'
    assert len(results['signals']) == 0
    assert len(results['trades']) == 0
    assert len(engine.signal_generator.active_signal_groups) == 0

@pytest.mark.asyncio
async def test_multiple_days_backtest(config):
    """Test backtest over multiple trading days."""
    engine = BacktestEngine(config)
    
    # Create multi-day data
    multi_day_data = {
        'RELIANCE': pd.DataFrame([
            # Day 1 - Long breakout
            {
                'timestamp': pd.Timestamp('2024-04-15 09:15:00'),
                'open': 95.0, 'high': 100.0, 'low': 90.0, 'close': 95.0,
                'DAILY_ATR_14': 5.0
            },
            {
                'timestamp': pd.Timestamp('2024-04-15 09:35:00'),
                'open': 95.5, 'high': 101.0, 'low': 95.0, 'close': 100.5,
                'DAILY_ATR_14': 5.0
            },
            # Day 2 - Short breakout
            {
                'timestamp': pd.Timestamp('2024-04-16 09:15:00'),
                'open': 105.0, 'high': 110.0, 'low': 100.0, 'close': 105.0,
                'DAILY_ATR_14': 5.0
            },
            {
                'timestamp': pd.Timestamp('2024-04-16 09:35:00'),
                'open': 104.5, 'high': 105.0, 'low': 99.0, 'close': 99.5,
                'DAILY_ATR_14': 5.0
            }
        ])
    }
    
    # Run backtest
    results = await engine.run_backtest(multi_day_data)
    
    # Verify results
    assert results['status'] == 'success'
    assert len(results['signals']) == 2  # One signal per day
    assert len(results['trades']) == 2  # One trade per day
    
    # Verify signal types and directions
    assert results['signals'][0].type == SignalType.IMMEDIATE_BREAKOUT
    assert results['signals'][0].direction == SignalDirection.LONG
    assert results['signals'][1].type == SignalType.IMMEDIATE_BREAKOUT
    assert results['signals'][1].direction == SignalDirection.SHORT
    
    # Verify trade types and directions
    assert results['trades'][0]['trade_type'] == 'IMMEDIATE_BREAKOUT'
    assert results['trades'][0]['position_type'] == 'LONG'
    assert results['trades'][1]['trade_type'] == 'IMMEDIATE_BREAKOUT'
    assert results['trades'][1]['position_type'] == 'SHORT'
    
    # Verify signal groups are reset between days
    assert len(engine.signal_generator.active_signal_groups) == 0 