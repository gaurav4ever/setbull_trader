"""
Technical Indicators Module for Morning Range Strategy.

This module provides configurable technical indicators using pandas-ta
with a focus on scalability and parameter customization.
"""

from typing import Dict, List, Optional, Union, Tuple
import pandas as pd
import pandas_ta as ta
import numpy as np
import logging
from datetime import datetime, time

logger = logging.getLogger(__name__)

class TechnicalIndicators:
    """Configurable technical indicators calculator."""
    
    def __init__(self, config: Optional[Dict] = None):
        """
        Initialize technical indicators calculator.
        
        Args:
            config: Dictionary containing indicator configurations
                   Example:
                   {
                       'ema': {
                           'periods': [5, 9, 50],
                           'column_prefix': 'EMA'
                       },
                       'rsi': {
                           'period': 14,
                           'column_prefix': 'RSI'
                       },
                       'atr': {
                           'period': 14,
                           'column_prefix': 'ATR'
                       }
                   }
        """
        self.config = config or {
            'ema': {
                'periods': [5, 9, 50],
                'column_prefix': 'EMA'
            },
            'rsi': {
                'period': 14,
                'column_prefix': 'RSI'
            },
            'atr': {
                'period': 14,
                'column_prefix': 'ATR'
            }
        }
        self.daily_indicators = {}  # Store daily indicators
        logger.info(f"Initialized TechnicalIndicators with config: {self.config}")

    def calculate_ema(self, df: pd.DataFrame, price_column: str = 'close', timeframe: str = 'intraday') -> pd.DataFrame:
        """
        Calculate EMA for configured periods.
        
        Args:
            df: DataFrame with price data
            price_column: Column name containing price data
            timeframe: 'intraday' or 'daily'
            
        Returns:
            DataFrame with EMA columns added
        """
        ema_config = self.config['ema']
        periods = ema_config['periods']
        prefix = f"{timeframe.upper()}_{ema_config['column_prefix']}"
        
        for period in periods:
            column_name = f"{prefix}_{period}"
            df[column_name] = ta.ema(df[price_column], length=period)
            
        logger.debug(f"Calculated {timeframe} EMAs for periods: {periods}")
        return df

    def calculate_rsi(self, df: pd.DataFrame, price_column: str = 'close', timeframe: str = 'intraday') -> pd.DataFrame:
        """
        Calculate RSI.
        
        Args:
            df: DataFrame with price data
            price_column: Column name containing price data
            timeframe: 'intraday' or 'daily'
            
        Returns:
            DataFrame with RSI column added
        """
        rsi_config = self.config['rsi']
        period = rsi_config['period']
        prefix = f"{timeframe.upper()}_{rsi_config['column_prefix']}"
        
        column_name = f"{prefix}_{period}"
        df[column_name] = ta.rsi(df[price_column], length=period)
        
        logger.debug(f"Calculated {timeframe} RSI for period: {period}")
        return df

    def calculate_atr(self, df: pd.DataFrame, timeframe: str = 'intraday') -> pd.DataFrame:
        """
        Calculate ATR.
        
        Args:
            df: DataFrame with OHLC data
            timeframe: 'intraday' or 'daily'
            
        Returns:
            DataFrame with ATR column added
        """
        atr_config = self.config['atr']
        period = atr_config['period']
        prefix = f"{timeframe.upper()}_{atr_config['column_prefix']}"
        
        column_name = f"{prefix}_{period}"
        df[column_name] = ta.atr(df['high'], df['low'], df['close'], length=period)
        
        logger.debug(f"Calculated {timeframe} ATR for period: {period}")
        return df

    def calculate_all(self, df: pd.DataFrame, timeframe: str = 'intraday') -> pd.DataFrame:
        """
        Calculate all configured indicators.
        
        Args:
            df: DataFrame with OHLC data
            timeframe: 'intraday' or 'daily'
            
        Returns:
            DataFrame with all indicators added
        """
        # Calculate EMAs
        df = self.calculate_ema(df, timeframe=timeframe)
        
        # Calculate RSI
        df = self.calculate_rsi(df, timeframe=timeframe)
        
        # Calculate ATR
        df = self.calculate_atr(df, timeframe=timeframe)
        
        logger.info(f"Calculated all configured {timeframe} technical indicators")
        return df

    def update_daily_indicators(self, daily_df: pd.DataFrame):
        """
        Calculate and store daily indicators.
        
        Args:
            daily_df: DataFrame with daily candle data
        """
        # Calculate daily indicators
        daily_df = self.calculate_all(daily_df, timeframe='daily')
        
        # Store the latest values
        self.daily_indicators = {
            name: daily_df[name].iloc[-1]
            for name in self.get_indicator_names(timeframe='daily')
        }
        
        logger.info(f"Updated daily indicators: {self.daily_indicators}")

    def apply_daily_indicators(self, df: pd.DataFrame) -> pd.DataFrame:
        """
        Apply stored daily indicators to intraday data.
        
        Args:
            df: DataFrame with intraday candle data
            
        Returns:
            DataFrame with daily indicators added
        """
        for name, value in self.daily_indicators.items():
            df[name] = value
            
        logger.debug("Applied daily indicators to intraday data")
        return df

    def get_indicator_names(self, timeframe: str = 'intraday') -> List[str]:
        """
        Get list of all indicator column names.
        
        Args:
            timeframe: 'intraday' or 'daily'
            
        Returns:
            List of indicator column names
        """
        names = []
        prefix = f"{timeframe.upper()}_"
        
        # Add EMA names
        ema_config = self.config['ema']
        for period in ema_config['periods']:
            names.append(f"{prefix}{ema_config['column_prefix']}_{period}")
            
        # Add RSI name
        rsi_config = self.config['rsi']
        names.append(f"{prefix}{rsi_config['column_prefix']}_{rsi_config['period']}")
        
        # Add ATR name
        atr_config = self.config['atr']
        names.append(f"{prefix}{atr_config['column_prefix']}_{atr_config['period']}")
        
        return names

    def update_config(self, new_config: Dict):
        """
        Update indicator configurations.
        
        Args:
            new_config: New configuration dictionary
        """
        self.config.update(new_config)
        logger.info(f"Updated technical indicators config: {self.config}") 