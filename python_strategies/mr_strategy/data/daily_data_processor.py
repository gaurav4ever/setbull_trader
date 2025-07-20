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
            'bb_indicators': self._process_bb_indicators,
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
        df['DAILY_EMA_5'] = df['ema_5']
        df['DAILY_EMA_9'] = df['ema_9']
        df['DAILY_EMA_50'] = df['ema_50']
        return df
    
    def _process_rsi_indicators(self, df: pd.DataFrame) -> pd.DataFrame:
        """Calculate daily RSI indicators."""
        df['DAILY_RSI_14'] = df['rsi']
        return df
    
    def _process_atr_indicators(self, df: pd.DataFrame) -> pd.DataFrame:
        """Calculate daily ATR indicators."""
        df['DAILY_ATR_14'] = df['atr']
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

    def _process_bb_indicators(self, df: pd.DataFrame) -> pd.DataFrame:
        """Calculate daily BB indicators."""
        df['DAILY_BB_UPPER'] = df['bb_upper']
        df['DAILY_BB_LOWER'] = df['bb_lower']
        df['DAILY_BB_MIDDLE'] = df['bb_middle']
        df['DAILY_BB_WIDTH'] = df['bb_width']
        df['DAILY_LOWEST_BB_WIDTH'] = df['lowest_bb_width']
        return df