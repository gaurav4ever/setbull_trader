"""
Tests for entry strategy implementations.
"""

import pytest
from datetime import datetime, time
import pandas as pd

from ..strategy.entry_strategies.factory import EntryStrategyFactory
from ..strategy.entry_strategies.first_entry import FirstEntryStrategy
from ..strategy.entry_strategies.two_thirty_entry import TwoThirtyEntryStrategy
from ..strategy.models import SignalType, SignalDirection
from ..strategy.config import MRStrategyConfig

@pytest.fixture
def config():
    """Create test configuration."""
    return MRStrategyConfig(
        buffer_ticks=5,
        tick_size=0.05,
        breakout_percentage=0.003,
        invalidation_percentage=0.005
    )

@pytest.fixture
def mr_values():
    """Create test morning range values."""
    return {
        'high': 100.0,
        'low': 90.0,
        'size': 10.0,
        'is_valid': True
    }

@pytest.fixture
def sample_candle():
    """Create a sample candle."""
    return {
        'timestamp': pd.Timestamp('2024-04-15 09:30:00'),
        'open': 95.0,
        'high': 105.0,
        'low': 85.0,
        'close': 100.0
    }

@pytest.mark.asyncio
async def test_first_entry_strategy_long(config, mr_values, sample_candle):
    """Test first entry strategy long signal."""
    strategy = FirstEntryStrategy(config)
    
    # Test long breakout
    signal = await strategy.check_entry_conditions(sample_candle, mr_values)
    assert signal is not None
    assert signal.type == SignalType.IMMEDIATE_BREAKOUT
    assert signal.direction == SignalDirection.LONG
    assert signal.price == mr_values['high']

@pytest.mark.asyncio
async def test_first_entry_strategy_short(config, mr_values, sample_candle):
    """Test first entry strategy short signal."""
    strategy = FirstEntryStrategy(config)
    
    # Test short breakout
    signal = await strategy.check_entry_conditions(sample_candle, mr_values)
    assert signal is not None
    assert signal.type == SignalType.IMMEDIATE_BREAKOUT
    assert signal.direction == SignalDirection.SHORT
    assert signal.price == mr_values['low']

@pytest.mark.asyncio
async def test_two_thirty_entry_strategy_long(config, mr_values):
    """Test 2:30 PM entry strategy long signal."""
    strategy = TwoThirtyEntryStrategy(config)
    
    # Create 2:30 PM candle above MR high
    candle = {
        'timestamp': pd.Timestamp('2024-04-15 14:30:00'),
        'open': 101.0,
        'high': 102.0,
        'low': 100.5,
        'close': 101.5
    }
    
    signal = await strategy.check_entry_conditions(candle, mr_values)
    assert signal is not None
    assert signal.type == SignalType.TWO_THIRTY_ENTRY
    assert signal.direction == SignalDirection.LONG
    assert signal.price == candle['close']

@pytest.mark.asyncio
async def test_two_thirty_entry_strategy_short(config, mr_values):
    """Test 2:30 PM entry strategy short signal."""
    strategy = TwoThirtyEntryStrategy(config)
    
    # Create 2:30 PM candle below MR low
    candle = {
        'timestamp': pd.Timestamp('2024-04-15 14:30:00'),
        'open': 89.0,
        'high': 89.5,
        'low': 88.0,
        'close': 88.5
    }
    
    signal = await strategy.check_entry_conditions(candle, mr_values)
    assert signal is not None
    assert signal.type == SignalType.TWO_THIRTY_ENTRY
    assert signal.direction == SignalDirection.SHORT
    assert signal.price == candle['close']

@pytest.mark.asyncio
async def test_two_thirty_entry_strategy_no_signal(config, mr_values):
    """Test 2:30 PM entry strategy with no signal conditions."""
    strategy = TwoThirtyEntryStrategy(config)
    
    # Test with non-2:30 PM candle
    candle = {
        'timestamp': pd.Timestamp('2024-04-15 14:15:00'),
        'open': 101.0,
        'high': 102.0,
        'low': 100.5,
        'close': 101.5
    }
    
    signal = await strategy.check_entry_conditions(candle, mr_values)
    assert signal is None
    
    # Test with price within MR range
    candle = {
        'timestamp': pd.Timestamp('2024-04-15 14:30:00'),
        'open': 95.0,
        'high': 96.0,
        'low': 94.0,
        'close': 95.5
    }
    
    signal = await strategy.check_entry_conditions(candle, mr_values)
    assert signal is None

def test_entry_strategy_factory(config):
    """Test entry strategy factory."""
    # Test first entry strategy creation
    first_entry = EntryStrategyFactory.create_strategy("1ST_ENTRY", config)
    assert isinstance(first_entry, FirstEntryStrategy)
    
    # Test two thirty entry strategy creation
    two_thirty_entry = EntryStrategyFactory.create_strategy("2_30_ENTRY", config)
    assert isinstance(two_thirty_entry, TwoThirtyEntryStrategy)
    
    # Test invalid entry type
    with pytest.raises(ValueError):
        EntryStrategyFactory.create_strategy("INVALID_ENTRY", config) 