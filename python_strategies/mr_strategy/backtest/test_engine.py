"""
Test suite for the backtesting engine.

This module provides comprehensive test cases for the backtesting engine,
focusing on multi-instrument support and portfolio-level analysis.
"""

import pytest
import pandas as pd
import numpy as np
from datetime import datetime, timedelta
from typing import Dict, List

from ..backtest.engine import BacktestEngine, BacktestConfig
from ..strategy.config import MRStrategyConfig
from ..strategy.models import SignalType, TradeType, TradeStatus

# Configure logging
import logging
logger = logging.getLogger(__name__)
logger.setLevel(logging.DEBUG)

def create_sample_candle_data(
    start_date: datetime,
    end_date: datetime,
    instrument_key: str,
    base_price: float = 100.0,
    volatility: float = 0.01
) -> pd.DataFrame:
    """Create sample candle data for testing."""
    dates = pd.date_range(start=start_date, end=end_date, freq='1min')
    n = len(dates)
    
    # Generate random returns
    returns = np.random.normal(0, volatility, n)
    prices = base_price * (1 + returns).cumprod()
    
    # Create OHLCV data
    data = {
        'timestamp': dates,
        'open': prices,
        'high': prices * (1 + np.random.uniform(0, 0.01, n)),
        'low': prices * (1 - np.random.uniform(0, 0.01, n)),
        'close': prices * (1 + np.random.normal(0, 0.005, n)),
        'volume': np.random.randint(1000, 10000, n)
    }
    
    df = pd.DataFrame(data)
    df['instrument_key'] = instrument_key
    return df

@pytest.fixture
def sample_config() -> BacktestConfig:
    """Create a sample backtest configuration."""
    return BacktestConfig(
        start_date=datetime(2024, 1, 1),
        end_date=datetime(2024, 1, 31),
        instruments=[
            {'key': 'AAPL', 'direction': 'BULLISH'},
            {'key': 'MSFT', 'direction': 'BEARISH'}
        ],
        strategies=[
            MRStrategyConfig(
                instrument_key={'key': 'AAPL'},
                range_type='FIXED',
                entry_type='IMMEDIATE_BREAKOUT',
                sl_percentage=0.5
            ),
            MRStrategyConfig(
                instrument_key={'key': 'MSFT'},
                range_type='FIXED',
                entry_type='IMMEDIATE_BREAKOUT',
                sl_percentage=0.5
            )
        ],
        initial_capital=100000.0,
        position_size_type='FIXED',
        max_positions=2
    )

@pytest.fixture
def sample_data(sample_config: BacktestConfig) -> Dict[str, pd.DataFrame]:
    """Create sample data for multiple instruments."""
    data = {}
    for instrument in sample_config.instruments:
        data[instrument['key']] = create_sample_candle_data(
            start_date=sample_config.start_date,
            end_date=sample_config.end_date,
            instrument_key=instrument['key']
        )
    return data

@pytest.mark.asyncio
async def test_multi_instrument_backtest(sample_config: BacktestConfig, sample_data: Dict[str, pd.DataFrame]):
    """Test backtesting with multiple instruments."""
    engine = BacktestEngine(sample_config)
    results = await engine.run_backtest(sample_data)
    
    # Verify results structure
    assert 'signals' in results
    assert 'trades' in results
    assert 'metrics' in results
    assert 'instruments' in results
    assert 'portfolio' in results
    
    # Verify instrument-specific results
    for instrument in sample_config.instruments:
        assert instrument['key'] in results['instruments']
        instrument_results = results['instruments'][instrument['key']]
        
        assert 'direction' in instrument_results
        assert instrument_results['direction'] == instrument['direction']
        assert 'signals' in instrument_results
        assert 'trades' in instrument_results
        assert 'metrics' in instrument_results
        assert 'equity_curve' in instrument_results
        
        # Verify signal filtering by direction
        for signal in instrument_results['signals']:
            if instrument['direction'] == 'BULLISH':
                assert signal.direction.value == 'LONG'
            else:
                assert signal.direction.value == 'SHORT'
    
    # Verify portfolio-level results
    portfolio = results['portfolio']
    assert 'equity_curve' in portfolio
    assert 'metrics' in portfolio
    
    # Verify portfolio metrics
    portfolio_metrics = portfolio['metrics']
    assert 'total_trades' in portfolio_metrics
    assert 'win_rate' in portfolio_metrics
    assert 'profit_factor' in portfolio_metrics
    assert 'total_return' in portfolio_metrics
    assert 'max_drawdown' in portfolio_metrics
    assert 'sharpe_ratio' in portfolio_metrics

@pytest.mark.asyncio
async def test_portfolio_equity_curve(sample_config: BacktestConfig, sample_data: Dict[str, pd.DataFrame]):
    """Test portfolio equity curve calculation."""
    engine = BacktestEngine(sample_config)
    results = await engine.run_backtest(sample_data)
    
    equity_curve = results['portfolio']['equity_curve']
    
    # Verify equity curve properties
    assert not equity_curve.empty
    assert equity_curve.index.is_monotonic_increasing
    assert equity_curve.iloc[0] == sample_config.initial_capital
    assert equity_curve.iloc[-1] >= sample_config.initial_capital - (sample_config.initial_capital * 0.5)  # Max drawdown check

@pytest.mark.asyncio
async def test_metrics_calculation(sample_config: BacktestConfig, sample_data: Dict[str, pd.DataFrame]):
    """Test metrics calculation for both instrument and portfolio levels."""
    engine = BacktestEngine(sample_config)
    results = await engine.run_backtest(sample_data)
    
    # Verify instrument-level metrics
    for instrument_key, instrument_results in results['instruments'].items():
        metrics = instrument_results['metrics']
        
        assert 'total_trades' in metrics
        assert 'win_rate' in metrics
        assert 'profit_factor' in metrics
        assert 'total_return' in metrics
        assert 'max_drawdown' in metrics
        assert 'sharpe_ratio' in metrics
        
        # Verify metrics are within reasonable ranges
        assert 0 <= metrics['win_rate'] <= 1
        assert metrics['profit_factor'] >= 0
        assert metrics['max_drawdown'] >= 0
        assert metrics['max_drawdown'] <= 1
    
    # Verify portfolio-level metrics
    portfolio_metrics = results['portfolio']['metrics']
    
    assert 'total_trades' in portfolio_metrics
    assert 'win_rate' in portfolio_metrics
    assert 'profit_factor' in portfolio_metrics
    assert 'total_return' in portfolio_metrics
    assert 'max_drawdown' in portfolio_metrics
    assert 'sharpe_ratio' in portfolio_metrics
    
    # Verify portfolio metrics are consistent with instrument metrics
    total_trades = sum(
        results['instruments'][key]['metrics']['total_trades']
        for key in results['instruments']
    )
    assert portfolio_metrics['total_trades'] == total_trades

@pytest.mark.asyncio
async def test_error_handling(sample_config: BacktestConfig):
    """Test error handling for invalid inputs."""
    engine = BacktestEngine(sample_config)
    
    # Test with empty data
    with pytest.raises(ValueError):
        await engine.run_backtest({})
    
    # Test with invalid instrument key
    invalid_data = {'INVALID': create_sample_candle_data(
        start_date=sample_config.start_date,
        end_date=sample_config.end_date,
        instrument_key='INVALID'
    )}
    with pytest.raises(ValueError):
        await engine.run_backtest(invalid_data)
    
    # Test with missing required columns
    invalid_df = pd.DataFrame({
        'timestamp': pd.date_range(start=sample_config.start_date, end=sample_config.end_date, freq='1min'),
        'open': [100] * 100,
        'close': [101] * 100
    })
    with pytest.raises(ValueError):
        await engine.run_backtest({'AAPL': invalid_df})

@pytest.mark.asyncio
async def test_performance_metrics(sample_config: BacktestConfig, sample_data: Dict[str, pd.DataFrame]):
    """Test performance metrics calculation."""
    engine = BacktestEngine(sample_config)
    results = await engine.run_backtest(sample_data)
    
    # Verify Sharpe ratio calculation
    portfolio_metrics = results['portfolio']['metrics']
    assert 'sharpe_ratio' in portfolio_metrics
    assert isinstance(portfolio_metrics['sharpe_ratio'], float)
    
    # Verify max drawdown calculation
    equity_curve = results['portfolio']['equity_curve']
    rolling_max = equity_curve.expanding().max()
    drawdowns = (equity_curve - rolling_max) / rolling_max
    calculated_max_drawdown = abs(drawdowns.min())
    assert abs(portfolio_metrics['max_drawdown'] - calculated_max_drawdown) < 0.0001
    
    # Verify win rate calculation
    all_trades = results['trades']
    winning_trades = [t for t in all_trades if t['realized_pnl'] > 0]
    calculated_win_rate = len(winning_trades) / len(all_trades) if all_trades else 0
    assert abs(portfolio_metrics['win_rate'] - calculated_win_rate) < 0.0001

@pytest.mark.asyncio
async def test_trade_management(sample_config: BacktestConfig, sample_data: Dict[str, pd.DataFrame]):
    """Test trade management functionality."""
    engine = BacktestEngine(sample_config)
    results = await engine.run_backtest(sample_data)
    
    # Verify trade creation and management
    for instrument_key, instrument_results in results['instruments'].items():
        trades = instrument_results['trades']
        
        # Verify trade structure
        for trade in trades:
            assert 'instrument_key' in trade
            assert 'entry_price' in trade
            assert 'exit_price' in trade
            assert 'position_size' in trade
            assert 'realized_pnl' in trade
            assert 'status' in trade
            
            # Verify trade status
            assert trade['status'] in [
                TradeStatus.CLOSED.value,
                TradeStatus.STOPPED_OUT.value,
                TradeStatus.TAKE_PROFIT.value
            ]
            
            # Verify position size calculation
            assert trade['position_size'] > 0
            assert trade['position_size'] <= sample_config.initial_capital * 0.5  # Max position size check
            
            # Verify P&L calculation
            if trade['status'] == TradeStatus.STOPPED_OUT.value:
                assert trade['realized_pnl'] < 0
            elif trade['status'] == TradeStatus.TAKE_PROFIT.value:
                assert trade['realized_pnl'] > 0 