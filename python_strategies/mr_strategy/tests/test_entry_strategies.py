"""
Tests for entry strategy implementations.
"""

import pytest
from datetime import datetime, time
import pandas as pd

from ..strategy.entry_strategies.factory import EntryStrategyFactory
from ..strategy.entry_strategies.first_entry import FirstEntryStrategy
from ..strategy.entry_strategies.two_thirty_entry import TwoThirtyEntryStrategy
from ..strategy.entry_strategies.bb_lower_entry import BBLowerEntryStrategy
from ..strategy.models import SignalType, SignalDirection
from ..strategy.config import MRStrategyConfig

@pytest.fixture
def config():
    """Create test configuration."""
    return MRStrategyConfig(
        buffer_ticks=5,
        tick_size=0.05,
        breakout_percentage=0.003,
        invalidation_percentage=0.005,
        instrument_key={"direction": "BULLISH"}
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
    
    # Test BB Lower entry strategy creation
    bb_lower_entry = EntryStrategyFactory.create_strategy("BB_LOWER_ENTRY", config)
    assert isinstance(bb_lower_entry, BBLowerEntryStrategy)
    
    # Test invalid entry type
    with pytest.raises(ValueError):
        EntryStrategyFactory.create_strategy("INVALID_ENTRY", config)

@pytest.mark.asyncio
async def test_bb_lower_entry_strategy_creation(config):
    """Test BB Lower entry strategy creation."""
    strategy = BBLowerEntryStrategy(config)
    assert strategy is not None
    assert strategy.confirmation_time == time(12, 0)
    assert strategy.stop_loss_percentage == 0.002
    assert strategy.observing_confirmation is True
    assert strategy.confirmation_complete is False

@pytest.mark.asyncio
async def test_bb_lower_entry_day_level_tracking(config):
    """Test BB Lower entry strategy day level tracking."""
    strategy = BBLowerEntryStrategy(config)
    
    # Test day high tracking
    candle1 = {
        'timestamp': pd.Timestamp('2024-01-01 10:00:00'),
        'open': 100.0, 'high': 102.0, 'low': 99.0, 'close': 101.0,
        'bb_upper': 103.0, 'bb_lower': 97.0, 'bb_middle': 100.0, 'bb_width': 0.06
    }
    
    strategy._update_day_levels(candle1)
    assert strategy.day_high == 102.0
    assert strategy.day_low == 99.0
    assert strategy.middle_line == 100.5
    
    # Test day high update
    candle2 = {
        'timestamp': pd.Timestamp('2024-01-01 11:00:00'),
        'open': 101.0, 'high': 104.0, 'low': 100.0, 'close': 103.0,
        'bb_upper': 103.0, 'bb_lower': 97.0, 'bb_middle': 100.0, 'bb_width': 0.06
    }
    
    strategy._update_day_levels(candle2)
    assert strategy.day_high == 104.0  # Updated
    assert strategy.day_low == 99.0    # Unchanged
    assert strategy.middle_line == 101.5  # Updated

@pytest.mark.asyncio
async def test_bb_lower_entry_bb_data_validation(config):
    """Test BB Lower entry strategy BB data validation."""
    strategy = BBLowerEntryStrategy(config)
    
    # Test valid BB data
    valid_candle = {
        'timestamp': pd.Timestamp('2024-01-01 10:00:00'),
        'open': 100.0, 'high': 102.0, 'low': 99.0, 'close': 101.0,
        'bb_upper': 103.0, 'bb_lower': 97.0, 'bb_middle': 100.0, 'bb_width': 0.06
    }
    assert strategy._validate_bb_data(valid_candle) is True
    
    # Test missing BB data
    invalid_candle = {
        'timestamp': pd.Timestamp('2024-01-01 10:00:00'),
        'open': 100.0, 'high': 102.0, 'low': 99.0, 'close': 101.0
        # Missing BB data
    }
    assert strategy._validate_bb_data(invalid_candle) is False
    
    # Test invalid BB relationships
    invalid_bb_candle = {
        'timestamp': pd.Timestamp('2024-01-01 10:00:00'),
        'open': 100.0, 'high': 102.0, 'low': 99.0, 'close': 101.0,
        'bb_upper': 97.0, 'bb_lower': 103.0, 'bb_middle': 100.0, 'bb_width': 0.06  # Upper < Lower
    }
    assert strategy._validate_bb_data(invalid_bb_candle) is False

@pytest.mark.asyncio
async def test_bb_lower_entry_confirmation_phase(config):
    """Test BB Lower entry strategy confirmation phase."""
    strategy = BBLowerEntryStrategy(config)
    
    # Test before 12:00 PM (should be in confirmation phase)
    candle_before = {
        'timestamp': pd.Timestamp('2024-01-01 10:00:00'),
        'open': 100.0, 'high': 102.0, 'low': 99.0, 'close': 101.0,
        'bb_upper': 103.0, 'bb_lower': 97.0, 'bb_middle': 100.0, 'bb_width': 0.06
    }
    
    signal = await strategy.check_entry_conditions(candle_before, {})
    assert signal is None  # Should not generate signal during confirmation phase
    assert strategy.observing_confirmation is True
    assert strategy.confirmation_complete is False

@pytest.mark.asyncio
async def test_bb_lower_entry_12pm_confirmation(config):
    """Test BB Lower entry strategy 12:00 PM confirmation."""
    strategy = BBLowerEntryStrategy(config)
    
    # Set up day levels first
    strategy.day_high = 102.0
    strategy.day_low = 99.0
    strategy.middle_line = 100.5
    
    # Test 12:00 PM confirmation (bullish trend confirmed)
    candle_1200 = {
        'timestamp': pd.Timestamp('2024-01-01 12:00:00'),
        'open': 101.0, 'high': 102.5, 'low': 100.5, 'close': 101.5,  # Close above middle line
        'bb_upper': 103.0, 'bb_lower': 97.0, 'bb_middle': 100.0, 'bb_width': 0.06
    }
    
    signal = await strategy.check_entry_conditions(candle_1200, {})
    assert signal is None  # Should not generate signal at confirmation time
    assert strategy.observing_confirmation is False
    assert strategy.confirmation_complete is True
    assert strategy.trend_strength_confirmed is True
    assert strategy.confirmation_attempted is True
    assert strategy.confirmation_failed is False
    assert strategy.price_position_at_confirmation == 0.83  # (101.5 - 99.0) / (102.0 - 99.0) = 2.5 / 3.0 = 0.83

@pytest.mark.asyncio
async def test_bb_lower_entry_confirmation_failed(config):
    """Test BB Lower entry strategy confirmation failure."""
    strategy = BBLowerEntryStrategy(config)
    
    # Set up day levels first
    strategy.day_high = 102.0
    strategy.day_low = 99.0
    strategy.middle_line = 100.5
    
    # Test 12:00 PM confirmation (bullish trend failed - price below middle line)
    candle_1200 = {
        'timestamp': pd.Timestamp('2024-01-01 12:00:00'),
        'open': 100.0, 'high': 100.5, 'low': 99.5, 'close': 100.2,  # Close below middle line
        'bb_upper': 103.0, 'bb_lower': 97.0, 'bb_middle': 100.0, 'bb_width': 0.06
    }
    
    signal = await strategy.check_entry_conditions(candle_1200, {})
    assert signal is None  # Should not generate signal at confirmation time
    assert strategy.observing_confirmation is False
    assert strategy.confirmation_complete is True
    assert strategy.trend_strength_confirmed is False
    assert strategy.confirmation_attempted is True
    assert strategy.confirmation_failed is True

@pytest.mark.asyncio
async def test_bb_lower_entry_insufficient_trend_strength(config):
    """Test BB Lower entry strategy insufficient trend strength."""
    strategy = BBLowerEntryStrategy(config)
    
    # Set up day levels first
    strategy.day_high = 102.0
    strategy.day_low = 99.0
    strategy.middle_line = 100.5
    
    # Test 12:00 PM confirmation (price above middle but in lower half of range)
    candle_1200 = {
        'timestamp': pd.Timestamp('2024-01-01 12:00:00'),
        'open': 100.0, 'high': 100.8, 'low': 99.8, 'close': 100.2,  # Close above middle but position < 50%
        'bb_upper': 103.0, 'bb_lower': 97.0, 'bb_middle': 100.0, 'bb_width': 0.06
    }
    
    signal = await strategy.check_entry_conditions(candle_1200, {})
    assert signal is None  # Should not generate signal at confirmation time
    assert strategy.observing_confirmation is False
    assert strategy.confirmation_complete is True
    assert strategy.trend_strength_confirmed is False
    assert strategy.confirmation_attempted is True
    assert strategy.confirmation_failed is True
    assert strategy.price_position_at_confirmation == 0.4  # (100.2 - 99.0) / (102.0 - 99.0) = 1.2 / 3.0 = 0.4

@pytest.mark.asyncio
async def test_bb_lower_entry_bearish_confirmation(config):
    """Test BB Lower entry strategy bearish confirmation."""
    # Create config with bearish direction
    bearish_config = MRStrategyConfig(
        buffer_ticks=5,
        tick_size=0.05,
        breakout_percentage=0.003,
        invalidation_percentage=0.005,
        instrument_key={"direction": "BEARISH"}
    )
    
    strategy = BBLowerEntryStrategy(bearish_config)
    
    # Set up day levels first
    strategy.day_high = 102.0
    strategy.day_low = 99.0
    strategy.middle_line = 100.5
    
    # Test 12:00 PM confirmation (bearish trend confirmed - price below middle line)
    candle_1200 = {
        'timestamp': pd.Timestamp('2024-01-01 12:00:00'),
        'open': 100.0, 'high': 100.5, 'low': 99.5, 'close': 99.8,  # Close below middle line
        'bb_upper': 103.0, 'bb_lower': 97.0, 'bb_middle': 100.0, 'bb_width': 0.06
    }
    
    signal = await strategy.check_entry_conditions(candle_1200, {})
    assert signal is None  # Should not generate signal at confirmation time
    assert strategy.observing_confirmation is False
    assert strategy.confirmation_complete is True
    assert strategy.trend_strength_confirmed is True
    assert strategy.confirmation_attempted is True
    assert strategy.confirmation_failed is False
    assert strategy.price_position_at_confirmation == 0.73  # (102.0 - 99.8) / (102.0 - 99.0) = 2.2 / 3.0 = 0.73

@pytest.mark.asyncio
async def test_bb_lower_entry_signal_type(config):
    """Test BB Lower entry strategy signal type."""
    strategy = BBLowerEntryStrategy(config)
    
    # Set up confirmation as complete
    strategy.confirmation_complete = True
    strategy.trend_strength_confirmed = True
    strategy.confirmation_attempted = True
    strategy.day_high = 102.0
    strategy.day_low = 99.0
    strategy.middle_line = 100.5
    strategy.bb_upper = 103.0
    strategy.bb_lower = 97.0
    strategy.bb_middle = 100.0
    strategy.current_bb_width = 0.06
    
    # Test entry signal generation
    candle_entry = {
        'timestamp': pd.Timestamp('2024-01-01 14:00:00'),
        'open': 97.5, 'high': 98.0, 'low': 96.5, 'close': 97.0,  # Low <= BB lower
        'bb_upper': 103.0, 'bb_lower': 97.0, 'bb_middle': 100.0, 'bb_width': 0.06
    }
    
    signal = await strategy.check_entry_conditions(candle_entry, {})
    assert signal is not None
    assert signal.type == SignalType.BB_LOWER_ENTRY
    assert signal.direction == SignalDirection.LONG
    assert signal.price == 97.0  # BB lower band

@pytest.mark.asyncio
async def test_bb_lower_entry_validation_bullish(config):
    """Test BB Lower entry strategy bullish entry validation."""
    strategy = BBLowerEntryStrategy(config)
    
    # Set up confirmation as complete
    strategy.confirmation_complete = True
    strategy.trend_strength_confirmed = True
    strategy.confirmation_attempted = True
    strategy.day_high = 102.0
    strategy.day_low = 99.0
    strategy.middle_line = 100.5
    strategy.bb_upper = 103.0
    strategy.bb_lower = 97.0
    strategy.bb_middle = 100.0
    strategy.current_bb_width = 0.06
    
    # Test valid bullish entry
    candle_valid = {
        'timestamp': pd.Timestamp('2024-01-01 14:00:00'),
        'open': 97.5, 'high': 98.0, 'low': 96.5, 'close': 101.0,  # Close above middle line
        'bb_upper': 103.0, 'bb_lower': 97.0, 'bb_middle': 100.0, 'bb_width': 0.06
    }
    
    signal = await strategy.check_entry_conditions(candle_valid, {})
    assert signal is not None
    assert signal.direction == SignalDirection.LONG
    
    # Test invalid bullish entry (price below middle line)
    candle_invalid = {
        'timestamp': pd.Timestamp('2024-01-01 14:00:00'),
        'open': 97.5, 'high': 98.0, 'low': 96.5, 'close': 100.0,  # Close below middle line
        'bb_upper': 103.0, 'bb_lower': 97.0, 'bb_middle': 100.0, 'bb_width': 0.06
    }
    
    signal = await strategy.check_entry_conditions(candle_invalid, {})
    assert signal is None

@pytest.mark.asyncio
async def test_bb_lower_entry_validation_bearish(config):
    """Test BB Lower entry strategy bearish entry validation."""
    # Create config with bearish direction
    bearish_config = MRStrategyConfig(
        buffer_ticks=5,
        tick_size=0.05,
        breakout_percentage=0.003,
        invalidation_percentage=0.005,
        instrument_key={"direction": "BEARISH"}
    )
    
    strategy = BBLowerEntryStrategy(bearish_config)
    
    # Set up confirmation as complete
    strategy.confirmation_complete = True
    strategy.trend_strength_confirmed = True
    strategy.confirmation_attempted = True
    strategy.day_high = 102.0
    strategy.day_low = 99.0
    strategy.middle_line = 100.5
    strategy.bb_upper = 103.0
    strategy.bb_lower = 97.0
    strategy.bb_middle = 100.0
    strategy.current_bb_width = 0.06
    
    # Test valid bearish entry
    candle_valid = {
        'timestamp': pd.Timestamp('2024-01-01 14:00:00'),
        'open': 103.5, 'high': 104.0, 'low': 102.5, 'close': 99.0,  # Close below middle line
        'bb_upper': 103.0, 'bb_lower': 97.0, 'bb_middle': 100.0, 'bb_width': 0.06
    }
    
    signal = await strategy.check_entry_conditions(candle_valid, {})
    assert signal is not None
    assert signal.direction == SignalDirection.SHORT
    assert signal.price == 103.0  # BB upper band

@pytest.mark.asyncio
async def test_bb_lower_entry_target_calculation(config):
    """Test BB Lower entry strategy target price calculation."""
    strategy = BBLowerEntryStrategy(config)
    
    # Set up confirmation as complete
    strategy.confirmation_complete = True
    strategy.trend_strength_confirmed = True
    strategy.confirmation_attempted = True
    strategy.day_high = 102.0
    strategy.day_low = 99.0
    strategy.middle_line = 100.5
    strategy.bb_upper = 103.0
    strategy.bb_lower = 97.0
    strategy.bb_middle = 100.0
    strategy.current_bb_width = 0.06
    strategy.entry_price = 97.0
    
    # Test target calculation for long
    target_long = strategy._calculate_target_price("LONG")
    expected_target = 97.0 + (3.0 * 1.5)  # entry + (day_range * 1.5)
    assert target_long == expected_target
    
    # Test target calculation for short
    target_short = strategy._calculate_target_price("SHORT")
    expected_target = 97.0 - (3.0 * 1.5)  # entry - (day_range * 1.5)
    assert target_short == expected_target

@pytest.mark.asyncio
async def test_bb_lower_entry_signal_metadata(config):
    """Test BB Lower entry strategy signal metadata."""
    strategy = BBLowerEntryStrategy(config)
    
    # Set up confirmation as complete
    strategy.confirmation_complete = True
    strategy.trend_strength_confirmed = True
    strategy.confirmation_attempted = True
    strategy.day_high = 102.0
    strategy.day_low = 99.0
    strategy.middle_line = 100.5
    strategy.bb_upper = 103.0
    strategy.bb_lower = 97.0
    strategy.bb_middle = 100.0
    strategy.current_bb_width = 0.06
    
    # Test entry signal generation
    candle_entry = {
        'timestamp': pd.Timestamp('2024-01-01 14:00:00'),
        'open': 97.5, 'high': 98.0, 'low': 96.5, 'close': 101.0,
        'bb_upper': 103.0, 'bb_lower': 97.0, 'bb_middle': 100.0, 'bb_width': 0.06
    }
    
    signal = await strategy.check_entry_conditions(candle_entry, {})
    assert signal is not None
    
    # Check signal metadata
    assert signal.metadata['entry_type'] == 'bb_lower_entry'
    assert signal.metadata['strategy'] == 'BB_LOWER_ENTRY'
    assert signal.metadata['trade_type'] == 'BB_LOWER_ENTRY'
    assert signal.metadata['stop_loss_percentage'] == 0.002
    assert signal.metadata['bb_width_at_entry'] == 0.06
    assert signal.metadata['trend_strength_confirmed'] is True
    assert signal.metadata['confirmation_attempted'] is True
    assert signal.metadata['confirmation_failed'] is False
    
    # Check range values
    assert signal.range_values['entry_price'] == 97.0
    assert signal.range_values['stop_loss_price'] == 97.0 * 0.998  # 0.2% below entry
    assert signal.range_values['day_high'] == 102.0
    assert signal.range_values['day_low'] == 99.0
    assert signal.range_values['middle_line'] == 100.5
    assert 'risk_amount' in signal.range_values
    assert 'reward_amount' in signal.range_values
    assert 'risk_reward_ratio' in signal.range_values

@pytest.mark.asyncio
async def test_bb_lower_entry_data_processing_integration(config):
    """Test BB Lower entry strategy data processing integration."""
    from mr_strategy.data.data_processor import CandleProcessor
    import pandas as pd
    import numpy as np
    
    # Create sample candle data
    dates = pd.date_range('2024-01-01 09:15:00', '2024-01-01 15:30:00', freq='5min')
    sample_data = pd.DataFrame({
        'timestamp': dates,
        'open': np.random.uniform(100, 102, len(dates)),
        'high': np.random.uniform(102, 104, len(dates)),
        'low': np.random.uniform(98, 100, len(dates)),
        'close': np.random.uniform(100, 102, len(dates)),
        'volume': np.random.randint(1000, 10000, len(dates))
    })
    
    # Process data with BB indicators
    processor = CandleProcessor()
    processed_data = processor.add_bb_indicators_for_strategy(sample_data)
    
    # Validate BB data is present
    assert 'bb_upper' in processed_data.columns
    assert 'bb_lower' in processed_data.columns
    assert 'bb_middle' in processed_data.columns
    assert 'bb_width' in processed_data.columns
    
    # Validate BB data relationships
    assert (processed_data['bb_upper'] >= processed_data['bb_lower']).all()
    assert (processed_data['bb_middle'] >= processed_data['bb_lower']).all()
    assert (processed_data['bb_upper'] >= processed_data['bb_middle']).all()
    
    # Test BB data validation
    assert processor.validate_bb_data_for_strategy(processed_data) is True
    
    # Test with missing BB data
    invalid_data = sample_data.copy()
    assert processor.validate_bb_data_for_strategy(invalid_data) is False

@pytest.mark.asyncio
async def test_bb_lower_entry_configuration_integration():
    """Test BB Lower entry strategy configuration integration."""
    from mr_strategy.strategy.config import MRStrategyConfig
    from datetime import time
    
    # Test default configuration
    config = MRStrategyConfig(
        instrument_key={"direction": "BULLISH"}
    )
    
    assert config.bb_lower_period == 20
    assert config.bb_lower_std_dev == 2.0
    assert config.bb_lower_confirmation_time == time(12, 0)
    assert config.bb_lower_stop_loss_percentage == 0.002
    assert config.bb_lower_target_multiplier == 1.5
    assert config.bb_lower_volume_confirmation_enabled is True
    assert config.bb_lower_buying_volume_ratio == 1.5
    assert config.bb_lower_selling_volume_ratio == 1.5
    assert config.bb_lower_volume_analysis_window == 5
    
    # Test custom configuration
    custom_config = MRStrategyConfig(
        instrument_key={"direction": "BULLISH"},
        bb_lower_period=30,
        bb_lower_std_dev=2.5,
        bb_lower_confirmation_time=time(11, 30),
        bb_lower_stop_loss_percentage=0.003,
        bb_lower_target_multiplier=2.0,
        bb_lower_volume_confirmation_enabled=False,
        bb_lower_buying_volume_ratio=2.0,
        bb_lower_selling_volume_ratio=2.0,
        bb_lower_volume_analysis_window=10
    )
    
    assert custom_config.bb_lower_period == 30
    assert custom_config.bb_lower_std_dev == 2.5
    assert custom_config.bb_lower_confirmation_time == time(11, 30)
    assert custom_config.bb_lower_stop_loss_percentage == 0.003
    assert custom_config.bb_lower_target_multiplier == 2.0
    assert custom_config.bb_lower_volume_confirmation_enabled is False
    assert custom_config.bb_lower_buying_volume_ratio == 2.0
    assert custom_config.bb_lower_selling_volume_ratio == 2.0
    assert custom_config.bb_lower_volume_analysis_window == 10

@pytest.mark.asyncio
async def test_bb_lower_entry_strategy_with_processed_data(config):
    """Test BB Lower entry strategy with processed data from data processor."""
    from mr_strategy.data.data_processor import CandleProcessor
    import pandas as pd
    import numpy as np
    
    # Create sample candle data
    dates = pd.date_range('2024-01-01 09:15:00', '2024-01-01 15:30:00', freq='5min')
    sample_data = pd.DataFrame({
        'timestamp': dates,
        'open': np.random.uniform(100, 102, len(dates)),
        'high': np.random.uniform(102, 104, len(dates)),
        'low': np.random.uniform(98, 100, len(dates)),
        'close': np.random.uniform(100, 102, len(dates)),
        'volume': np.random.randint(1000, 10000, len(dates))
    })
    
    # Process data with BB indicators
    processor = CandleProcessor()
    processed_data = processor.add_bb_indicators_for_strategy(sample_data)
    
    # Create strategy with custom configuration
    custom_config = MRStrategyConfig(
        instrument_key={"direction": "BULLISH"},
        bb_lower_period=20,
        bb_lower_std_dev=2.0,
        bb_lower_target_multiplier=2.0
    )
    
    strategy = BBLowerEntryStrategy(custom_config)
    
    # Test with processed data
    for idx, row in processed_data.iterrows():
        candle = row.to_dict()
        
        # Skip first few candles (need enough data for BB calculation)
        if idx < 20:
            continue
            
        # Test at 12:00 PM for confirmation
        if row['timestamp'].time() == time(12, 0):
            signal = await strategy.check_entry_conditions(candle, {})
            # Should not generate signal at confirmation time
            assert signal is None
            
        # Test after 12:00 PM for entry
        elif row['timestamp'].time() > time(12, 0):
            signal = await strategy.check_entry_conditions(candle, {})
            # May or may not generate signal depending on conditions
            if signal is not None:
                assert signal.type == SignalType.BB_LOWER_ENTRY
                assert signal.direction == SignalDirection.LONG
                assert 'bb_upper' in signal.range_values
                assert 'bb_lower' in signal.range_values
                assert 'bb_middle' in signal.range_values
                assert 'bb_width' in signal.range_values 