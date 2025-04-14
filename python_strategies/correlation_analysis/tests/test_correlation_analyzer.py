"""
Tests for the CorrelationAnalyzer class.
"""

import pytest
import pandas as pd
from correlation_analyzer import CorrelationAnalyzer

@pytest.fixture
def sample_trade_data():
    """Create sample trade data for testing."""
    return pd.DataFrame({
        'Date': ['2025-04-01', '2025-04-02', '2025-04-02'],
        'Name': ['NACLIND', 'FUSION', 'OLAELEC'],
        'P&L': [-49.92, -49.70, -49.92],
        'Status': ['LOSS', 'LOSS', 'LOSS'],
        'Direction': ['LONG', 'SHORT', 'SHORT'],
        'Trade Type': ['1ST_ENTRY', '1ST_ENTRY', '1ST_ENTRY'],
        'Max R Multiple': [1.00, -1.00, -1.00],
        'Cumulative': [-100.17, -249.49, -13.92]
    })

def test_spearman_correlation(sample_trade_data):
    """Test Spearman correlation calculation."""
    analyzer = CorrelationAnalyzer(sample_trade_data)
    correlation_matrix = analyzer.calculate_spearman_correlation()
    assert correlation_matrix is not None
    assert 'P&L' in correlation_matrix.columns
    assert 'Max R Multiple' in correlation_matrix.columns

def test_binary_correlation(sample_trade_data):
    """Test binary win/loss correlation calculation."""
    analyzer = CorrelationAnalyzer(sample_trade_data)
    binary_correlation_matrix = analyzer.calculate_binary_correlation()
    assert binary_correlation_matrix is not None
    assert 'Win' in binary_correlation_matrix.columns

def test_r_multiple_correlation(sample_trade_data):
    """Test R-multiple correlation calculation."""
    analyzer = CorrelationAnalyzer(sample_trade_data)
    r_multiple_correlation_matrix = analyzer.calculate_r_multiple_correlation()
    assert r_multiple_correlation_matrix is not None
    assert 'Max R Multiple' in r_multiple_correlation_matrix.columns

def test_significant_correlations(sample_trade_data):
    """Test extraction of significant correlations."""
    analyzer = CorrelationAnalyzer(sample_trade_data)
    correlation_matrix = analyzer.calculate_spearman_correlation()
    significant_corr = analyzer.get_significant_correlations(correlation_matrix, threshold=0.5)
    assert significant_corr is not None
    assert not significant_corr.empty

    
    # Calculate correlation
    corr_matrix = correlation_analyzer.calculate_binary_correlation()
    
    # Check if correlation matrix is correct shape
    assert corr_matrix.shape == (len(pivot_df.columns) + 1, len(pivot_df.columns) + 1)  # +1 for 'Win' column
    
    # Check if diagonal values are 1.0
    assert all(np.diag(corr_matrix) == 1.0)
    
    # Check if correlation values are between -1 and 1
    assert (corr_matrix >= -1.0).all().all()
    assert (corr_matrix <= 1.0).all().all()

def test_calculate_r_multiple_correlation(correlation_analyzer):
    """Test R-multiple correlation calculation."""
    # Calculate correlation
    corr_matrix = correlation_analyzer.calculate_r_multiple_correlation()
    
    # Check if correlation matrix is correct shape
    assert corr_matrix.shape == (1, 1)  # Only one column 'Max R Multiple'
    
    # Check if correlation value is 1.0 (self-correlation)
    assert corr_matrix.iloc[0, 0] == 1.0

def test_get_significant_correlations(correlation_analyzer):
    """Test filtering of significant correlations."""
    # Calculate correlation matrix
    corr_matrix = correlation_analyzer.calculate_spearman_correlation()
    
    # Get significant correlations
    threshold = 0.5
    significant_corr = correlation_analyzer.get_significant_correlations(corr_matrix, threshold)
    
    # Check if all correlations are above threshold
    assert (significant_corr.abs() > threshold).all().all()
    
    # Check if diagonal values are removed
    assert not any(significant_corr.index == significant_corr.columns)

def test_empty_data():
    """Test handling of empty data."""
    empty_df = pd.DataFrame(columns=['Date', 'Name', 'P&L', 'Status', 'Direction', 'Trade Type', 'Max R Multiple', 'Cumulative'])
    analyzer = CorrelationAnalyzer(empty_df)
    
    # Test Spearman correlation
    corr_matrix = analyzer.calculate_spearman_correlation()
    assert corr_matrix.empty
    
    # Test binary correlation
    corr_matrix = analyzer.calculate_binary_correlation()
    assert corr_matrix.empty
    
    # Test R-multiple correlation
    corr_matrix = analyzer.calculate_r_multiple_correlation()
    assert corr_matrix.empty