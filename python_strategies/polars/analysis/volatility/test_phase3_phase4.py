#!/usr/bin/env python3
"""
Test Script for Phase 3 and Phase 4: Individual Analysis and Performance Profiling
==================================================================================

This script tests the new Phase 3 (Individual Analysis) and Phase 4 (Historical Performance Analysis)
functionality of the volatility squeeze analyzer.

Author: Setbull Trader Team
Version: 1.0.0
Date: 2024
"""

import sys
import os
import unittest
import tempfile
import json
from datetime import datetime, timedelta
import polars as pl
import pandas as pd
import numpy as np

# Add the current directory to the path to import the analyzer
sys.path.append(os.path.dirname(os.path.abspath(__file__)))

from volatility_squeeze_analyzer import (
    ConfigurationManager, IndividualAnalyzer, BacktestEngine, 
    RangeOptimizer, PerformanceProfileAnalyzer, BollingerBandCalculator
)

class TestPhase3IndividualAnalysis(unittest.TestCase):
    """Test Phase 3: Individual Analysis functionality."""
    
    def setUp(self):
        """Set up test data and configuration."""
        self.config = ConfigurationManager()
        
        # Create mock data for testing
        self.mock_data = self._create_mock_stock_data()
        
        # Initialize analyzers
        self.individual_analyzer = IndividualAnalyzer(self.config)
        self.bb_calculator = BollingerBandCalculator(self.config)
    
    def _create_mock_stock_data(self):
        """Create realistic mock stock data for testing."""
        # Generate 300 days of data
        dates = pd.date_range(start='2023-01-01', periods=300, freq='D')
        
        # Create realistic price data with some volatility patterns
        np.random.seed(42)  # For reproducible results
        
        # Base price trend
        base_price = 100
        trend = np.linspace(0, 20, 300)  # Upward trend
        
        # Add volatility cycles
        volatility_cycles = 10 * np.sin(np.linspace(0, 4*np.pi, 300))
        
        # Add random noise
        noise = np.random.normal(0, 2, 300)
        
        # Generate prices
        prices = base_price + trend + volatility_cycles + noise
        prices = np.maximum(prices, 10)  # Ensure minimum price
        
        # Generate OHLC data
        data = []
        for i, (date, price) in enumerate(zip(dates, prices)):
            # Create realistic OHLC from close price
            daily_volatility = np.random.uniform(0.5, 2.0)
            high = price * (1 + daily_volatility/100)
            low = price * (1 - daily_volatility/100)
            open_price = price * (1 + np.random.uniform(-1, 1)/100)
            
            # Generate volume (higher for volatile periods)
            volume = int(np.random.uniform(50000, 200000) * (1 + daily_volatility/100))
            
            data.append({
                'timestamp': date,
                'open': round(open_price, 2),
                'high': round(high, 2),
                'low': round(low, 2),
                'close': round(price, 2),
                'volume': volume
            })
        
        return pl.DataFrame(data)
    
    def test_individual_analyzer_initialization(self):
        """Test IndividualAnalyzer initialization."""
        analyzer = IndividualAnalyzer(self.config)
        self.assertIsNotNone(analyzer)
        self.assertEqual(analyzer.config, self.config)
    
    def test_historical_percentiles_calculation(self):
        """Test historical percentiles calculation."""
        # Calculate Bollinger Bands first
        df_with_bb = self.bb_calculator.calculate_bollinger_bands(self.mock_data)
        
        # Get historical data (last 126 days)
        historical_df = df_with_bb.tail(126)
        
        # Calculate percentiles
        percentiles = self.individual_analyzer._calculate_historical_percentiles(historical_df)
        
        # Verify percentiles structure
        self.assertIsInstance(percentiles, dict)
        self.assertIn('current_bbw', percentiles)
        self.assertIn('percentile_10', percentiles)
        self.assertIn('percentile_25', percentiles)
        self.assertIn('percentile_50', percentiles)
        self.assertIn('percentile_75', percentiles)
        self.assertIn('percentile_90', percentiles)
        self.assertIn('current_percentile_rank', percentiles)
        
        # Verify percentile values are logical
        self.assertGreater(percentiles['percentile_10'], 0)
        self.assertLess(percentiles['percentile_10'], percentiles['percentile_25'])
        self.assertLess(percentiles['percentile_25'], percentiles['percentile_50'])
        self.assertLess(percentiles['percentile_50'], percentiles['percentile_75'])
        self.assertLess(percentiles['percentile_75'], percentiles['percentile_90'])
        
        # Verify current percentile rank is between 0 and 100
        self.assertGreaterEqual(percentiles['current_percentile_rank'], 0)
        self.assertLessEqual(percentiles['current_percentile_rank'], 100)
    
    def test_contraction_confirmation_analysis(self):
        """Test contraction confirmation analysis."""
        # Calculate Bollinger Bands first
        df_with_bb = self.bb_calculator.calculate_bollinger_bands(self.mock_data)
        
        # Analyze contraction
        contraction = self.individual_analyzer._analyze_contraction_confirmation(df_with_bb)
        
        # Verify contraction analysis structure
        self.assertIsInstance(contraction, dict)
        self.assertIn('bbw_decline_percent', contraction)
        self.assertIn('consecutive_declines', contraction)
        self.assertIn('volume_decline_percent', contraction)
        self.assertIn('is_contracting', contraction)
        self.assertIn('volume_confirms', contraction)
        self.assertIn('contraction_strength', contraction)
        
        # Verify contraction strength is one of expected values
        self.assertIn(contraction['contraction_strength'], ['STRONG', 'MODERATE', 'WEAK'])
    
    def test_tradable_range_analysis(self):
        """Test tradable range analysis."""
        # Calculate Bollinger Bands first
        df_with_bb = self.bb_calculator.calculate_bollinger_bands(self.mock_data)
        
        # Analyze tradable range
        range_analysis = self.individual_analyzer._analyze_tradable_range(df_with_bb)
        
        # Verify range analysis structure
        self.assertIsInstance(range_analysis, dict)
        self.assertIn('current_range', range_analysis)
        self.assertIn('ranges', range_analysis)
        self.assertIn('current_bbw', range_analysis)
        self.assertIn('optimal_range_min', range_analysis)
        self.assertIn('optimal_range_max', range_analysis)
        self.assertIn('optimal_range_avg', range_analysis)
        self.assertIn('distance_to_optimal', range_analysis)
        
        # Verify ranges structure
        ranges = range_analysis['ranges']
        self.assertIn('ultra_tight', ranges)
        self.assertIn('tight', ranges)
        self.assertIn('normal', ranges)
        self.assertIn('wide', ranges)
        self.assertIn('ultra_wide', ranges)
        
        # Verify current range is one of expected values
        expected_ranges = ['ULTRA_TIGHT', 'TIGHT', 'NORMAL', 'WIDE', 'ULTRA_WIDE', 'UNKNOWN']
        self.assertIn(range_analysis['current_range'], expected_ranges)
    
    def test_performance_profile_generation(self):
        """Test performance profile generation."""
        # Calculate Bollinger Bands first
        df_with_bb = self.bb_calculator.calculate_bollinger_bands(self.mock_data)
        
        # Generate performance profile
        performance_profile = self.individual_analyzer._generate_performance_profile(df_with_bb)
        
        # Verify performance profile structure
        self.assertIsInstance(performance_profile, dict)
        self.assertIn('performance_by_range', performance_profile)
        self.assertIn('best_performing_range', performance_profile)
        self.assertIn('best_win_rate', performance_profile)
        self.assertIn('total_analysis_periods', performance_profile)
        self.assertIn('current_bbw_percentile', performance_profile)
        
        # Verify performance by range structure
        performance_by_range = performance_profile['performance_by_range']
        if performance_by_range:  # May be empty for some test data
            for range_name, metrics in performance_by_range.items():
                self.assertIn('avg_return', metrics)
                self.assertIn('win_rate', metrics)
                self.assertIn('max_return', metrics)
                self.assertIn('min_return', metrics)
                self.assertIn('periods', metrics)
    
    def test_analysis_summary_generation(self):
        """Test analysis summary generation."""
        # Create mock data for summary generation
        current_bbw = 0.05
        percentiles = {
            'current_percentile_rank': 15.0,
            'percentile_10': 0.03,
            'percentile_25': 0.06
        }
        contraction = {
            'is_contracting': True,
            'volume_confirms': True,
            'contraction_strength': 'STRONG'
        }
        tradable_range = {
            'current_range': 'TIGHT'
        }
        performance = {
            'best_performing_range': 'tight'
        }
        
        # Generate summary
        summary = self.individual_analyzer._generate_analysis_summary(
            current_bbw, percentiles, contraction, tradable_range, performance
        )
        
        # Verify summary structure
        self.assertIsInstance(summary, dict)
        self.assertIn('squeeze_status', summary)
        self.assertIn('current_percentile', summary)
        self.assertIn('recommendation', summary)
        self.assertIn('confidence', summary)
        self.assertIn('risk_level', summary)
        self.assertIn('contraction_strength', summary)
        self.assertIn('optimal_range_status', summary)
        self.assertIn('best_performing_range', summary)
        
        # Verify expected values for this test case
        self.assertEqual(summary['squeeze_status'], 'IN_SQUEEZE')
        self.assertEqual(summary['recommendation'], 'STRONG_BUY')
        self.assertEqual(summary['confidence'], 'HIGH')
        self.assertEqual(summary['risk_level'], 'LOW')
    
    def test_complete_individual_analysis(self):
        """Test complete individual analysis workflow."""
        # Run complete individual analysis
        result = self.individual_analyzer.analyze_individual_stock(
            'TEST_SYMBOL', 'TEST_SYMBOL', self.mock_data
        )
        
        # Verify result structure
        self.assertIsNotNone(result)
        self.assertIsInstance(result, dict)
        self.assertIn('instrument_key', result)
        self.assertIn('symbol', result)
        self.assertIn('analysis_date', result)
        self.assertIn('latest_close', result)
        self.assertIn('latest_bb_width', result)
        self.assertIn('historical_percentiles', result)
        self.assertIn('contraction_analysis', result)
        self.assertIn('tradable_range_analysis', result)
        self.assertIn('performance_profile', result)
        self.assertIn('analysis_summary', result)
        
        # Verify data types
        self.assertEqual(result['symbol'], 'TEST_SYMBOL')
        self.assertIsInstance(result['latest_close'], (int, float))
        self.assertIsInstance(result['latest_bb_width'], (int, float))
        self.assertGreater(result['latest_bb_width'], 0)

class TestPhase4PerformanceAnalysis(unittest.TestCase):
    """Test Phase 4: Historical Performance Analysis functionality."""
    
    def setUp(self):
        """Set up test data and configuration."""
        self.config = ConfigurationManager()
        
        # Create mock data for testing
        self.mock_data = self._create_mock_stock_data()
        
        # Initialize analyzers
        self.backtest_engine = BacktestEngine(self.config)
        self.range_optimizer = RangeOptimizer(self.config)
        self.performance_analyzer = PerformanceProfileAnalyzer(self.config)
    
    def _create_mock_stock_data(self):
        """Create realistic mock stock data with squeeze patterns for testing."""
        # Generate 300 days of data
        dates = pd.date_range(start='2023-01-01', periods=300, freq='D')
        
        # Create realistic price data with squeeze patterns
        np.random.seed(42)  # For reproducible results
        
        # Base price trend
        base_price = 100
        trend = np.linspace(0, 30, 300)  # Upward trend
        
        # Create squeeze patterns (periods of low volatility)
        squeeze_periods = []
        for i in range(0, 300, 50):  # Every 50 days
            squeeze_periods.extend(range(i, min(i+20, 300)))
        
        # Generate prices with squeeze patterns
        prices = []
        for i in range(300):
            if i in squeeze_periods:
                # Low volatility during squeeze
                daily_volatility = np.random.uniform(0.1, 0.5)
            else:
                # Normal volatility
                daily_volatility = np.random.uniform(1.0, 3.0)
            
            if i == 0:
                price = base_price
            else:
                price = prices[-1] * (1 + np.random.uniform(-daily_volatility, daily_volatility)/100)
            
            prices.append(max(price, 10))  # Ensure minimum price
        
        # Generate OHLC data
        data = []
        for i, (date, price) in enumerate(zip(dates, prices)):
            # Create realistic OHLC from close price
            daily_volatility = np.random.uniform(0.5, 2.0)
            high = price * (1 + daily_volatility/100)
            low = price * (1 - daily_volatility/100)
            open_price = price * (1 + np.random.uniform(-1, 1)/100)
            
            # Generate volume (higher for volatile periods)
            volume = int(np.random.uniform(50000, 200000) * (1 + daily_volatility/100))
            
            data.append({
                'timestamp': date,
                'open': round(open_price, 2),
                'high': round(high, 2),
                'low': round(low, 2),
                'close': round(price, 2),
                'volume': volume
            })
        
        return pl.DataFrame(data)
    
    def test_backtest_engine_initialization(self):
        """Test BacktestEngine initialization."""
        engine = BacktestEngine(self.config)
        self.assertIsNotNone(engine)
        self.assertEqual(engine.config, self.config)
    
    def test_squeeze_entry_identification(self):
        """Test squeeze entry identification."""
        # Calculate Bollinger Bands first
        bb_calculator = BollingerBandCalculator(self.config)
        df_with_bb = bb_calculator.calculate_bollinger_bands(self.mock_data)
        
        # Identify squeeze entries
        entries = self.backtest_engine._identify_squeeze_entries(df_with_bb)
        
        # Verify entries structure
        self.assertIsInstance(entries, list)
        
        if entries:  # May be empty for some test data
            entry = entries[0]
            self.assertIn('entry_date', entry)
            self.assertIn('entry_price', entry)
            self.assertIn('entry_bbw', entry)
            self.assertIn('threshold', entry)
            self.assertIn('bbw_decline', entry)
            self.assertIn('entry_index', entry)
            
            # Verify data types
            self.assertIsInstance(entry['entry_price'], (int, float))
            self.assertIsInstance(entry['entry_bbw'], (int, float))
            self.assertIsInstance(entry['threshold'], (int, float))
            self.assertIsInstance(entry['bbw_decline'], (int, float))
            self.assertIsInstance(entry['entry_index'], int)
    
    def test_trade_return_calculation(self):
        """Test trade return calculation."""
        # Create mock entries
        entries = [
            {
                'entry_date': datetime.now(),
                'entry_price': 100.0,
                'entry_bbw': 0.05,
                'threshold': 0.06,
                'bbw_decline': 10.0,
                'entry_index': 50
            }
        ]
        
        # Calculate Bollinger Bands first
        bb_calculator = BollingerBandCalculator(self.config)
        df_with_bb = bb_calculator.calculate_bollinger_bands(self.mock_data)
        
        # Calculate trade returns
        trade_results = self.backtest_engine._calculate_trade_returns(df_with_bb, entries)
        
        # Verify trade results structure
        self.assertIsInstance(trade_results, list)
        
        if trade_results:
            trade = trade_results[0]
            self.assertIn('entry_date', trade)
            self.assertIn('exit_date', trade)
            self.assertIn('entry_price', trade)
            self.assertIn('exit_price', trade)
            self.assertIn('return_pct', trade)
            self.assertIn('hold_days', trade)
            self.assertIn('exit_reason', trade)
            self.assertIn('entry_bbw', trade)
            self.assertIn('bbw_decline', trade)
            
            # Verify data types
            self.assertIsInstance(trade['entry_price'], (int, float))
            self.assertIsInstance(trade['exit_price'], (int, float))
            self.assertIsInstance(trade['return_pct'], (int, float))
            self.assertIsInstance(trade['hold_days'], int)
            self.assertIn(trade['exit_reason'], ['MAX_HOLD', 'END_OF_DATA'])
    
    def test_performance_metrics_calculation(self):
        """Test performance metrics calculation."""
        # Create mock trade results
        trade_results = [
            {'return_pct': 5.0, 'hold_days': 3},
            {'return_pct': -2.0, 'hold_days': 2},
            {'return_pct': 8.0, 'hold_days': 4},
            {'return_pct': 3.0, 'hold_days': 1},
            {'return_pct': -1.0, 'hold_days': 2}
        ]
        
        # Calculate performance metrics
        metrics = self.backtest_engine._calculate_performance_metrics(trade_results)
        
        # Verify metrics structure
        self.assertIsInstance(metrics, dict)
        self.assertIn('total_trades', metrics)
        self.assertIn('winning_trades', metrics)
        self.assertIn('losing_trades', metrics)
        self.assertIn('win_rate', metrics)
        self.assertIn('avg_return', metrics)
        self.assertIn('avg_win', metrics)
        self.assertIn('avg_loss', metrics)
        self.assertIn('max_win', metrics)
        self.assertIn('max_loss', metrics)
        self.assertIn('total_return', metrics)
        self.assertIn('profit_factor', metrics)
        self.assertIn('sharpe_ratio', metrics)
        self.assertIn('max_drawdown', metrics)
        
        # Verify expected values for this test case
        self.assertEqual(metrics['total_trades'], 5)
        self.assertEqual(metrics['winning_trades'], 3)
        self.assertEqual(metrics['losing_trades'], 2)
        self.assertEqual(metrics['win_rate'], 60.0)  # 3/5 * 100
        self.assertAlmostEqual(metrics['avg_return'], 2.6)  # (5-2+8+3-1)/5
        self.assertAlmostEqual(metrics['avg_win'], 5.33, places=1)  # (5+8+3)/3
        self.assertAlmostEqual(metrics['avg_loss'], -1.5)  # (-2-1)/2
    
    def test_risk_metrics_calculation(self):
        """Test risk metrics calculation."""
        # Create mock trade results
        trade_results = [
            {'return_pct': 5.0, 'hold_days': 3},
            {'return_pct': -2.0, 'hold_days': 2},
            {'return_pct': 8.0, 'hold_days': 4},
            {'return_pct': 3.0, 'hold_days': 1},
            {'return_pct': -1.0, 'hold_days': 2}
        ]
        
        # Calculate risk metrics
        risk_metrics = self.backtest_engine._calculate_risk_metrics(trade_results)
        
        # Verify risk metrics structure
        self.assertIsInstance(risk_metrics, dict)
        self.assertIn('volatility', risk_metrics)
        self.assertIn('var_95', risk_metrics)
        self.assertIn('max_consecutive_losses', risk_metrics)
        self.assertIn('avg_hold_days', risk_metrics)
        self.assertIn('risk_reward_ratio', risk_metrics)
        
        # Verify data types
        self.assertIsInstance(risk_metrics['volatility'], (int, float))
        self.assertIsInstance(risk_metrics['var_95'], (int, float))
        self.assertIsInstance(risk_metrics['max_consecutive_losses'], int)
        self.assertIsInstance(risk_metrics['avg_hold_days'], (int, float))
        self.assertIsInstance(risk_metrics['risk_reward_ratio'], (int, float))
        
        # Verify expected values for this test case
        self.assertEqual(risk_metrics['avg_hold_days'], 2.4)  # (3+2+4+1+2)/5
        self.assertGreaterEqual(risk_metrics['max_consecutive_losses'], 0)
    
    def test_range_optimizer_initialization(self):
        """Test RangeOptimizer initialization."""
        optimizer = RangeOptimizer(self.config)
        self.assertIsNotNone(optimizer)
        self.assertEqual(optimizer.config, self.config)
    
    def test_bbw_range_testing(self):
        """Test BBW range testing."""
        # Calculate Bollinger Bands first
        bb_calculator = BollingerBandCalculator(self.config)
        df_with_bb = bb_calculator.calculate_bollinger_bands(self.mock_data)
        
        # Test BBW ranges
        range_performances = self.range_optimizer._test_bbw_ranges(df_with_bb)
        
        # Verify range performances structure
        self.assertIsInstance(range_performances, dict)
        
        if range_performances:  # May be empty for some test data
            for range_name, range_data in range_performances.items():
                self.assertIn('min_bbw', range_data)
                self.assertIn('max_bbw', range_data)
                self.assertIn('min_percentile', range_data)
                self.assertIn('max_percentile', range_data)
                self.assertIn('performance', range_data)
                
                # Verify performance structure
                performance = range_data['performance']
                self.assertIn('avg_return', performance)
                self.assertIn('win_rate', performance)
                self.assertIn('max_return', performance)
                self.assertIn('min_return', performance)
                self.assertIn('periods', performance)
    
    def test_best_range_selection(self):
        """Test best range selection."""
        # Create mock range performances
        range_performances = {
            '5-10%': {
                'performance': {'win_rate': 60, 'avg_return': 2.0, 'periods': 20}
            },
            '10-15%': {
                'performance': {'win_rate': 55, 'avg_return': 1.5, 'periods': 25}
            },
            '15-20%': {
                'performance': {'win_rate': 50, 'avg_return': 1.0, 'periods': 30}
            }
        }
        
        # Select best range
        best_range = self.range_optimizer._select_best_range(range_performances)
        
        # Verify best range structure
        self.assertIsNotNone(best_range)
        self.assertIn('range_name', best_range)
        self.assertIn('range_data', best_range)
        self.assertIn('score', best_range)
        
        # Verify expected selection (5-10% should be best based on win_rate * avg_return)
        self.assertEqual(best_range['range_name'], '5-10%')
    
    def test_performance_profile_analyzer_initialization(self):
        """Test PerformanceProfileAnalyzer initialization."""
        analyzer = PerformanceProfileAnalyzer(self.config)
        self.assertIsNotNone(analyzer)
        self.assertEqual(analyzer.config, self.config)
        self.assertIsNotNone(analyzer.backtest_engine)
        self.assertIsNotNone(analyzer.range_optimizer)
    
    def test_complete_performance_profile_generation(self):
        """Test complete performance profile generation."""
        # Generate performance profile
        profile = self.performance_analyzer.generate_performance_profile(
            'TEST_SYMBOL', 'TEST_SYMBOL', self.mock_data
        )
        
        # Verify profile structure
        self.assertIsNotNone(profile)
        self.assertIsInstance(profile, dict)
        self.assertIn('instrument_key', profile)
        self.assertIn('symbol', profile)
        self.assertIn('generation_date', profile)
        self.assertIn('backtest_result', profile)
        self.assertIn('optimal_range', profile)
        self.assertIn('profile_summary', profile)
        
        # Verify data types
        self.assertEqual(profile['symbol'], 'TEST_SYMBOL')
        self.assertIsInstance(profile['generation_date'], str)
        
        # Verify backtest result structure
        if profile['backtest_result']:
            backtest = profile['backtest_result']
            self.assertIn('symbol', backtest)
            self.assertIn('total_trades', backtest)
            self.assertIn('performance_metrics', backtest)
            self.assertIn('risk_metrics', backtest)
            self.assertIn('backtest_summary', backtest)
        
        # Verify optimal range structure
        if profile['optimal_range']:
            optimal = profile['optimal_range']
            self.assertIn('symbol', optimal)
            self.assertIn('best_range', optimal)
            self.assertIn('range_performances', optimal)
            self.assertIn('optimization_summary', optimal)
        
        # Verify profile summary structure
        summary = profile['profile_summary']
        self.assertIn('overall_status', summary)
        self.assertIn('strategy_viability', summary)
        self.assertIn('optimization_status', summary)
        self.assertIn('recommendation', summary)
        self.assertIn('key_metrics', summary)

class TestIntegration(unittest.TestCase):
    """Test integration between Phase 3 and Phase 4 components."""
    
    def setUp(self):
        """Set up test data and configuration."""
        self.config = ConfigurationManager()
        
        # Create mock data
        self.mock_data = self._create_mock_stock_data()
        
        # Initialize all analyzers
        self.individual_analyzer = IndividualAnalyzer(self.config)
        self.performance_analyzer = PerformanceProfileAnalyzer(self.config)
    
    def _create_mock_stock_data(self):
        """Create mock stock data for integration testing."""
        # Generate 300 days of data
        dates = pd.date_range(start='2023-01-01', periods=300, freq='D')
        
        # Create realistic price data
        np.random.seed(42)
        base_price = 100
        prices = [base_price]
        
        for i in range(1, 300):
            # Add some squeeze patterns
            if i % 50 < 20:  # Every 50 days, 20 days of low volatility
                volatility = np.random.uniform(0.1, 0.5)
            else:
                volatility = np.random.uniform(1.0, 3.0)
            
            price_change = np.random.uniform(-volatility, volatility)
            new_price = prices[-1] * (1 + price_change/100)
            prices.append(max(new_price, 10))
        
        # Generate OHLC data
        data = []
        for i, (date, price) in enumerate(zip(dates, prices)):
            daily_volatility = np.random.uniform(0.5, 2.0)
            high = price * (1 + daily_volatility/100)
            low = price * (1 - daily_volatility/100)
            open_price = price * (1 + np.random.uniform(-1, 1)/100)
            volume = int(np.random.uniform(50000, 200000))
            
            data.append({
                'timestamp': date,
                'open': round(open_price, 2),
                'high': round(high, 2),
                'low': round(low, 2),
                'close': round(price, 2),
                'volume': volume
            })
        
        return pl.DataFrame(data)
    
    def test_individual_and_performance_integration(self):
        """Test integration between individual analysis and performance profiling."""
        # Run individual analysis
        individual_result = self.individual_analyzer.analyze_individual_stock(
            'TEST_SYMBOL', 'TEST_SYMBOL', self.mock_data
        )
        
        # Run performance profiling
        performance_result = self.performance_analyzer.generate_performance_profile(
            'TEST_SYMBOL', 'TEST_SYMBOL', self.mock_data
        )
        
        # Verify both results are consistent
        self.assertIsNotNone(individual_result)
        self.assertIsNotNone(performance_result)
        
        # Verify symbol consistency
        self.assertEqual(individual_result['symbol'], 'TEST_SYMBOL')
        self.assertEqual(performance_result['symbol'], 'TEST_SYMBOL')
        
        # Verify BBW consistency (should be same for same data)
        individual_bbw = individual_result['latest_bb_width']
        performance_bbw = performance_result['backtest_result']['squeeze_entries'][0]['entry_bbw'] if performance_result['backtest_result']['squeeze_entries'] else None
        
        if performance_bbw is not None:
            # Should be close (within 1% due to different calculation methods)
            self.assertAlmostEqual(individual_bbw, performance_bbw, delta=individual_bbw * 0.01)
    
    def test_data_consistency_across_phases(self):
        """Test data consistency across different phases."""
        # Calculate Bollinger Bands using the calculator
        bb_calculator = BollingerBandCalculator(self.config)
        df_with_bb = bb_calculator.calculate_bollinger_bands(self.mock_data)
        
        # Get latest BBW from different sources
        latest_bbw_1 = df_with_bb.tail(1).select("bb_width").item()
        
        # Get BBW from individual analysis
        individual_result = self.individual_analyzer.analyze_individual_stock(
            'TEST_SYMBOL', 'TEST_SYMBOL', self.mock_data
        )
        latest_bbw_2 = individual_result['latest_bb_width']
        
        # Verify consistency
        self.assertAlmostEqual(latest_bbw_1, latest_bbw_2, places=6)
    
    def test_error_handling(self):
        """Test error handling in both phases."""
        # Test with empty data
        empty_data = pl.DataFrame()
        
        # Individual analysis should handle empty data gracefully
        individual_result = self.individual_analyzer.analyze_individual_stock(
            'TEST_SYMBOL', 'TEST_SYMBOL', empty_data
        )
        self.assertIsNone(individual_result)
        
        # Performance profiling should handle empty data gracefully
        performance_result = self.performance_analyzer.generate_performance_profile(
            'TEST_SYMBOL', 'TEST_SYMBOL', empty_data
        )
        self.assertIsNone(performance_result)
        
        # Test with insufficient data
        insufficient_data = pl.DataFrame({
            'timestamp': [datetime.now()],
            'open': [100],
            'high': [101],
            'low': [99],
            'close': [100.5],
            'volume': [100000]
        })
        
        # Both should handle insufficient data gracefully
        individual_result = self.individual_analyzer.analyze_individual_stock(
            'TEST_SYMBOL', 'TEST_SYMBOL', insufficient_data
        )
        self.assertIsNone(individual_result)
        
        performance_result = self.performance_analyzer.generate_performance_profile(
            'TEST_SYMBOL', 'TEST_SYMBOL', insufficient_data
        )
        self.assertIsNone(performance_result)

def run_performance_tests():
    """Run performance tests to ensure reasonable execution times."""
    print("\n" + "="*60)
    print("PERFORMANCE TESTS")
    print("="*60)
    
    config = ConfigurationManager()
    
    # Create larger dataset for performance testing
    dates = pd.date_range(start='2022-01-01', periods=500, freq='D')
    np.random.seed(42)
    
    base_price = 100
    prices = [base_price]
    for i in range(1, 500):
        volatility = np.random.uniform(0.5, 3.0)
        price_change = np.random.uniform(-volatility, volatility)
        new_price = prices[-1] * (1 + price_change/100)
        prices.append(max(new_price, 10))
    
    data = []
    for i, (date, price) in enumerate(zip(dates, prices)):
        daily_volatility = np.random.uniform(0.5, 2.0)
        high = price * (1 + daily_volatility/100)
        low = price * (1 - daily_volatility/100)
        open_price = price * (1 + np.random.uniform(-1, 1)/100)
        volume = int(np.random.uniform(50000, 200000))
        
        data.append({
            'timestamp': date,
            'open': round(open_price, 2),
            'high': round(high, 2),
            'low': round(low, 2),
            'close': round(price, 2),
            'volume': volume
        })
    
    large_dataset = pl.DataFrame(data)
    
    # Test individual analysis performance
    print("Testing Individual Analysis Performance...")
    individual_analyzer = IndividualAnalyzer(config)
    
    import time
    start_time = time.time()
    result = individual_analyzer.analyze_individual_stock(
        'PERF_TEST', 'PERF_TEST', large_dataset
    )
    individual_time = time.time() - start_time
    
    print(f"Individual Analysis: {individual_time:.2f} seconds")
    print(f"Result: {'SUCCESS' if result else 'FAILED'}")
    
    # Test performance profiling performance
    print("\nTesting Performance Profiling Performance...")
    performance_analyzer = PerformanceProfileAnalyzer(config)
    
    start_time = time.time()
    result = performance_analyzer.generate_performance_profile(
        'PERF_TEST', 'PERF_TEST', large_dataset
    )
    profiling_time = time.time() - start_time
    
    print(f"Performance Profiling: {profiling_time:.2f} seconds")
    print(f"Result: {'SUCCESS' if result else 'FAILED'}")
    
    # Performance benchmarks
    print(f"\nPerformance Benchmarks:")
    print(f"Individual Analysis: {'PASS' if individual_time < 5.0 else 'FAIL'} (< 5.0s)")
    print(f"Performance Profiling: {'PASS' if profiling_time < 10.0 else 'FAIL'} (< 10.0s)")
    
    return individual_time < 5.0 and profiling_time < 10.0

if __name__ == "__main__":
    # Run unit tests
    print("Running Phase 3 and Phase 4 Unit Tests...")
    unittest.main(argv=[''], exit=False, verbosity=2)
    
    # Run performance tests
    performance_passed = run_performance_tests()
    
    print("\n" + "="*60)
    print("TEST SUMMARY")
    print("="*60)
    print(f"Performance Tests: {'PASSED' if performance_passed else 'FAILED'}")
    print("="*60) 