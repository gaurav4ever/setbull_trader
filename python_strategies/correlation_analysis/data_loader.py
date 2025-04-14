"""
Data loader for correlation analysis system.

This module handles loading and validating trade data from CSV files.
"""

import logging
import pandas as pd
from typing import Dict, List, Optional, Union, Any, Tuple
from datetime import datetime

logger = logging.getLogger(__name__)

class DataLoader:
    """Load and validate trade data from CSV files."""
    
    def __init__(self, file_path: str):
        """
        Initialize the data loader.
        
        Args:
            file_path: Path to the CSV file containing trade data
        """
        self.file_path = file_path
        
    def load_csv(self) -> pd.DataFrame:
        """
        Load trade data from CSV file.
        
        Returns:
            DataFrame containing trade data
            
        Raises:
            ValueError: If data format is invalid
        """
        try:
            # Load CSV file
            df = pd.read_csv(self.file_path)
            
            # Validate required columns
            required_columns = ['Date', 'Name', 'P&L', 'Status', 'Direction', 'Trade Type', 'Max R Multiple', 'Cumulative']
            missing_columns = [col for col in required_columns if col not in df.columns]
            if missing_columns:
                raise ValueError(f"Missing required columns: {missing_columns}")
            
            # Convert Date to datetime
            df['Date'] = pd.to_datetime(df['Date'])
            
            # Ensure numeric columns are the correct type
            numeric_cols = ['P&L', 'Max R Multiple', 'Cumulative']
            for col in numeric_cols:
                df[col] = pd.to_numeric(df[col], errors='coerce')
            
            # Sort by Date
            df = df.sort_values('Date')
            
            logger.info(f"Successfully loaded {len(df)} trades from {self.file_path}")
            return df
            
        except Exception as e:
            logger.error(f"Error loading CSV file: {str(e)}")
            raise ValueError(f"Failed to load CSV file: {str(e)}")
            
    def validate_data(self, df: pd.DataFrame) -> bool:
        """
        Validate the loaded trade data.
        
        Args:
            df: DataFrame containing trade data
            
        Returns:
            True if data is valid, False otherwise
        """
        try:
            # Check for missing values
            if df.isnull().any().any():
                logger.warning("Found missing values in the data")
                return False
                
            # Validate Status values
            valid_statuses = ['PROFIT', 'LOSS', 'FLAT']
            invalid_statuses = df[~df['Status'].isin(valid_statuses)]['Status'].unique()
            if len(invalid_statuses) > 0:
                logger.warning(f"Found invalid status values: {invalid_statuses}")
                return False
                
            # Validate Direction values
            valid_directions = ['LONG', 'SHORT']
            invalid_directions = df[~df['Direction'].isin(valid_directions)]['Direction'].unique()
            if len(invalid_directions) > 0:
                logger.warning(f"Found invalid direction values: {invalid_directions}")
                return False
                
            # Validate Trade Type values
            valid_trade_types = ['1ST_ENTRY']  # Add more as needed
            invalid_trade_types = df[~df['Trade Type'].isin(valid_trade_types)]['Trade Type'].unique()
            if len(invalid_trade_types) > 0:
                logger.warning(f"Found invalid trade type values: {invalid_trade_types}")
                return False
                
            logger.info("Data validation successful")
            return True
            
        except Exception as e:
            logger.error(f"Error validating data: {str(e)}")
            return False
            
    def clean_data(self, df: pd.DataFrame) -> pd.DataFrame:
        """
        Clean the trade data by handling missing values and outliers.
        
        Args:
            df: DataFrame containing trade data
            
        Returns:
            Cleaned DataFrame
        """
        try:
            # Make a copy to avoid modifying original data
            cleaned_df = df.copy()
            
            # Handle missing values in numeric columns
            numeric_cols = ['P&L', 'Max R Multiple', 'Cumulative']
            for col in numeric_cols:
                cleaned_df[col] = cleaned_df[col].fillna(0)
            
            # Handle missing values in categorical columns
            categorical_cols = ['Status', 'Direction', 'Trade Type']
            for col in categorical_cols:
                cleaned_df[col] = cleaned_df[col].fillna('UNKNOWN')
            
            # Remove duplicate trades
            cleaned_df = cleaned_df.drop_duplicates(subset=['Date', 'Name', 'Direction', 'Trade Type'])
            
            logger.info("Data cleaning completed")
            return cleaned_df
            
        except Exception as e:
            logger.error(f"Error cleaning data: {str(e)}")
            raise ValueError(f"Failed to clean data: {str(e)}") 