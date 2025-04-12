import pytest
import pandas as pd
from datetime import datetime, timedelta
from ..data.data_processor import CandleProcessor
import logging

# Configure logging
logging.basicConfig(level=logging.DEBUG)
logger = logging.getLogger(__name__)

@pytest.fixture
def candle_processor():
    return CandleProcessor()

@pytest.fixture
def sample_daily_candles():
    """Create sample daily candles for ATR calculation"""
    dates = pd.date_range(start='2024-01-01', periods=14, freq='D')
    data = {
        'timestamp': dates,
        'open': [100 + i for i in range(14)],
        'high': [105 + i for i in range(14)],
        'low': [95 + i for i in range(14)],
        'close': [102 + i for i in range(14)],
        'volume': [1000 + i * 100 for i in range(14)]
    }
    return pd.DataFrame(data)

@pytest.fixture
def sample_morning_candles():
    """Create sample 5-minute candles for morning range calculation"""
    times = pd.date_range(start='2024-01-15 09:15:00', periods=6, freq='5T')
    data = {
        'timestamp': times,
        'open': [100, 101, 102, 103, 104, 105],
        'high': [105, 106, 107, 108, 109, 110],
        'low': [95, 96, 97, 98, 99, 100],
        'close': [102, 103, 104, 105, 106, 107],
        'volume': [1000, 1100, 1200, 1300, 1400, 1500]
    }
    return pd.DataFrame(data)

@pytest.mark.asyncio
async def test_calculate_atr(candle_processor, sample_daily_candles):
    """Test ATR calculation"""
    # Mock the load_daily_data method
    async def mock_load_daily_data(*args, **kwargs):
        return sample_daily_candles
    
    candle_processor.load_daily_data = mock_load_daily_data
    
    # Calculate ATR
    atr = await candle_processor.calculate_atr(sample_daily_candles.iloc[[-1]])
    
    # Verify ATR calculation
    assert isinstance(atr, float)
    assert atr > 0
    logger.info(f"Calculated ATR: {atr}")

@pytest.mark.asyncio
async def test_calculate_morning_range(candle_processor, sample_morning_candles, sample_daily_candles):
    """Test morning range calculation"""
    # Mock the load_daily_data method
    async def mock_load_daily_data(*args, **kwargs):
        return sample_daily_candles
    
    candle_processor.load_daily_data = mock_load_daily_data
    
    # Calculate morning range
    mr_values = await candle_processor.calculate_morning_range(sample_morning_candles)
    
    # Verify morning range values
    assert isinstance(mr_values, dict)
    assert 'mr_high' in mr_values
    assert 'mr_low' in mr_values
    assert 'mr_size' in mr_values
    assert 'mr_value' in mr_values
    assert 'is_valid' in mr_values
    
    # Verify calculations
    assert mr_values['mr_high'] == 110  # Max high from sample data
    assert mr_values['mr_low'] == 95    # Min low from sample data
    assert mr_values['mr_size'] == 15   # High - Low
    assert isinstance(mr_values['mr_value'], float)
    assert isinstance(mr_values['is_valid'], bool)
    
    logger.info(f"Morning Range Values: {mr_values}")

@pytest.mark.asyncio
async def test_morning_range_validation(candle_processor, sample_morning_candles, sample_daily_candles):
    """Test MR value validation"""
    # Mock the load_daily_data method
    async def mock_load_daily_data(*args, **kwargs):
        return sample_daily_candles
    
    candle_processor.load_daily_data = mock_load_daily_data
    
    # Calculate morning range
    mr_values = await candle_processor.calculate_morning_range(sample_morning_candles)
    
    # Verify MR value validation
    if mr_values['mr_value'] > 3:
        assert mr_values['is_valid'] is True
    else:
        assert mr_values['is_valid'] is False
    
    logger.info(f"MR Validation - Value: {mr_values['mr_value']}, Is Valid: {mr_values['is_valid']}")

@pytest.mark.asyncio
async def test_empty_data_handling(candle_processor):
    """Test handling of empty data"""
    empty_df = pd.DataFrame()
    
    # Test empty DataFrame for ATR
    with pytest.raises(ValueError):
        await candle_processor.calculate_atr(empty_df)
    
    # Test empty DataFrame for morning range
    mr_values = await candle_processor.calculate_morning_range(empty_df)
    assert mr_values['mr_high'] == 0
    assert mr_values['mr_low'] == 0
    assert mr_values['mr_size'] == 0
    assert mr_values['mr_value'] == 0
    assert mr_values['is_valid'] is False 