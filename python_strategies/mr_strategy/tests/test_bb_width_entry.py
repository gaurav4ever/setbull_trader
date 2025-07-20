"""
Unit tests for BB Width Entry Strategy.

This module contains comprehensive tests for the BBWidthEntryStrategy class.
"""

import pytest
import pandas as pd
from datetime import datetime, time
from unittest.mock import Mock

from ..strategy.entry_strategies.bb_width_entry import BBWidthEntryStrategy
from ..strategy.models import Signal, SignalType, SignalDirection


class TestBBWidthEntryStrategy:
    """Test cases for BBWidthEntryStrategy."""
    
    @pytest.fixture
    def config(self):
        """Create a mock configuration for testing."""
        config = Mock()
        config.bb_width_threshold = 0.001  # 0.1%
        config.bb_period = 20
        config.bb_std_dev = 2.0
        config.squeeze_duration_min = 3
        config.squeeze_duration_max = 5
        config.instrument_key = {"direction": "BULLISH"}
        return config
    
    @pytest.fixture
    def strategy(self, config):
        """Create a BBWidthEntryStrategy instance for testing."""
        return BBWidthEntryStrategy(config)
    
    def test_strategy_creation(self, config):
        """Test that BBWidthEntryStrategy can be created."""
        strategy = BBWidthEntryStrategy(config)
        assert strategy is not None
        assert strategy.bb_width_threshold == 0.001
        assert strategy.bb_period == 20
        assert strategy.bb_std_dev == 2.0
        assert strategy.squeeze_duration_min == 3
        assert strategy.squeeze_duration_max == 5
    
    def test_validate_bb_data_valid(self, strategy):
        """Test BB data validation with valid data."""
        candle = {
            'bb_upper': 102.0,
            'bb_lower': 98.0,
            'bb_middle': 100.0,
            'bb_width': 0.04
        }
        assert strategy._validate_bb_data(candle) is True
    
    def test_validate_bb_data_missing_field(self, strategy):
        """Test BB data validation with missing field."""
        candle = {
            'bb_upper': 102.0,
            'bb_lower': 98.0,
            'bb_middle': 100.0
            # Missing bb_width
        }
        assert strategy._validate_bb_data(candle) is False
    
    def test_validate_bb_data_invalid_relationship(self, strategy):
        """Test BB data validation with invalid BB relationships."""
        candle = {
            'bb_upper': 98.0,  # Upper < Lower
            'bb_lower': 102.0,
            'bb_middle': 100.0,
            'bb_width': 0.04
        }
        assert strategy._validate_bb_data(candle) is False
    
    def test_update_bb_width_history(self, strategy):
        """Test BB width history update."""
        strategy._update_bb_width_history(0.05)
        strategy._update_bb_width_history(0.03)
        strategy._update_bb_width_history(0.04)
        
        assert len(strategy.bb_width_history) == 3
        assert strategy.lowest_bb_width == 0.03
    
    def test_update_bb_width_history_max_length(self, strategy):
        """Test that BB width history respects max length."""
        # Add more than max_history_length values
        for i in range(60):
            strategy._update_bb_width_history(0.01 + i * 0.001)
        
        assert len(strategy.bb_width_history) == strategy.max_history_length
        assert strategy.lowest_bb_width == 0.01  # First value should be lowest
    
    @pytest.mark.asyncio
    async def test_check_entry_conditions_no_squeeze(self, strategy):
        """Test entry conditions when no squeeze is detected."""
        candle = {
            'timestamp': pd.Timestamp('2024-01-01 10:00:00'),
            'open': 100.0, 'high': 101.0, 'low': 99.0, 'close': 100.5,
            'bb_upper': 102.0, 'bb_lower': 98.0, 'bb_middle': 100.0, 'bb_width': 0.05  # Wide BB
        }
        mr_values = {}
        
        signal = await strategy.check_entry_conditions(candle, mr_values)
        assert signal is None
        assert strategy.squeeze_detected is False
    
    @pytest.mark.asyncio
    async def test_check_entry_conditions_squeeze_detected(self, strategy):
        """Test entry conditions when squeeze is detected."""
        # First, add some history to establish lowest BB width
        strategy._update_bb_width_history(0.05)
        strategy._update_bb_width_history(0.04)
        strategy._update_bb_width_history(0.03)
        strategy.lowest_bb_width = 0.03
        
        # Create candle with squeeze condition (BB width <= lowest + threshold)
        squeeze_threshold = 0.03 * (1 + 0.001)  # 0.03003
        candle = {
            'timestamp': pd.Timestamp('2024-01-01 10:00:00'),
            'open': 100.0, 'high': 101.0, 'low': 99.0, 'close': 100.5,
            'bb_upper': 102.0, 'bb_lower': 98.0, 'bb_middle': 100.0, 'bb_width': 0.03  # Within threshold
        }
        mr_values = {}
        
        signal = await strategy.check_entry_conditions(candle, mr_values)
        assert signal is None  # No entry yet, just squeeze detected
        assert strategy.squeeze_detected is True
        assert strategy.squeeze_candle_count == 1
    
    @pytest.mark.asyncio
    async def test_check_entry_conditions_long_entry(self, strategy):
        """Test long entry during squeeze."""
        # Setup squeeze condition
        strategy._update_bb_width_history(0.05)
        strategy._update_bb_width_history(0.04)
        strategy._update_bb_width_history(0.03)
        strategy.lowest_bb_width = 0.03
        strategy.squeeze_detected = True
        strategy.squeeze_candle_count = 3  # Within duration range
        
        # Create candle with price above BB upper (long entry condition)
        candle = {
            'timestamp': pd.Timestamp('2024-01-01 10:00:00'),
            'open': 100.0, 'high': 103.0, 'low': 99.0, 'close': 102.5,  # Close above BB upper
            'bb_upper': 102.0, 'bb_lower': 98.0, 'bb_middle': 100.0, 'bb_width': 0.03
        }
        mr_values = {}
        
        signal = await strategy.check_entry_conditions(candle, mr_values)
        
        assert signal is not None
        assert signal.type == SignalType.IMMEDIATE_BREAKOUT
        assert signal.direction == SignalDirection.LONG
        assert signal.price == 102.0  # BB upper band
        assert strategy.in_long_trade is True
    
    @pytest.mark.asyncio
    async def test_check_entry_conditions_short_entry(self, strategy):
        """Test short entry during squeeze."""
        # Setup for short entry
        strategy.config.instrument_key = {"direction": "BEARISH"}
        
        # Setup squeeze condition
        strategy._update_bb_width_history(0.05)
        strategy._update_bb_width_history(0.04)
        strategy._update_bb_width_history(0.03)
        strategy.lowest_bb_width = 0.03
        strategy.squeeze_detected = True
        strategy.squeeze_candle_count = 3  # Within duration range
        
        # Create candle with price below BB lower (short entry condition)
        candle = {
            'timestamp': pd.Timestamp('2024-01-01 10:00:00'),
            'open': 100.0, 'high': 101.0, 'low': 97.0, 'close': 97.5,  # Close below BB lower
            'bb_upper': 102.0, 'bb_lower': 98.0, 'bb_middle': 100.0, 'bb_width': 0.03
        }
        mr_values = {}
        
        signal = await strategy.check_entry_conditions(candle, mr_values)
        
        assert signal is not None
        assert signal.type == SignalType.IMMEDIATE_BREAKOUT
        assert signal.direction == SignalDirection.SHORT
        assert signal.price == 98.0  # BB lower band
        assert strategy.in_short_trade is True
    
    @pytest.mark.asyncio
    async def test_check_entry_conditions_squeeze_duration_too_short(self, strategy):
        """Test that entry is not generated if squeeze duration is too short."""
        # Setup squeeze condition but with short duration
        strategy._update_bb_width_history(0.05)
        strategy._update_bb_width_history(0.04)
        strategy._update_bb_width_history(0.03)
        strategy.lowest_bb_width = 0.03
        strategy.squeeze_detected = True
        strategy.squeeze_candle_count = 2  # Below minimum duration
        
        candle = {
            'timestamp': pd.Timestamp('2024-01-01 10:00:00'),
            'open': 100.0, 'high': 103.0, 'low': 99.0, 'close': 102.5,
            'bb_upper': 102.0, 'bb_lower': 98.0, 'bb_middle': 100.0, 'bb_width': 0.03
        }
        mr_values = {}
        
        signal = await strategy.check_entry_conditions(candle, mr_values)
        assert signal is None  # No entry due to short duration
    
    @pytest.mark.asyncio
    async def test_check_entry_conditions_squeeze_duration_too_long(self, strategy):
        """Test that entry is not generated if squeeze duration is too long."""
        # Setup squeeze condition but with long duration
        strategy._update_bb_width_history(0.05)
        strategy._update_bb_width_history(0.04)
        strategy._update_bb_width_history(0.03)
        strategy.lowest_bb_width = 0.03
        strategy.squeeze_detected = True
        strategy.squeeze_candle_count = 6  # Above maximum duration
        
        candle = {
            'timestamp': pd.Timestamp('2024-01-01 10:00:00'),
            'open': 100.0, 'high': 103.0, 'low': 99.0, 'close': 102.5,
            'bb_upper': 102.0, 'bb_lower': 98.0, 'bb_middle': 100.0, 'bb_width': 0.03
        }
        mr_values = {}
        
        signal = await strategy.check_entry_conditions(candle, mr_values)
        assert signal is None  # No entry due to long duration
    
    @pytest.mark.asyncio
    async def test_check_entry_conditions_already_in_trade(self, strategy):
        """Test that no entry is generated if already in a trade."""
        # Setup squeeze condition
        strategy._update_bb_width_history(0.05)
        strategy._update_bb_width_history(0.04)
        strategy._update_bb_width_history(0.03)
        strategy.lowest_bb_width = 0.03
        strategy.squeeze_detected = True
        strategy.squeeze_candle_count = 3
        strategy.in_long_trade = True  # Already in trade
        
        candle = {
            'timestamp': pd.Timestamp('2024-01-01 10:00:00'),
            'open': 100.0, 'high': 103.0, 'low': 99.0, 'close': 102.5,
            'bb_upper': 102.0, 'bb_lower': 98.0, 'bb_middle': 100.0, 'bb_width': 0.03
        }
        mr_values = {}
        
        signal = await strategy.check_entry_conditions(candle, mr_values)
        assert signal is None  # No entry due to existing trade
    
    @pytest.mark.asyncio
    async def test_check_entry_conditions_outside_trading_hours(self, strategy):
        """Test that no entry is generated outside trading hours."""
        candle = {
            'timestamp': pd.Timestamp('2024-01-01 08:00:00'),  # Before market open
            'open': 100.0, 'high': 101.0, 'low': 99.0, 'close': 100.5,
            'bb_upper': 102.0, 'bb_lower': 98.0, 'bb_middle': 100.0, 'bb_width': 0.03
        }
        mr_values = {}
        
        signal = await strategy.check_entry_conditions(candle, mr_values)
        assert signal is None  # No entry outside trading hours
    
    def test_reset_state(self, strategy):
        """Test that state is properly reset."""
        # Set some state
        strategy.in_long_trade = True
        strategy.in_short_trade = False
        strategy.squeeze_detected = True
        strategy.squeeze_candle_count = 3
        strategy.lowest_bb_width = 0.03
        strategy.bb_width_history = [0.03, 0.04, 0.05]
        
        # Reset state
        strategy.reset_state()
        
        # Verify reset
        assert strategy.in_long_trade is False
        assert strategy.in_short_trade is False
        assert strategy.squeeze_detected is False
        assert strategy.squeeze_candle_count == 0
        assert strategy.lowest_bb_width == float('inf')
        assert len(strategy.bb_width_history) == 0
        assert strategy.can_generate_long is True
        assert strategy.can_generate_short is True
    
    def test_format_candle_info(self, strategy):
        """Test candle information formatting."""
        candle = {
            'timestamp': pd.Timestamp('2024-01-01 10:00:00'),
            'open': 100.0,
            'high': 101.0,
            'low': 99.0,
            'close': 100.5
        }
        
        info = strategy._format_candle_info(candle)
        expected = "[2024-01-01 10:00:00] [O:100.00 H:101.00 L:99.00 C:100.50] - "
        assert info == expected
    
    def test_format_candle_info_empty(self, strategy):
        """Test candle information formatting with empty candle."""
        info = strategy._format_candle_info({})
        assert info == ""
        
        info = strategy._format_candle_info(None)
        assert info == "" 