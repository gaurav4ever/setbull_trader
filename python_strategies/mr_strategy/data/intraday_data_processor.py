import pandas as pd
import numpy as np
from typing import Dict, Optional
import logging

logger = logging.getLogger(__name__)

class IntradayDataProcessor:
    """
    Processes intraday data and adds derived columns based on various indicators.
    This class follows a modular approach where each indicator is processed separately.
    """
    
    def __init__(self, config: Optional[Dict] = None):
        self.config = config or {}
        self.processors = {
            'ema_indicators': self._process_ema_indicators,
            'rsi_indicators': self._process_rsi_indicators,
            'atr_indicators': self._process_atr_indicators,
            'bb_indicators': self._process_bb_indicators,
        }
    
    def process(self, intraday_df: pd.DataFrame) -> pd.DataFrame:
        """
        Process intraday data using all registered processors.
        
        Args:
            intraday_df: DataFrame with intraday candle data
            
        Returns:
            DataFrame with additional derived columns
        """
        processed_df = intraday_df.copy()
        
        # Apply each processor in sequence
        for processor_name, processor_func in self.processors.items():
            try:
                processed_df = processor_func(processed_df)
            except Exception as e:
                logger.error(f"Error in processor {processor_name}: {str(e)}")
                continue
                
        return processed_df
    
    def _process_ema_indicators(self, df: pd.DataFrame) -> pd.DataFrame:
        # """Calculate EMA indicators."""
        # df['INTRADAY_EMA_5'] = df['close'].ewm(span=5, adjust=False).mean()
        # df['INTRADAY_EMA_9'] = df['close'].ewm(span=9, adjust=False).mean()
        # df['INTRADAY_EMA_50'] = df['close'].ewm(span=50, adjust=False).mean()
        return df
    
    def _process_rsi_indicators(self, df: pd.DataFrame) -> pd.DataFrame:
        # """Calculate RSI indicators."""
        # delta = df['close'].diff()
        # gain = (delta.where(delta > 0, 0)).rolling(window=14).mean()
        # loss = (-delta.where(delta < 0, 0)).rolling(window=14).mean()
        # rs = gain / loss
        # df['INTRADAY_RSI_14'] = 100 - (100 / (1 + rs))
        return df
    
    def _process_atr_indicators(self, df: pd.DataFrame) -> pd.DataFrame:
        # """Calculate ATR indicators."""
        # high_low = df['high'] - df['low']
        # high_close = np.abs(df['high'] - df['close'].shift())
        # low_close = np.abs(df['low'] - df['close'].shift())
        # ranges = pd.concat([high_low, high_close, low_close], axis=1)
        # true_range = np.max(ranges, axis=1)
        # df['INTRADAY_ATR_14'] = true_range.rolling(14).mean()
        return df
    
    def _process_bb_indicators(self, df: pd.DataFrame) -> pd.DataFrame:
        """Calculate Bollinger Bands indicators."""
        # Get BB parameters from config or use defaults
        # bb_period = self.config.get('bb_period', 20)
        # bb_std_dev = self.config.get('bb_std_dev', 2.0)
        
        # # Calculate BB middle (SMA)
        # df['bb_middle'] = df['close'].rolling(window=bb_period).mean()
        
        # # Calculate BB standard deviation
        # bb_std = df['close'].rolling(window=bb_period).std()
        
        # # Calculate BB upper and lower bands
        # df['bb_upper'] = df['bb_middle'] + (bb_std * bb_std_dev)
        # df['bb_lower'] = df['bb_middle'] - (bb_std * bb_std_dev)
        
        # # Calculate BB width (percentage)
        # df['bb_width'] = (df['bb_upper'] - df['bb_lower']) / df['bb_middle']
        
        return df 