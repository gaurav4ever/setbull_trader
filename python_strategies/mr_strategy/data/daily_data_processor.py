import pandas as pd
import numpy as np
from typing import Dict, Optional
import logging

logger = logging.getLogger(__name__)

class DailyDataProcessor:
    """
    Processes daily data and adds derived columns based on various trading patterns and indicators.
    This class follows a modular approach where each pattern/indicator is processed separately.
    """
    
    def __init__(self, config: Optional[Dict] = None):
        self.config = config or {}
        self.processors = {
            'ema_indicators': self._process_ema_indicators,
            'rsi_indicators': self._process_rsi_indicators,
            'atr_indicators': self._process_atr_indicators,
            'player_type': self._process_player_type,
        }
    
    def process(self, daily_df: pd.DataFrame) -> pd.DataFrame:
        """
        Process daily data using all registered processors.
        
        Args:
            daily_df: DataFrame with daily candle data
            
        Returns:
            DataFrame with additional derived columns
        """
        processed_df = daily_df.copy()
        processed_df['timestamp'] = pd.to_datetime(processed_df['timestamp'])
        processed_df = processed_df.sort_values('timestamp')
        
        # Apply each processor in sequence
        for processor_name, processor_func in self.processors.items():
            try:
                processed_df = processor_func(processed_df)
            except Exception as e:
                logger.error(f"Error in processor {processor_name}: {str(e)}")
                continue
                
        return processed_df
    
    def _process_ema_indicators(self, df: pd.DataFrame) -> pd.DataFrame:
        """Calculate daily EMA indicators."""
        df['DAILY_EMA_5'] = df['close'].ewm(span=5, adjust=False).mean()
        df['DAILY_EMA_9'] = df['close'].ewm(span=9, adjust=False).mean()
        df['DAILY_EMA_50'] = df['close'].ewm(span=50, adjust=False).mean()
        return df
    
    def _process_rsi_indicators(self, df: pd.DataFrame) -> pd.DataFrame:
        """Calculate daily RSI indicators."""
        delta = df['close'].diff()
        gain = (delta.where(delta > 0, 0)).rolling(window=14).mean()
        loss = (-delta.where(delta < 0, 0)).rolling(window=14).mean()
        rs = gain / loss
        df['DAILY_RSI_14'] = 100 - (100 / (1 + rs))
        return df
    
    def _process_atr_indicators(self, df: pd.DataFrame) -> pd.DataFrame:
        """Calculate daily ATR indicators."""
        high_low = df['high'] - df['low']
        high_close = np.abs(df['high'] - df['close'].shift())
        low_close = np.abs(df['low'] - df['close'].shift())
        ranges = pd.concat([high_low, high_close, low_close], axis=1)
        true_range = np.max(ranges, axis=1)
        df['DAILY_ATR_14'] = true_range.rolling(14).mean()
        return df
    
    def _process_player_type(self, df: pd.DataFrame) -> pd.DataFrame:
        """
        Process player type classification and related indicators.
        """
        # Add previous day's data
        df['prev_day_open'] = df['open'].shift(1)
        df['prev_day_high'] = df['high'].shift(1)
        df['prev_day_low'] = df['low'].shift(1)
        df['prev_day_close'] = df['close'].shift(1)
        df['prev_day_mid'] = (df['prev_day_high'] + df['prev_day_low']) / 2
        
        # Calculate boolean flags
        df['prev_day_high_range'] = df['prev_day_high'] - df['prev_day_low']
        buffer = 0.03 * df['prev_day_high_range'] # 3% buffer
        df['OAH'] = df['open'] > (df['prev_day_mid'] + buffer)  
        df['OAL'] = df['open'] < (df['prev_day_mid'] - buffer)
        df['OAM'] = (df['open'] < df['prev_day_mid'] + buffer) & (df['open'] > df['prev_day_mid'] - buffer)
        
        # Calculate player type
        def _determine_player_type(row):
            if pd.isna(row['prev_day_high']) or pd.isna(row['prev_day_low']):
                return None
            
            prev_day_range = row['prev_day_high'] - row['prev_day_low']
            
            if (row['high'] > row['prev_day_high'] + 0.5 * prev_day_range or 
                row['low'] < row['prev_day_low'] - 0.5 * prev_day_range):
                return 'RUNNER'
            
            if (row['high'] <= row['prev_day_high'] and 
                row['low'] >= row['prev_day_low']):
                return 'COILER'
            
            if (row['high'] > row['prev_day_high'] or 
                row['low'] < row['prev_day_low']):
                return 'SPRINGBOARDER'
            
            return 'COILER'
        
        # df['player_type'] = df.apply(_determine_player_type, axis=1)
        return df
