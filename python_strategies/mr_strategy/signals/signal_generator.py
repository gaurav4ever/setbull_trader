"""
Signal generator for the Morning Range strategy.

This module generates trading signals based on morning range breakouts and technical indicators,
integrated with Backtrader's signal system.
"""

import backtrader as bt
import pandas as pd
import numpy as np
from typing import Dict, Optional, Any, List
import logging
from datetime import datetime, time, timedelta
from enum import Enum

logger = logging.getLogger(__name__)

class SignalType(Enum):
    """Types of trading signals."""
    LONG = 1
    SHORT = -1
    EXIT = 0

class SignalGenerator(bt.Indicator):
    """
    Backtrader indicator that generates trading signals for the Morning Range strategy.
    
    This indicator:
    1. Tracks morning range (9:15-9:20)
    2. Monitors price action relative to the range
    3. Generates signals based on breakouts and technical indicators
    """
    
    lines = ('signal',)  # Signal line for Backtrader
    params = (
        ('mr_start_time', time(9, 15)),  # Morning range start time
        ('mr_end_time', time(9, 20)),    # Morning range end time
        ('market_close_time', time(15, 20)),  # Market close time
        ('rsi_period', 14),              # RSI period
        ('rsi_overbought', 70),          # RSI overbought level
        ('rsi_oversold', 30),            # RSI oversold level
        ('ema_period', 50),              # EMA period
        ('volume_ma_period', 20),        # Volume MA period
    )
    
    def __init__(self):
        """Initialize the signal generator."""
        # Add required indicators
        self.rsi = bt.indicators.RSI(period=self.p.rsi_period)
        self.ema = bt.indicators.EMA(period=self.p.ema_period)
        self.volume_ma = bt.indicators.SMA(self.data.volume, period=self.p.volume_ma_period)
        
        # Morning range tracking
        self.mr_high = None
        self.mr_low = None
        self.mr_established = False
        
        logger.info("Initialized SignalGenerator")
        
    def next(self):
        """Calculate the next signal value."""
        # Initialize signal to EXIT
        self.lines.signal[0] = SignalType.EXIT.value
        
        # Skip if we don't have enough data
        if len(self) < max(self.p.ema_period, self.p.rsi_period, self.p.volume_ma_period):
            return
            
        # Get current time
        current_time = self.data.datetime.time()
        
        # Morning range establishment
        if not self.mr_established and self.p.mr_start_time <= current_time <= self.p.mr_end_time:
            self._update_morning_range()
            
        # Generate signals after morning range is established
        if self.mr_established:
            self._generate_signals()
            
    def _update_morning_range(self):
        """Update morning range values."""
        # Update high and low during morning range period
        if self.mr_high is None or self.data.high[0] > self.mr_high:
            self.mr_high = self.data.high[0]
        if self.mr_low is None or self.data.low[0] < self.mr_low:
            self.mr_low = self.data.low[0]
            
        # Calculate range size at the end of morning range period
        if self.data.datetime.time() == self.p.mr_end_time:
            self.mr_established = True
            logger.info(f"Morning Range Established - High: {self.mr_high:.2f}, Low: {self.mr_low:.2f}")
            
    def _generate_signals(self):
        """Generate trading signals based on morning range and indicators."""
        # Check for breakout signals
        if self.data.close[0] > self.mr_high:
            # Bullish breakout
            if self._validate_bullish_signal():
                self.lines.signal[0] = SignalType.LONG.value
                
        elif self.data.close[0] < self.mr_low:
            # Bearish breakout
            if self._validate_bearish_signal():
                self.lines.signal[0] = SignalType.SHORT.value
                
        # Check for market close
        if self.data.datetime.time() >= self.p.market_close_time:
            self.lines.signal[0] = SignalType.EXIT.value
            
    def _validate_bullish_signal(self) -> bool:
        """Validate bullish breakout signal."""
        # Check RSI is not overbought
        if self.rsi[0] > self.p.rsi_overbought:
            return False
            
        # Check price is above EMA
        if self.data.close[0] < self.ema[0]:
            return False
            
        # Check volume confirmation
        if self.data.volume[0] < self.volume_ma[0]:
            return False
            
        return True
        
    def _validate_bearish_signal(self) -> bool:
        """Validate bearish breakout signal."""
        # Check RSI is not oversold
        if self.rsi[0] < self.p.rsi_oversold:
            return False
            
        # Check price is below EMA
        if self.data.close[0] > self.ema[0]:
            return False
            
        # Check volume confirmation
        if self.data.volume[0] < self.volume_ma[0]:
            return False
            
        return True 