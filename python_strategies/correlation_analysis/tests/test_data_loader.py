"""
Tests for the DataLoader class.
"""

import pytest
import pandas as pd
from datetime import datetime
from ..data_loader import DataLoader
import os

@pytest.fixture
def sample_data():
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

@pytest.fixture
def data_loader(tmp_path, sample_data):
    """Create a DataLoader instance with sample data."""
    # Create a temporary CSV file
    file_path = tmp_path / "test_trades.csv"
    sample_data.to_csv(file_path, index=False)
    
    return DataLoader(str(file_path))

def test_load_csv(data_loader, sample_data):
    """Test loading CSV file."""
    df = data_loader.load_csv()
    
    # Check if data was loaded correctly
    assert len(df) == len(sample_data)
    assert all(col in df.columns for col in sample_data.columns)
    assert df['Date'].dtype == 'datetime64[ns]'
    assert df['P&L'].dtype == 'float64'
    assert df['Max R Multiple'].dtype == 'float64'
    assert df['Cumulative'].dtype == 'float64'

def test_validate_data(data_loader, sample_data):
    """Test data validation."""
    # Test with valid data
    assert data_loader.validate_data(sample_data) is True
    
    # Test with invalid status
    invalid_data = sample_data.copy()
    invalid_data.loc[0, 'Status'] = 'INVALID'
    assert data_loader.validate_data(invalid_data) is False
    
    # Test with invalid direction
    invalid_data = sample_data.copy()
    invalid_data.loc[0, 'Direction'] = 'INVALID'
    assert data_loader.validate_data(invalid_data) is False
    
    # Test with invalid trade type
    invalid_data = sample_data.copy()
    invalid_data.loc[0, 'Trade Type'] = 'INVALID'
    assert data_loader.validate_data(invalid_data) is False

def test_clean_data(data_loader, sample_data):
    """Test data cleaning."""
    # Add some missing values and duplicates
    dirty_data = sample_data.copy()
    dirty_data.loc[0, 'P&L'] = None
    dirty_data.loc[1, 'Status'] = None
    dirty_data = pd.concat([dirty_data, dirty_data.iloc[[0]]])  # Add duplicate
    
    # Clean the data
    cleaned_data = data_loader.clean_data(dirty_data)
    
    # Check if cleaning worked
    assert not cleaned_data.isnull().any().any()
    assert len(cleaned_data) == len(sample_data)  # Duplicates should be removed
    assert cleaned_data['P&L'].iloc[0] == 0  # Missing value should be filled with 0
    assert cleaned_data['Status'].iloc[1] == 'UNKNOWN'  # Missing value should be filled with 'UNKNOWN'

def test_invalid_file_path():
    """Test handling of invalid file path."""
    with pytest.raises(ValueError):
        DataLoader("nonexistent_file.csv").load_csv()

def test_missing_columns(tmp_path):
    """Test handling of missing required columns."""
    # Create CSV with missing columns
    file_path = tmp_path / "invalid_trades.csv"
    pd.DataFrame({'Date': ['2025-04-01'], 'Name': ['TEST']}).to_csv(file_path, index=False)
    
    with pytest.raises(ValueError):
        DataLoader(str(file_path)).load_csv() 