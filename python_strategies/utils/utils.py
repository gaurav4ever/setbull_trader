import numpy as np
import pytz
from datetime import datetime, time
from typing import Union

def convert_numpy_types(obj):
    if isinstance(obj, dict):
        return {k: convert_numpy_types(v) for k, v in obj.items()}
    elif isinstance(obj, list):
        return [convert_numpy_types(item) for item in obj]
    elif isinstance(obj, np.generic):
        return obj.item()
    else:
        return obj

def convert_utc_to_ist(utc_datetime: datetime) -> time:
    """
    Convert UTC datetime to IST time.
    
    Args:
        utc_datetime: UTC datetime to convert
        
    Returns:
        time: Time in IST
    """
    utc = pytz.utc
    ist = pytz.timezone('Asia/Kolkata')
    dt_ist = utc.localize(utc_datetime).astimezone(ist)
    return dt_ist

def get_nearest_price(price: Union[float, int]) -> float:
    """
    Get the nearest whole number or 5 multiple price.
    
    Rules:
    - If decimal part is <= 0.2, round down to nearest whole number
    - If decimal part is > 0.2 and < 0.3, round down to nearest whole number
    - If decimal part is >= 0.3, round up to nearest 5 multiple
    
    Examples:
    >>> get_nearest_price(95.03)  # Returns 95.05
    >>> get_nearest_price(93.01)  # Returns 93.00
    >>> get_nearest_price(92.2)   # Returns 92.00
    >>> get_nearest_price(92.3)   # Returns 92.00
    
    Args:
        price: The price to round
        
    Returns:
        float: The rounded price
    """
    # Get the decimal part
    decimal_part = price - int(price)
    
    if decimal_part <= 0.2:
        # Round down to nearest whole number
        return float(int(price))
    elif decimal_part < 0.3:
        # Round down to nearest whole number
        return float(int(price))
    else:
        # Round up to nearest 5 multiple
        base = int(price)
        return base + 0.05