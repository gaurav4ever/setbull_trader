"""
Test script for Phase 1 components of the Morning Range strategy.

This script tests the functionality of:
1. API Client
2. Data Processor
3. Morning Range Calculator
4. Signal Generator
5. Configuration and State Management
6. Signal Models and Types
7. Enhanced Signal Generator

Run this script to verify the basic functionality of the Morning Range strategy components.
"""

import os
import sys
import logging
import pandas as pd
import numpy as np
from datetime import datetime, time, timedelta
import pytz  # Add this import at the top of your file
from mr_strategy.strategy.config import MRStrategyConfig, BreakoutState
from mr_strategy.strategy.models import SignalType, SignalDirection, Signal, SignalGroup

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger('mr_strategy_test')

# Add the parent directory to sys.path to allow importing the package
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

# Define IST timezone
IST = pytz.timezone('Asia/Kolkata')

# Import strategy components
from mr_strategy.data.api_client import ApiClient
from mr_strategy.data.data_processor import CandleProcessor
from mr_strategy.strategy.morning_range import MorningRangeCalculator
from mr_strategy.strategy.signal_generator import SignalGenerator
from mr_strategy.utils.time_utils import is_market_open, get_trading_days_between

# Test configuration
TEST_CONFIG = {
    'api_base_url': 'http://localhost:8083/api/v1',  # Update with your actual API URL
    'instrument_key': 'NSE_EQ|INE777F01014',  # Example: EXICOM
    'start_date': datetime.now(IST) - timedelta(days=1),  # Set to IST
    'end_date': datetime.now(IST),  # Set to IST
    'range_type': '5MR',  # '5MR' or '15MR'
    'buffer_ticks': 5,
    'tick_size': 0.05
}

def test_api_client():
    """Test the API client functionality."""
    logger.info("Testing API Client...")
    
    try:
        # Create API client
        client = ApiClient(base_url=TEST_CONFIG['api_base_url'])
        
        # Test health check
        health = client.get_health()
        logger.info(f"API Health: {health}")
        
        # Format start and end times to RFC3339
        start_time = TEST_CONFIG['start_date'].strftime('%Y-%m-%dT%H:%M:%SZ')  # Format to RFC3339
        end_time = TEST_CONFIG['end_date'].strftime('%Y-%m-%dT%H:%M:%SZ')      # Format to RFC3339

        start_time = "2025-04-07T09:15:00+05:30"
        end_time = "2025-04-07T15:25:00+05:30"
        
        logger.info(f"Formatted start time: {start_time}, end time: {end_time}")
        
        # Test candle data fetching
        candles = client.get_candles(
            instrument_key=TEST_CONFIG['instrument_key'],
            timeframe='5minute',
            start_time=start_time,
            end_time=end_time
        )

        logger.info(f"Candles fetched")
        
        if 'data' in candles and 'data' in candles['data']:
            candle_data = candles['data']['data']  # Access the inner data array
            candle_count = len(candle_data)
            logger.info(f"Successfully fetched {candle_count} candles for {TEST_CONFIG['instrument_key']}")
            return candle_data  # Return the extracted candle data
        else:
            logger.error("No candle data retrieved")
            return None
    
    except Exception as e:
        logger.error(f"API Client test failed: {str(e)}")
        # Since API might not be available in all test environments, 
        # we'll generate mock data for further testing
        logger.info("Generating mock candle data for further tests...")
        return generate_mock_candle_data()

def generate_mock_candle_data():
    """Generate mock candle data for testing when API is not available."""
    logger.info("Generating mock candle data...")
    # Current date
    today = datetime.now().replace(hour=0, minute=0, second=0, microsecond=0)
    
    # Generate a series of candles for a single day
    candles = []
    
    # Morning range candles (9:15 to 9:30)
    start_time = today.replace(hour=9, minute=15)
    for i in range(4):  # 4 five-minute candles
        candle_time = start_time + timedelta(minutes=5*i)
        candles.append({
            'timestamp': candle_time.isoformat(),
            'open': 100 + np.random.uniform(-1, 1),
            'high': 101 + np.random.uniform(0, 1),
            'low': 99 + np.random.uniform(-1, 0),
            'close': 100 + np.random.uniform(-1, 1),
            'volume': int(np.random.uniform(1000, 5000))
        })
    
    # Rest of the day candles
    start_time = today.replace(hour=9, minute=35)
    for i in range(70):  # Rest of the day
        candle_time = start_time + timedelta(minutes=5*i)
        # Create a breakout around 10:30
        if i in range(10, 15):
            high_value = 102 + np.random.uniform(0, 1)  # Breakout above morning range
            low_value = 100 + np.random.uniform(-0.5, 0.5)
        # Create a breakdown around 11:30
        elif i in range(20, 25):
            high_value = 100 + np.random.uniform(0, 1)
            low_value = 98 + np.random.uniform(-1, 0)  # Breakdown below morning range
        else:
            high_value = 101 + np.random.uniform(-0.5, 0.5)
            low_value = 99 + np.random.uniform(-0.5, 0.5)
            
        candles.append({
            'timestamp': candle_time.isoformat(),
            'open': 100 + np.random.uniform(-1, 1),
            'high': high_value,
            'low': low_value,
            'close': 100 + np.random.uniform(-1, 1),
            'volume': int(np.random.uniform(1000, 5000))
        })
    
    # Create mock daily candles for ATR
    daily_candles = []
    for i in range(20):
        day = today - timedelta(days=i)
        daily_candles.append({
            'timestamp': day.isoformat(),
            'open': 100 + np.random.uniform(-5, 5),
            'high': 105 + np.random.uniform(-2, 2),
            'low': 95 + np.random.uniform(-2, 2),
            'close': 100 + np.random.uniform(-5, 5),
            'volume': int(np.random.uniform(100000, 500000))
        })
    
    logger.info("Mock candle data generated successfully.")
    
    return {
        'status': 'success',
        'data': candles,
        'mock_daily_data': {
            'status': 'success',
            'data': daily_candles
        }
    }

def test_data_processor(candle_data):
    """Test the data processor functionality."""
    logger.info("Testing Data Processor...")
    
    try:
        # Create data processor
        processor = CandleProcessor()
        
        # Parse candles
        df = processor.parse_candles(candle_data)
        
        if df.empty:
            logger.error("Failed to parse candles, empty DataFrame")
            return None
        
        logger.info(f"Successfully parsed {len(df)} candles")
        logger.info(f"Candle DataFrame columns: {df.columns.tolist()}")
        
        # Test morning range extraction
        morning_candles, mr_values = processor.extract_morning_range(
            df, 
            range_type=TEST_CONFIG['range_type']
        )
        
        logger.info(f"Morning range: high={mr_values.get('high')}, low={mr_values.get('low')}, size={mr_values.get('size')}")
        logger.info(f"Morning candles count: {len(morning_candles)}")
        
        # Test filtering trading day candles
        today = datetime.now() - timedelta(days=1)  # This is a datetime, not just a date
        day_candles = processor.filter_trading_day_candles(df, today)
        logger.info(f"Trading day candles count: {len(day_candles)}")
        
        # Test ATR calculation if we have daily data
        if 'mock_daily_data' in candle_data:
            daily_df = processor.parse_candles(candle_data['mock_daily_data'])
            atr_df = processor.calculate_atr(daily_df, period=14)
            logger.info(f"ATR calculation: latest ATR = {atr_df['atr'].iloc[-1]}")
        
        return df
        
    except Exception as e:
        logger.error(f"Data Processor test failed: {str(e)}")
        return None

def test_morning_range_calculator(candle_df, daily_df=None):
    """Test the morning range calculator functionality."""
    logger.info("Testing Morning Range Calculator...")
    
    try:
        # Create morning range calculator
        mr_calculator = MorningRangeCalculator(
            range_type=TEST_CONFIG['range_type'],
            market_open=time(9, 15),
            respect_trend=True
        )
        
        # Calculate morning range
        mr_values = mr_calculator.calculate_morning_range(candle_df)
        
        logger.info(f"Morning range: {mr_values}")
        
        # Test range validation
        is_valid = mr_calculator.is_morning_range_valid(mr_values)
        logger.info(f"Morning range is valid: {is_valid}")
        
        # Test entry prices
        entry_prices = mr_calculator.get_entry_prices(
            mr_values,
            buffer_ticks=TEST_CONFIG['buffer_ticks'],
            tick_size=TEST_CONFIG['tick_size']
        )
        
        logger.info(f"Entry prices: {entry_prices}")
        
        # Test ATR ratio and trend if daily data is available
        if daily_df is not None and not daily_df.empty:
            # Calculate ATR ratio
            atr_ratio = mr_calculator.calculate_atr_ratio(mr_values, daily_df)
            logger.info(f"ATR ratio: {atr_ratio}")
            
            # Test ATR ratio validation
            atr_valid = mr_calculator.is_atr_ratio_valid(atr_ratio)
            logger.info(f"ATR ratio is valid: {atr_valid}")
            
            # Test trend determination
            trend = mr_calculator.determine_trend(candle_df)
            logger.info(f"Trend: {trend}")
            
            # Test comprehensive signal validation
            signals = mr_calculator.get_valid_signals(
                mr_values=mr_values,
                daily_candles=daily_df,
                intraday_candles=candle_df,
                buffer_ticks=TEST_CONFIG['buffer_ticks'],
                tick_size=TEST_CONFIG['tick_size']
            )
            
            logger.info(f"Valid signals: {signals}")
        
        return mr_values, mr_calculator
        
    except Exception as e:
        logger.error(f"Morning Range Calculator test failed: {str(e)}")
        return None, None

def test_signal_generator(candle_df, mr_calculator, mr_values, daily_df=None):
    """Test the signal generator functionality."""
    logger.info("Testing Signal Generator...")
    
    try:
        # Create signal generator
        signal_generator = SignalGenerator(
            buffer_ticks=TEST_CONFIG['buffer_ticks'],
            tick_size=TEST_CONFIG['tick_size']
        )
        
        # Get entry prices
        entry_prices = mr_calculator.get_entry_prices(
            mr_values,
            buffer_ticks=TEST_CONFIG['buffer_ticks'],
            tick_size=TEST_CONFIG['tick_size']
        )
        
        logger.debug(f"Entry prices for signal generation: {entry_prices}")
        
        # Scan for breakout
        breakout = signal_generator.scan_for_breakout(candle_df, mr_values, entry_prices)
        
        if breakout:
            logger.info(f"Breakout found: {breakout['breakout_type']} at index {breakout.get('candle_index')}")
        else:
            logger.info("No breakout found")
        
        # Generate entry signal
        signal = signal_generator.generate_entry_signal(
            mr_calculator=mr_calculator,
            candles=candle_df,
            daily_candles=daily_df
        )
        
        if signal:
            logger.info(f"Signal generated: {signal}")
            
            if signal.get('has_breakout', False):
                logger.info(f"Breakout type: {signal['breakout_type']}")
                logger.info(f"Valid long: {signal.get('valid_long', False)}")
                logger.info(f"Valid short: {signal.get('valid_short', False)}")
            else:
                logger.info("No breakout in the provided candle data")
        else:
            logger.info("No signal generated")
        
        # Test signals for day
        today = datetime.now() - timedelta(days=1)
        day_signals = signal_generator.generate_signals_for_day(
            mr_calculator=mr_calculator,
            intraday_candles=candle_df,
            daily_candles=daily_df,
            trading_date=today
        )
        
        logger.info(f"Signals for day {today.date()}: {day_signals}")
        
        return signal
        
    except Exception as e:
        logger.error(f"Signal Generator test failed: {str(e)}")
        return None

def test_configuration():
    """Test the configuration and state management functionality."""
    logger.info("Testing Configuration and State Management...")
    
    try:
        # Test default configuration
        default_config = MRStrategyConfig()
        logger.info(f"Default configuration: {default_config}")
        
        # Test custom configuration
        custom_config = MRStrategyConfig(
            breakout_percentage=0.005,  # 0.5%
            invalidation_percentage=0.008,  # 0.8%
            max_retest_candles=10,
            buffer_ticks=3,
            tick_size=0.1,
            range_type='15MR',
            market_open=time(9, 15),
            respect_trend=False
        )
        logger.info(f"Custom configuration: {custom_config}")
        
        # Test invalid configurations
        try:
            MRStrategyConfig(breakout_percentage=-0.001)
            logger.error("Should have raised ValueError for negative breakout_percentage")
            return False
        except ValueError:
            logger.info("Correctly caught negative breakout_percentage")
        
        try:
            MRStrategyConfig(range_type='invalid')
            logger.error("Should have raised ValueError for invalid range_type")
            return False
        except ValueError:
            logger.info("Correctly caught invalid range_type")
        
        # Test state management
        state = BreakoutState()
        logger.info(f"Initial state: {state.to_dict()}")
        
        # Test state updates
        state.is_breakout_confirmed = True
        state.breakout_type = 'LONG'
        state.breakout_price = 100.0
        state.breakout_time = datetime.now()
        state.mr_level = 99.0
        state.threshold_level = 100.3
        
        logger.info(f"Updated state: {state.to_dict()}")
        
        # Test state validation
        logger.info(f"State is valid: {state.is_valid()}")
        
        # Test state reset
        state.reset()
        logger.info(f"Reset state: {state.to_dict()}")
        logger.info(f"State is valid after reset: {state.is_valid()}")
        
        return True
        
    except Exception as e:
        logger.error(f"Configuration test failed: {str(e)}")
        return False

def test_models():
    """Test the signal models and types functionality."""
    logger.info("Testing Signal Models and Types...")
    
    try:
        # Test SignalType enum
        logger.info("Testing SignalType enum...")
        assert SignalType.IMMEDIATE_BREAKOUT.value == "immediate_breakout"
        assert SignalType.BREAKOUT_CONFIRMATION.value == "breakout_confirmation"
        assert SignalType.RETEST_ENTRY.value == "retest_entry"
        
        # Test SignalDirection enum
        logger.info("Testing SignalDirection enum...")
        assert SignalDirection.LONG.value == "LONG"
        assert SignalDirection.SHORT.value == "SHORT"
        
        # Test Signal creation
        logger.info("Testing Signal creation...")
        timestamp = datetime.now()
        mr_values = {'high': 100.0, 'low': 99.0}
        
        signal = Signal(
            type=SignalType.IMMEDIATE_BREAKOUT,
            direction=SignalDirection.LONG,
            timestamp=timestamp,
            price=100.5,
            mr_values=mr_values,
            metadata={'test': 'value'}
        )
        
        logger.info(f"Created signal: {signal}")
        logger.info(f"Signal as dict: {signal.to_dict()}")
        
        # Test SignalGroup creation
        logger.info("Testing SignalGroup creation...")
        signal_group = SignalGroup(
            signals=[signal],
            start_time=timestamp,
            end_time=timestamp,
            status='active'
        )
        
        logger.info(f"Created signal group: {signal_group}")
        logger.info(f"Signal group as dict: {signal_group.to_dict()}")
        
        # Test adding signal to group
        new_signal = Signal(
            type=SignalType.RETEST_ENTRY,
            direction=SignalDirection.LONG,
            timestamp=timestamp + timedelta(minutes=5),
            price=100.2,
            mr_values=mr_values
        )
        
        signal_group.add_signal(new_signal)
        logger.info(f"Updated signal group: {signal_group}")
        
        # Test invalid signal creation
        try:
            Signal(
                type="invalid_type",  # Should be SignalType
                direction=SignalDirection.LONG,
                timestamp=timestamp,
                price=100.5,
                mr_values=mr_values
            )
            logger.error("Should have raised TypeError for invalid signal type")
            return False
        except TypeError:
            logger.info("Correctly caught invalid signal type")
        
        return True
        
    except Exception as e:
        logger.error(f"Models test failed: {str(e)}")
        return False

def test_enhanced_signal_generator():
    """Test the enhanced signal generator functionality."""
    logger.info("Testing Enhanced Signal Generator...")
    
    try:
        # Create configuration
        config = MRStrategyConfig(
            breakout_percentage=0.003,  # 0.3%
            invalidation_percentage=0.005,  # 0.5%
            buffer_ticks=5,
            tick_size=0.05
        )
        
        # Create signal generator
        signal_generator = SignalGenerator(config=config)
        
        # Create test data
        mr_values = {
            'high': 100.0,
            'low': 99.0,
            'range_type': '5MR'
        }
        
        # Test immediate breakout
        logger.info("Testing immediate breakout...")
        candle = {
            'timestamp': datetime.now(),
            'open': 99.5,
            'high': 100.1,  # Above MR high
            'low': 99.4,
            'close': 100.0
        }
        
        signals = signal_generator.process_candle(candle, mr_values)
        logger.info(f"Generated signals: {[s.to_dict() for s in signals]}")
        
        # Test breakout confirmation
        logger.info("Testing breakout confirmation...")
        candle = {
            'timestamp': datetime.now(),
            'open': 100.1,
            'high': 100.4,
            'low': 100.0,
            'close': 100.3  # Above threshold (100.0 * 1.003 = 100.3)
        }
        
        signals = signal_generator.process_candle(candle, mr_values)
        logger.info(f"Generated signals: {[s.to_dict() for s in signals]}")
        
        # Test retest
        logger.info("Testing retest...")
        candle = {
            'timestamp': datetime.now(),
            'open': 100.2,
            'high': 100.3,
            'low': 99.9,  # Tests MR high
            'close': 100.1  # Closes above MR high
        }
        
        signals = signal_generator.process_candle(candle, mr_values)
        logger.info(f"Generated signals: {[s.to_dict() for s in signals]}")
        
        # Test multiple candles
        logger.info("Testing multiple candles...")
        candles = pd.DataFrame([
            {
                'timestamp': datetime.now() + timedelta(minutes=i),
                'open': 99.5,
                'high': 100.1,
                'low': 99.4,
                'close': 100.0
            } for i in range(5)
        ])
        
        signals = signal_generator.process_candles(candles, mr_values)
        logger.info(f"Generated signals from multiple candles: {[s.to_dict() for s in signals]}")
        
        return True
        
    except Exception as e:
        logger.error(f"Enhanced Signal Generator test failed: {str(e)}")
        return False

def run_all_tests():
    """Run all tests in sequence."""
    logger.info("Starting Phase 1 component tests...")
    
    # Test configuration first
    if not test_configuration():
        logger.error("Configuration tests failed, aborting further tests")
        return False
    
    # Test models
    if test_models():
        logger.error("Models tests failed, aborting further tests")
        return False
    
    # Test enhanced signal generator
    if not test_enhanced_signal_generator():
        logger.error("Enhanced Signal Generator tests failed, aborting further tests")
        return False
    
    # Test API client
    candle_data = test_api_client()
    if candle_data is None:
        logger.error("Candle data could not be retrieved, aborting tests")
        return False
    
    # Test data processor
    candle_df = test_data_processor(candle_data)
    if candle_df is None or candle_df.empty:
        logger.error("Data processing failed, aborting tests")
        return False
    
    # Get daily data for ATR calculations (from mock data or fetch it)
    daily_df = None
    if 'mock_daily_data' in candle_data:
        daily_df = pd.DataFrame(candle_data['mock_daily_data']['data'])
        if 'timestamp' in daily_df.columns:
            daily_df['timestamp'] = pd.to_datetime(daily_df['timestamp'])
    
    # Test morning range calculator
    mr_values, mr_calculator = test_morning_range_calculator(candle_df, daily_df)
    if mr_values is None or mr_calculator is None:
        logger.error("Morning range calculation failed, aborting tests")
        return False
    
    # Test signal generator
    signal = test_signal_generator(candle_df, mr_calculator, mr_values, daily_df)
    
    logger.info("All Phase 1 component tests completed!")
    return True

if __name__ == "__main__":
    success = run_all_tests()
    if success:
        logger.info("✅ All tests passed successfully!")
    else:
        logger.error("❌ Some tests failed, check the logs for details") 