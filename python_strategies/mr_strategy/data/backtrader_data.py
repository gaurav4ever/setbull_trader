"""
Backtrader data feed implementation for Morning Range strategy.

This module provides custom data feed classes for Backtrader integration,
handling both intraday and daily data with technical indicators.
"""

import backtrader as bt
import pandas as pd
import numpy as np
from typing import Optional, Dict, Any
import logging

logger = logging.getLogger(__name__)

class MorningRangeDataFeed(bt.feeds.PandasData):
    """
    Custom data feed for Morning Range strategy that handles both intraday and daily data.
    """
    
    # Define the lines that will be available in the data feed
    lines = (
        'INTRADAY_EMA_5', 'INTRADAY_EMA_9', 'INTRADAY_EMA_50',  # Intraday EMAs
        'INTRADAY_RSI_14', 'INTRADAY_ATR_14',  # Intraday indicators
        'DAILY_EMA_5', 'DAILY_EMA_9', 'DAILY_EMA_50',  # Daily EMAs
        'DAILY_RSI_14', 'DAILY_ATR_14'  # Daily indicators
    )
    
    # Define the parameters for the data feed
    params = (
        ('datetime', None),
        ('open', 1),      # Column index for open price
        ('high', 2),      # Column index for high price
        ('low', 3),       # Column index for low price
        ('close', 4),     # Column index for close price
        ('volume', 5),    # Column index for volume
        ('openinterest', -1),  # Column index for open interest (not used)
        # Custom indicators (must match your DataFrame column names)
        ('INTRADAY_EMA_5', -1),
        ('INTRADAY_EMA_9', -1),
        ('INTRADAY_EMA_50', -1),
        ('INTRADAY_RSI_14', -1),
        ('INTRADAY_ATR_14', -1),
        ('DAILY_EMA_5', -1),
        ('DAILY_EMA_9', -1),
        ('DAILY_EMA_50', -1),
        ('DAILY_RSI_14', -1),
        ('DAILY_ATR_14', -1),
         # Optional: timeframe and compression
        ('timeframe', bt.TimeFrame.Minutes),  # Default timeframe
        ('compression', 5),  # Default compression (5 minutes)
    )

    def __init__(self, *args, **kwargs):
        """
        Initialize the data feed.
        
        Args:
            *args: Positional arguments for PandasData
            **kwargs: Keyword arguments for PandasData
        """
        super().__init__(*args, **kwargs)
        logger.info("Initialized MorningRangeDataFeed")

    def _load(self):
        """
        Load the data into the feed.
        This method is called by Backtrader to load the data.
        """
        try:
            # Call parent's _load method to load basic OHLCV data
            ret = super()._load()
            
            if ret:
                # Load intraday indicators
                self.lines.INTRADAY_EMA_5[0] = self._dataname['INTRADAY_EMA_5'].iloc[self._idx]
                self.lines.INTRADAY_EMA_9[0] = self._dataname['INTRADAY_EMA_9'].iloc[self._idx]
                self.lines.INTRADAY_EMA_50[0] = self._dataname['INTRADAY_EMA_50'].iloc[self._idx]
                self.lines.INTRADAY_RSI_14[0] = self._dataname['INTRADAY_RSI_14'].iloc[self._idx]
                self.lines.INTRADAY_ATR_14[0] = self._dataname['INTRADAY_ATR_14'].iloc[self._idx]
                
                # Load daily indicators
                self.lines.DAILY_EMA_5[0] = self._dataname['DAILY_EMA_5'].iloc[self._idx]
                self.lines.DAILY_EMA_9[0] = self._dataname['DAILY_EMA_9'].iloc[self._idx]
                self.lines.DAILY_EMA_50[0] = self._dataname['DAILY_EMA_50'].iloc[self._idx]
                self.lines.DAILY_RSI_14[0] = self._dataname['DAILY_RSI_14'].iloc[self._idx]
                self.lines.DAILY_ATR_14[0] = self._dataname['DAILY_ATR_14'].iloc[self._idx]
            
            return ret
            
        except Exception as e:
            logger.error(f"Error loading data: {str(e)}")
            return False

    @classmethod
    def from_dataframe(cls, df: pd.DataFrame, timeframe: bt.TimeFrame = bt.TimeFrame.Minutes,
                      compression: int = 5) -> 'MorningRangeDataFeed':
        """
        Create a data feed from a pandas DataFrame.
        
        Args:
            df: DataFrame containing the data
            timeframe: Timeframe for the data
            compression: Compression factor for the timeframe
            
        Returns:
            MorningRangeDataFeed instance
        """
        try:
            # Ensure required columns exist
            required_columns = [
                'timestamp', 'open', 'high', 'low', 'close', 'volume',
                'INTRADAY_EMA_5', 'INTRADAY_EMA_9', 'INTRADAY_EMA_50',
                'INTRADAY_RSI_14', 'INTRADAY_ATR_14',
                'DAILY_EMA_5', 'DAILY_EMA_9', 'DAILY_EMA_50',
                'DAILY_RSI_14', 'DAILY_ATR_14'
            ]
            
            # Check for missing columns
            missing_columns = [col for col in required_columns if col not in df.columns]
            if missing_columns:
                raise ValueError(f"Missing required columns: {missing_columns}")
            
            # Ensure timestamp is datetime
            df['timestamp_v2'] = pd.to_datetime(df['timestamp_v2'], errors='coerce')
            df = df[df['timestamp_v2'].notnull()]  # Drop bad datetime rows
            df.set_index('timestamp_v2', inplace=True)
            df.index = pd.DatetimeIndex(df.index)
            
            # Create data feed
            data = cls(
                dataname=df,
                timeframe=timeframe,
                compression=compression
            )
            
            logger.info(f"Created data feed with {len(df)} rows")
            return data
            
        except Exception as e:
            logger.error(f"Error creating data feed: {str(e)}")
            raise 