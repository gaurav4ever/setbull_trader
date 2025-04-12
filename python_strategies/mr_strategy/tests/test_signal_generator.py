import pytest
import pandas as pd
from datetime import datetime, time
from ..signals.signal_generator import SignalGenerator
from ..domain.signal import Signal, SignalType, Direction
import logging

# Configure logging
logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger(__name__)

@pytest.fixture
def signal_generator():
    return SignalGenerator()

@pytest.fixture
def sample_candle():
    """Create a sample candle for signal generation"""
    return {
        'timestamp': '2024-01-15 09:20:00',
        'open': 100.0,
        'high': 105.0,
        'low': 95.0,
        'close': 102.0,
        'volume': 1000
    }

@pytest.fixture
def valid_mr_values():
    """Create valid morning range values"""
    return {
        'mr_high': 110.0,
        'mr_low': 90.0,
        'mr_size': 20.0,
        'mr_value': 4.0,  # > 3, so valid
        'is_valid': True,
        'error': None
    }

@pytest.fixture
def invalid_mr_values():
    """Create invalid morning range values"""
    return {
        'mr_high': 110.0,
        'mr_low': 90.0,
        'mr_size': 20.0,
        'mr_value': 2.0,  # < 3, so invalid
        'is_valid': False,
        'error': None
    }

@pytest.mark.asyncio
async def test_skip_first_candle(signal_generator, sample_candle, valid_mr_values):
    """Test that first candle (9:15 AM) is skipped"""
    # Modify candle to be first candle of the day
    first_candle = sample_candle.copy()
    first_candle['timestamp'] = '2024-01-15 09:15:00'
    
    signals = await signal_generator.process_candle(first_candle, valid_mr_values)
    assert len(signals) == 0
    logger.info("First candle (9:15 AM) correctly skipped")

@pytest.mark.asyncio
async def test_skip_invalid_mr(signal_generator, sample_candle, invalid_mr_values):
    """Test that signals are not generated for invalid MR values"""
    signals = await signal_generator.process_candle(sample_candle, invalid_mr_values)
    assert len(signals) == 0
    logger.info("Invalid MR values correctly skipped")

@pytest.mark.asyncio
async def test_generate_breakout_signals(signal_generator, sample_candle, valid_mr_values):
    """Test breakout signal generation"""
    # Test upper breakout
    upper_breakout_candle = sample_candle.copy()
    upper_breakout_candle['high'] = valid_mr_values['mr_high'] + 1.0
    upper_breakout_candle['close'] = valid_mr_values['mr_high'] + 0.5
    
    signals = await signal_generator.process_candle(upper_breakout_candle, valid_mr_values)
    assert len(signals) == 1
    assert signals[0].type == SignalType.BREAKOUT
    assert signals[0].direction == Direction.LONG
    logger.info(f"Generated upper breakout signal: {signals[0]}")
    
    # Test lower breakout
    lower_breakout_candle = sample_candle.copy()
    lower_breakout_candle['low'] = valid_mr_values['mr_low'] - 1.0
    lower_breakout_candle['close'] = valid_mr_values['mr_low'] - 0.5
    
    signals = await signal_generator.process_candle(lower_breakout_candle, valid_mr_values)
    assert len(signals) == 1
    assert signals[0].type == SignalType.BREAKOUT
    assert signals[0].direction == Direction.SHORT
    logger.info(f"Generated lower breakout signal: {signals[0]}")

@pytest.mark.asyncio
async def test_generate_pullback_signals(signal_generator, sample_candle, valid_mr_values):
    """Test pullback signal generation"""
    # Test upper pullback
    upper_pullback_candle = sample_candle.copy()
    upper_pullback_candle['high'] = valid_mr_values['mr_high'] - 1.0
    upper_pullback_candle['close'] = valid_mr_values['mr_high'] - 0.5
    
    signals = await signal_generator.process_candle(upper_pullback_candle, valid_mr_values)
    assert len(signals) == 1
    assert signals[0].type == SignalType.PULLBACK
    assert signals[0].direction == Direction.LONG
    logger.info(f"Generated upper pullback signal: {signals[0]}")
    
    # Test lower pullback
    lower_pullback_candle = sample_candle.copy()
    lower_pullback_candle['low'] = valid_mr_values['mr_low'] + 1.0
    lower_pullback_candle['close'] = valid_mr_values['mr_low'] + 0.5
    
    signals = await signal_generator.process_candle(lower_pullback_candle, valid_mr_values)
    assert len(signals) == 1
    assert signals[0].type == SignalType.PULLBACK
    assert signals[0].direction == Direction.SHORT
    logger.info(f"Generated lower pullback signal: {signals[0]}")

@pytest.mark.asyncio
async def test_no_signals_in_range(signal_generator, sample_candle, valid_mr_values):
    """Test that no signals are generated when price is within MR range"""
    # Price within MR range
    in_range_candle = sample_candle.copy()
    in_range_candle['high'] = valid_mr_values['mr_high'] - 2.0
    in_range_candle['low'] = valid_mr_values['mr_low'] + 2.0
    in_range_candle['close'] = (valid_mr_values['mr_high'] + valid_mr_values['mr_low']) / 2
    
    signals = await signal_generator.process_candle(in_range_candle, valid_mr_values)
    assert len(signals) == 0
    logger.info("No signals generated for price within MR range") 