#!/usr/bin/env python3
"""
Test script for Volatility Squeeze Analyzer
===========================================

This script tests the basic functionality of the volatility squeeze analyzer
without running the full analysis.
"""

import sys
import os
sys.path.append(os.path.dirname(os.path.abspath(__file__)))

from volatility_squeeze_analyzer import (
    ConfigurationManager, 
    DatabaseManager, 
    LoggingManager,
    PerformanceMonitor,
    DataValidator,
    BollingerBandCalculator,
    SqueezeDetector,
    MetricsCalculator
)
import polars as pl
import logging
from datetime import datetime, timedelta
import math

def test_configuration():
    """Test configuration management."""
    print("Testing Configuration Management...")
    config = ConfigurationManager()
    
    # Test database config
    assert config.db_config['host'] == '127.0.0.1'
    assert config.db_config['database'] == 'setbull_trader'
    
    # Test trading params
    assert config.trading_params['bb_period'] == 20
    assert config.trading_params['bb_std_dev'] == 2.0
    assert config.trading_params['lookback_period'] == 126
    
    print("✅ Configuration management test passed")

def test_data_validation():
    """Test data validation functionality."""
    print("Testing Data Validation...")
    config = ConfigurationManager()
    validator = DataValidator(config)
    
    # Create test data with timestamp
    dates = [datetime.now() - timedelta(days=i) for i in range(3, 0, -1)]
    test_data = pl.DataFrame({
        "timestamp": dates,
        "open": [100, 101, 102],
        "high": [105, 106, 107],
        "low": [98, 99, 100],
        "close": [103, 104, 105],
        "volume": [1000, 1100, 1200]
    })
    
    # Test price validation
    assert validator.validate_price_data(test_data) == True
    
    # Test volume validation
    assert validator.validate_volume_data(test_data) == True
    
    # Test data completeness
    assert validator.check_data_completeness(test_data, 2) == True
    
    print("✅ Data validation test passed")

def test_bollinger_band_calculation():
    """Test Bollinger Band calculation."""
    print("Testing Bollinger Band Calculation...")
    config = ConfigurationManager()
    bb_calc = BollingerBandCalculator(config)
    
    # Create test data with more points for BB calculation
    dates = [datetime.now() - timedelta(days=i) for i in range(20, -1, -1)]
    test_data = pl.DataFrame({
        "timestamp": dates,
        "close": [100 + i for i in range(21)]
    })
    
    # Calculate Bollinger Bands
    result = bb_calc.calculate_bollinger_bands(test_data)
    
    # Check that BBW is calculated
    assert "bb_width" in result.columns
    assert "bb_upper" in result.columns
    assert "bb_lower" in result.columns
    
    print("✅ Bollinger Band calculation test passed")

def test_metrics_calculation():
    """Test metrics calculation."""
    print("Testing Metrics Calculation...")
    config = ConfigurationManager()
    metrics_calc = MetricsCalculator(config)
    
    # Test breakdown readiness
    breakdown = metrics_calc.calculate_breakdown_readiness(0.8)
    assert math.isclose(breakdown, 0.2, rel_tol=1e-9)
    
    # Test percentile rank
    historical_bbw = [0.02, 0.03, 0.04, 0.05, 0.06]
    percentile = metrics_calc.calculate_percentile_rank(0.04, historical_bbw)
    assert percentile == 40.0  # 0.04 is at 40th percentile
    
    print("✅ Metrics calculation test passed")

def test_performance_monitor():
    """Test performance monitoring."""
    print("Testing Performance Monitor...")
    monitor = PerformanceMonitor()
    
    # Test timer functionality
    monitor.start_timer("test_operation")
    import time
    time.sleep(0.1)  # Simulate some work
    duration = monitor.end_timer("test_operation")
    
    assert duration > 0
    assert "test_operation" in monitor.get_metrics()
    
    print("✅ Performance monitor test passed")

def main():
    """Run all tests."""
    print("🧪 Running Volatility Squeeze Analyzer Tests")
    print("=" * 50)
    
    try:
        test_configuration()
        test_data_validation()
        test_bollinger_band_calculation()
        test_metrics_calculation()
        test_performance_monitor()
        
        print("\n🎉 All tests passed successfully!")
        print("The volatility squeeze analyzer is ready for use.")
        
    except Exception as e:
        print(f"\n❌ Test failed: {e}")
        import traceback
        traceback.print_exc()
        return False
    
    return True

if __name__ == "__main__":
    main() 