"""
Time utilities for trading strategies.

This module provides functions for working with market hours, trading dates,
and time-related operations for trading strategies.
"""

import datetime
from datetime import time, date, datetime, timedelta
from typing import List, Optional, Tuple, Union
import logging
import pytz

logger = logging.getLogger(__name__)

# Default market hours for NSE (Indian National Stock Exchange)
DEFAULT_MARKET_OPEN = time(9, 15)  # 9:15 AM
DEFAULT_MARKET_CLOSE = time(15, 30)  # 3:30 PM
DEFAULT_TIMEZONE = pytz.timezone('Asia/Kolkata')  # IST

def is_market_open(timestamp: datetime, 
                  market_open: time = DEFAULT_MARKET_OPEN,
                  market_close: time = DEFAULT_MARKET_CLOSE,
                  timezone: pytz.timezone = DEFAULT_TIMEZONE) -> bool:
    """
    Check if the market is open at the given timestamp.
    
    Args:
        timestamp: The timestamp to check
        market_open: Market opening time
        market_close: Market closing time
        timezone: Timezone for the market
        
    Returns:
        True if market is open, False otherwise
    """
    # Convert timestamp to the market timezone if it has a tzinfo
    if timestamp.tzinfo is not None:
        timestamp = timestamp.astimezone(timezone)
    
    # Extract time component
    current_time = timestamp.time()
    
    # Check if it's within market hours
    return market_open <= current_time <= market_close

def is_trading_day(check_date: Union[date, datetime], 
                  market_holidays: Optional[List[date]] = None,
                  weekend_days: Optional[List[int]] = None) -> bool:
    """
    Check if a given date is a trading day.
    
    Args:
        check_date: Date to check
        market_holidays: List of market holidays
        weekend_days: List of weekend day numbers (0=Monday, 6=Sunday)
                     Default is [5, 6] for Saturday and Sunday
        
    Returns:
        True if it's a trading day, False otherwise
    """
    if isinstance(check_date, datetime):
        check_date = check_date.date()
        
    if weekend_days is None:
        weekend_days = [5, 6]  # Saturday and Sunday
        
    if market_holidays is None:
        market_holidays = []
        
    # Check if it's a weekend
    if check_date.weekday() in weekend_days:
        return False
        
    # Check if it's a holiday
    if check_date in market_holidays:
        return False
        
    return True

def get_next_trading_day(from_date: Union[date, datetime],
                        market_holidays: Optional[List[date]] = None,
                        weekend_days: Optional[List[int]] = None) -> date:
    """
    Get the next trading day from a given date.
    
    Args:
        from_date: Starting date
        market_holidays: List of market holidays
        weekend_days: List of weekend day numbers
        
    Returns:
        Next trading day
    """
    if isinstance(from_date, datetime):
        from_date = from_date.date()
    
    next_day = from_date + timedelta(days=1)
    
    # Keep checking days until we find a trading day
    while not is_trading_day(next_day, market_holidays, weekend_days):
        next_day += timedelta(days=1)
        
    return next_day

def get_previous_trading_day(from_date: Union[date, datetime],
                           market_holidays: Optional[List[date]] = None,
                           weekend_days: Optional[List[int]] = None) -> date:
    """
    Get the previous trading day from a given date.
    
    Args:
        from_date: Starting date
        market_holidays: List of market holidays
        weekend_days: List of weekend day numbers
        
    Returns:
        Previous trading day
    """
    if isinstance(from_date, datetime):
        from_date = from_date.date()
    
    prev_day = from_date - timedelta(days=1)
    
    # Keep checking days until we find a trading day
    while not is_trading_day(prev_day, market_holidays, weekend_days):
        prev_day -= timedelta(days=1)
        
    return prev_day

def get_trading_days_between(start_date: Union[date, datetime],
                           end_date: Union[date, datetime],
                           market_holidays: Optional[List[date]] = None,
                           weekend_days: Optional[List[int]] = None) -> List[date]:
    """
    Get a list of all trading days between two dates (inclusive).
    
    Args:
        start_date: Starting date
        end_date: Ending date
        market_holidays: List of market holidays
        weekend_days: List of weekend day numbers
        
    Returns:
        List of trading days
    """
    if isinstance(start_date, datetime):
        start_date = start_date.date()
    if isinstance(end_date, datetime):
        end_date = end_date.date()
    
    # Ensure start date is before end date
    if start_date > end_date:
        start_date, end_date = end_date, start_date
    
    trading_days = []
    current_date = start_date
    
    while current_date <= end_date:
        if is_trading_day(current_date, market_holidays, weekend_days):
            trading_days.append(current_date)
        current_date += timedelta(days=1)
        
    return trading_days

def get_market_hours(exchange: str = 'NSE') -> Tuple[time, time, pytz.timezone]:
    """
    Get the market hours for a given exchange.
    
    Args:
        exchange: Exchange code (NSE, BSE, etc.)
        
    Returns:
        Tuple with (market_open, market_close, timezone)
    """
    exchange = exchange.upper()
    
    if exchange == 'NSE' or exchange == 'BSE':
        return (DEFAULT_MARKET_OPEN, DEFAULT_MARKET_CLOSE, DEFAULT_TIMEZONE)
    elif exchange == 'NYSE':
        return (time(9, 30), time(16, 0), pytz.timezone('America/New_York'))
    elif exchange == 'NASDAQ':
        return (time(9, 30), time(16, 0), pytz.timezone('America/New_York'))
    else:
        logger.warning(f"Unknown exchange: {exchange}, using NSE hours by default")
        return (DEFAULT_MARKET_OPEN, DEFAULT_MARKET_CLOSE, DEFAULT_TIMEZONE)

def combine_date_and_time(date_value: date, time_value: time, 
                         timezone: Optional[pytz.timezone] = None) -> datetime:
    """
    Combine a date and time into a datetime object.
    
    Args:
        date_value: Date object
        time_value: Time object
        timezone: Optional timezone to apply
        
    Returns:
        DateTime object with the combined date and time
    """
    dt = datetime.combine(date_value, time_value)
    
    if timezone:
        dt = timezone.localize(dt)
        
    return dt

def format_time_for_display(dt: Union[datetime, time]) -> str:
    """
    Format a datetime or time for display in HH:MM format.
    
    Args:
        dt: Datetime or time object
        
    Returns:
        Formatted time string
    """
    if isinstance(dt, datetime):
        return dt.strftime('%H:%M')
    elif isinstance(dt, time):
        return dt.strftime('%H:%M')
    else:
        raise TypeError("Input must be a datetime or time object")

def format_date_for_display(dt: Union[datetime, date]) -> str:
    """
    Format a datetime or date for display in YYYY-MM-DD format.
    
    Args:
        dt: Datetime or date object
        
    Returns:
        Formatted date string
    """
    if isinstance(dt, datetime):
        return dt.strftime('%Y-%m-%d')
    elif isinstance(dt, date):
        return dt.strftime('%Y-%m-%d')
    else:
        raise TypeError("Input must be a datetime or date object") 