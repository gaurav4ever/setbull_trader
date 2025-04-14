"""
Correlation Analyzer for the correlation analysis system.

This module handles the calculation of correlations between stocks based on their trading data.
"""

import pandas as pd
import numpy as np
import logging

logger = logging.getLogger(__name__)

class CorrelationAnalyzer:
    """Analyze correlations between stocks based on trading data."""
    
    def __init__(self, trade_data: pd.DataFrame):
        """
        Initialize the correlation analyzer.
        
        Args:
            trade_data: DataFrame containing trade data with P&L values
        """
        self.trade_data = trade_data
        
    def calculate_spearman_correlation(self) -> pd.DataFrame:
        """Calculate Spearman rank correlation between stocks."""
        logger.info("Calculating Spearman correlation...")
        correlation_matrix = self.trade_data.corr(method='spearman')
        return correlation_matrix
    
    def calculate_binary_correlation(self) -> pd.DataFrame:
        """Calculate binary win/loss correlation between stocks."""
        logger.info("Calculating binary win/loss correlation...")
        # Create a binary win/loss DataFrame
        binary_df = self.trade_data.copy()
        binary_df = binary_df.applymap(lambda x: 1 if x > 0 else 0)
        binary_correlation_matrix = binary_df.corr(method='pearson')
        return binary_correlation_matrix
    
    def calculate_r_multiple_correlation(self) -> pd.DataFrame:
        """Calculate R-multiple correlation between stocks."""
        logger.info("Calculating R-multiple correlation...")
        # Calculate R-multiple (P&L / Initial Capital)
        r_multiple_df = self.trade_data.copy()
        r_multiple_df = r_multiple_df.applymap(lambda x: x / 100000.0)  # Using initial capital of 100,000
        r_multiple_correlation_matrix = r_multiple_df.corr(method='pearson')
        return r_multiple_correlation_matrix
    
    def get_significant_correlations(self, corr_matrix: pd.DataFrame, threshold: float) -> pd.DataFrame:
        """Get significant correlations above a certain threshold."""
        logger.info(f"Filtering correlations with threshold: {threshold}")
        # Create a copy to avoid modifying the original
        significant_corr = corr_matrix.copy()
        # Set diagonal and below-threshold values to NaN
        np.fill_diagonal(significant_corr.values, np.nan)
        significant_corr[significant_corr.abs() <= threshold] = np.nan
        # Drop rows and columns with all NaN values
        significant_corr = significant_corr.dropna(how='all', axis=0).dropna(how='all', axis=1)
        return significant_corr
