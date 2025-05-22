"""
Data processor for candle data.

This module processes candle data from the API into formats suitable for strategy calculations,
with a focus on morning range extraction.
"""

import logging
import pandas as pd
import numpy as np
from typing import Dict, List, Optional, Union, Any, Tuple
from datetime import datetime, time, timedelta
import asyncio
import aiohttp
import pytz
from urllib.parse import urlencode
import backoff
from aiohttp import ClientError, ClientResponseError
from .intraday_data_processor import IntradayDataProcessor
from .daily_data_processor import DailyDataProcessor

logger = logging.getLogger(__name__)

class ApiError(Exception):
    """Custom exception for API-related errors."""
    def __init__(self, message: str, status_code: Optional[int] = None, response: Optional[Dict] = None):
        self.message = message
        self.status_code = status_code
        self.response = response
        super().__init__(self.message)

class ApiClient:
    """API client for fetching candle data from the server."""
    
    def __init__(self, base_url: str = "http://localhost:8083/api/v1", max_retries: int = 3):
        """Initialize the API client.
        
        Args:
            base_url (str): Base URL for the API server
            max_retries (int): Maximum number of retry attempts
        """
        self.base_url = base_url.rstrip('/')
        self.session = None
        self.max_retries = max_retries
        
    async def __aenter__(self):
        """Create aiohttp session when entering context."""
        self.session = aiohttp.ClientSession(
            timeout=aiohttp.ClientTimeout(total=30)  # 30 second timeout
        )
        return self
        
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Close aiohttp session when exiting context."""
        if self.session:
            await self.session.close()
            
    @backoff.on_exception(
        backoff.expo,
        (ClientError, ClientResponseError, asyncio.TimeoutError),
        max_tries=3,
        max_time=30
    )
    async def _make_request(self, url: str) -> Dict[str, Any]:
        """Make an HTTP request with retry logic.
        
        Args:
            url (str): URL to request
            
        Returns:
            Dict[str, Any]: Response data
            
        Raises:
            ApiError: If the request fails after retries
        """
        try:
            async with self.session.get(url) as response:
                if response.status != 200:
                    error_text = await response.text()
                    raise ApiError(
                        f"API request failed: {response.status} - {error_text}",
                        status_code=response.status
                    )
                    
                data = await response.json()
                if not data:
                    raise ApiError("Empty response from API")
                    
                return data
                
        except (ClientError, ClientResponseError, asyncio.TimeoutError) as e:
            logger.error(f"Request failed: {str(e)}")
            raise
            
    def _validate_candle_data(self, data: Dict[str, Any]) -> None:
        """Validate candle data format."""
        if not isinstance(data, dict):
            raise ApiError("Invalid response format: expected dictionary")
        
        # Handle direct list of candles
        if isinstance(data.get('data'), list):
            candles = data['data']
            for candle in candles:
                self._validate_single_candle(candle)
            return
        
        # Handle nested data structure
        if not data.get('success', False):
            raise ApiError(f"API request failed: {data.get('error', 'Unknown error')}")
        
        inner_data = data.get('data', {})
        if not isinstance(inner_data, dict):
            raise ApiError("Invalid inner data format: expected dictionary")
        
        # Handle multiple instruments
        if 'data' in inner_data:
            candles = inner_data['data']
            if not isinstance(candles, list):
                raise ApiError("Invalid candles format: expected list")
            for candle in candles:
                self._validate_single_candle(candle)
        else:
            # Handle direct candle data
            for key, value in inner_data.items():
                if isinstance(value, list):
                    for candle in value:
                        self._validate_single_candle(candle)

    def _validate_single_candle(self, candle: Dict[str, Any]) -> None:
        """Validate a single candle's data."""
        required_fields = {'timestamp', 'open', 'high', 'low', 'close', 'volume'}
        missing_fields = required_fields - set(candle.keys())
        if missing_fields:
            raise ApiError(f"Missing required fields in candle data: {missing_fields}")
        
        # Validate numeric fields
        for field in ['open', 'high', 'low', 'close', 'volume']:
            try:
                float(candle[field])
            except (ValueError, TypeError):
                raise ApiError(f"Invalid {field} value in candle data: {candle[field]}")
                    
    async def get_candles(
        self,
        instrument_key: str,
        timeframe: str,
        start_date: str,
        end_date: str
    ) -> Dict[str, Any]:
        """Fetch candles from the API.
        
        Args:
            instrument_key (str): Instrument key (e.g., 'NSE_EQ|INE070D01027')
            timeframe (str): Timeframe ('5minute' or 'day')
            start_date (datetime): Start date
            end_date (datetime): End date
            
        Returns:
            Dict[str, Any]: API response containing candle data
            
        Raises:
            ApiError: If the request fails or data is invalid
        """
        if not self.session:
            raise RuntimeError("API client must be used as an async context manager")
            
        # Validate timeframe
        if timeframe not in ['5minute', 'day']:
            raise ValueError(f"Invalid timeframe: {timeframe}")
            
        # Build URL
        url = f"{self.base_url}/candles/{instrument_key}/{timeframe}"
        params = {
            'start': start_date,
            'end': end_date
        }
        url = f"{url}?{urlencode(params)}"
        
        try:
            # Make request with retry logic
            response = await self._make_request(url)
            
            # Validate response data
            self._validate_candle_data(response)
            
            return response
            
        except Exception as e:
            logger.error(f"Error fetching candles: {str(e)}")
            raise

class CandleProcessor:
    """Process and transform candle data for strategy calculations."""
    
    def __init__(self, config: Optional[Dict[str, Any]] = None):
        """
        Initialize the candle processor.
        
        Args:
            config: Optional configuration dictionary
        """
        self.config = config or {}
        self.api_client = ApiClient()
        
    async def __aenter__(self):
        """Create API client when entering context."""
        await self.api_client.__aenter__()
        return self
        
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Close API client when exiting context."""
        await self.api_client.__aexit__(exc_type, exc_val, exc_tb)
            
    @staticmethod
    def parse_candles(candle_data: Dict[str, Any]) -> pd.DataFrame:
        """
        Parse candle data from API response into a pandas DataFrame.
        
        Args:
            candle_data: API response containing candle data
            
        Returns:
            DataFrame with candle data
            
        Raises:
            ValueError: If data format is invalid
        """
        if not candle_data:
            logger.warning("No candle data provided")
            return pd.DataFrame()
            
        try:
            # Extract candles from the nested response structure
            candles = candle_data.get('data', {}).get('data', [])
            if not candles:
                logger.warning("Empty candle list in API response")
                return pd.DataFrame()
            
            # Convert to DataFrame
            df = pd.DataFrame(candles)
            
            # Convert timestamp to datetime but keep it as a column
            if 'timestamp' in df.columns:
                df['timestamp'] = pd.to_datetime(df['timestamp'])
                # Don't set timestamp as index
                # df.set_index('timestamp', inplace=True)
            
            # Ensure numeric columns are the correct type
            numeric_cols = ['open', 'high', 'low', 'close', 'volume', 'openInterest']
            for col in numeric_cols:
                if col in df.columns:
                    df[col] = pd.to_numeric(df[col], errors='coerce')
                    
            # Validate OHLC relationships
            invalid_ohlc = (
                (df['high'] < df['low']) |
                (df['open'] > df['high']) |
                (df['open'] < df['low']) |
                (df['close'] > df['high']) |
                (df['close'] < df['low'])
            )
            
            if invalid_ohlc.any():
                logger.warning("Invalid OHLC relationships found in data")
                # Fix invalid relationships
                df['high'] = df[['open', 'high', 'close']].max(axis=1)
                df['low'] = df[['open', 'low', 'close']].min(axis=1)
            
            # Sort by timestamp
            df = df.sort_values('timestamp')
            
            return df
            
        except Exception as e:
            logger.error(f"Error parsing candle data: {str(e)}")
            raise ValueError(f"Failed to parse candle data: {str(e)}")

    def extract_morning_range(self, 
                            df: pd.DataFrame, 
                            range_type: str = '5MR',
                            market_open: time = time(9, 15),
                            tz=None) -> Tuple[pd.DataFrame, Dict[str, float]]:
        """
        Extract the morning range from candle data.
        
        Args:
            df: DataFrame with candle data
            range_type: Type of morning range ('5MR' or '15MR')
            market_open: Market opening time
            tz: Timezone for the data (if None, assumed to be in local timezone)
            
        Returns:
            Tuple containing:
                - DataFrame with only the morning range candles
                - Dict with morning range values (high, low, size)
        """
        if df.empty:
            logger.warning("Empty DataFrame provided for morning range extraction")
            return df, {'high': np.nan, 'low': np.nan, 'size': np.nan}
        
        # Reset index if timestamp is the index
        if isinstance(df.index, pd.DatetimeIndex):
            df = df.reset_index()
        
        # Ensure timestamp column exists
        if 'timestamp' not in df.columns:
            logger.error("DataFrame must contain a 'timestamp' column")
            return df, {'high': np.nan, 'low': np.nan, 'size': np.nan}
        
        # Filter only the candles for the morning range calculation
        morning_end_time = None
        if range_type == '5MR':
            # 5-minute morning range: 9:15 to 9:20
            morning_end_time = time(9, 20)
        elif range_type == '15MR':
            # 15-minute morning range: 9:15 to 9:30
            morning_end_time = time(9, 30)
        else:
            logger.error(f"Invalid range type: {range_type}")
            return df, {'high': np.nan, 'low': np.nan, 'size': np.nan}
        
        # Filter candles within the morning range time window
        def is_in_morning_range(timestamp):
            t = timestamp.time()
            return market_open <= t < morning_end_time
        
        morning_candles = df[df['timestamp'].apply(is_in_morning_range)]
        
        if morning_candles.empty:
            logger.warning(f"No candles found within the {range_type} time window")
            return morning_candles, {'high': np.nan, 'low': np.nan, 'size': np.nan}
        
        # Calculate the morning range values
        mr_high = morning_candles['high'].max()
        mr_low = morning_candles['low'].min()
        mr_size = mr_high - mr_low
        
        mr_values = {
            'high': mr_high,
            'low': mr_low,
            'size': mr_size,
            'candle_count': len(morning_candles)
        }
        
        logger.debug(f"Extracted {range_type} values: high={mr_high}, low={mr_low}, size={mr_size}")
        
        return morning_candles, mr_values
    
    def filter_trading_day_candles(self, 
                                 df: pd.DataFrame, 
                                 trading_date: Optional[datetime] = None,
                                 market_open: time = time(9, 15),
                                 market_close: time = time(15, 30),
                                 tz=None) -> pd.DataFrame:
        """
        Filter candles for a specific trading day.
        
        Args:
            df: DataFrame with candle data
            trading_date: Date to filter (if None, use the latest date in the data)
            market_open: Market opening time
            market_close: Market closing time
            tz: Timezone for the data
            
        Returns:
            DataFrame with filtered candles
        """
        logger.info(f"Filtering trading day candles for date: {trading_date}")
        if df.empty:
            return df
        
        # Reset index if timestamp is the index
        if isinstance(df.index, pd.DatetimeIndex):
            df = df.reset_index()
        
        # Ensure timestamp column exists
        if 'timestamp' not in df.columns:
            logger.error("DataFrame must contain a 'timestamp' column")
            return df
        
        # Determine the trading date if not provided
        if trading_date is None:
            trading_date = df['timestamp'].max().date()
        else:
            trading_date = trading_date.date()
        
        logger.debug(f"Filtering candles for trading date: {trading_date}")
        
        # Filter candles for the trading day and within market hours
        def is_in_trading_day(timestamp):
            if timestamp.date() != trading_date:
                return False
            
            t = timestamp.time()
            return market_open <= t <= market_close
        
        return df[df['timestamp'].apply(is_in_trading_day)]
    
    # Phase 1.7: Added functions for ATR calculation and trading day handling
    
    async def calculate_atr(self, current_candle: pd.DataFrame, period: int = 14) -> float:
        """
        Calculate Average True Range (ATR) using daily candles.
        
        Args:
            current_candle: Current 5-minute candle DataFrame
            period: Period for ATR calculation (default: 14)
            
        Returns:
            float: Latest ATR value
            
        Raises:
            ValueError: If insufficient data or invalid period
        """
        try:
            # Get the date from current candle
            current_date = pd.to_datetime(current_candle['timestamp'].iloc[0]).date()
            start_date = (current_date - pd.Timedelta(days=period + 10)).strftime('%Y-%m-%d')
            current_date = current_date.strftime('%Y-%m-%d')
            start_date = start_date + "T00:00:00+05:30"
            current_date = current_date + "T00:00:00+05:30"
            
            # Get instrument key from config or use default
            instrument_key = self.config.get('instrument_key', 'NSE_EQ|INE070D01027').get('key')
            logger.info(f"Fetching daily candles for {instrument_key} from {start_date} to {current_date}")

            # Fetch daily candles for the period
            daily_candles = await self.api_client.get_candles(
                instrument_key=instrument_key,
                timeframe='day',
                start_date=start_date,
                end_date=current_date
            )

            # Fetch only 14 period days of candles
            # daily_candles is a dict so fetch only 14 period days of candles
            # daily_candles = daily_candles['data']['data'][:period]

            daily_candles = self.parse_candles(daily_candles)
            daily_candles = daily_candles.iloc[:period]
            logger.info(f"Number of daily candles: {len(daily_candles)}")
            if daily_candles.empty or len(daily_candles) < period:
                raise ValueError(f"Need at least {period} daily candles for ATR calculation")
                
            # Calculate True Range for each daily candle
            high = daily_candles['high']
            low = daily_candles['low']
            close_prev = daily_candles['close'].shift(1)  # Previous close
            
            # Handle NaN in first row's previous close
            close_prev.iloc[0] = daily_candles['open'].iloc[0]  # Use open price for first candle
            
            # Calculate True Range according to TradingView formula
            tr1 = high - low
            tr2 = abs(high - close_prev)
            tr3 = abs(low - close_prev)
            
            # Get the maximum of the three values for each row
            true_ranges = pd.DataFrame({
                'hl': tr1,
                'hc': tr2,
                'lc': tr3
            }).max(axis=1)
            
            # Initialize ATR series with NaN values
            atr_values = pd.Series(index=true_ranges.index, dtype=float)
            
            # First ATR value is simple average of true ranges for period
            if len(true_ranges) >= period:
                atr_values.iloc[period-1] = true_ranges.iloc[:period].mean()
                
                # Calculate remaining ATR values using Wilder's smoothing formula
                # ATR = ((period-1) * previousATR + currentTR) / period
                for i in range(period, len(true_ranges)):
                    atr_values.iloc[i] = ((period-1) * atr_values.iloc[i-1] + true_ranges.iloc[i]) / period
                
                # Return the latest ATR value
                atr = atr_values.iloc[-1]
                logger.info(f"Calculated ATR-{period} using daily candles: {atr}")
                return atr
            else:
                # Not enough data for calculation
                raise ValueError(f"Need at least {period} daily candles for ATR calculation. Got {len(true_ranges)}")
            
        except Exception as e:
            logger.error(f"Error calculating ATR: {str(e)}")
            raise ValueError(f"Failed to calculate ATR: {str(e)}")

    async def calculate_morning_range(self, candles: pd.DataFrame) -> Dict[str, float]:
        """
        Calculate morning range values including MR value.
        
        Args:
            candles: DataFrame with candle data for the day
            
        Returns:
            Dict with:
            - mr_high: Morning range high
            - mr_low: Morning range low
            - mr_size: Morning range size
            - mr_value: Morning range value (14-day ATR / MR size)
            - is_valid: Boolean indicating if MR value > 3
            - error: Error message if any (None if successful)
        """
        if candles.empty:
            logger.warning("Empty DataFrame provided for morning range calculation")
            return {
                'mr_high': 0,
                'mr_low': 0,
                'mr_size': 0,
                'mr_value': 0,
                'is_valid': False,
                'error': 'Empty DataFrame provided'
            }
            
        try:
            # Validate required columns
            required_columns = ['timestamp', 'high', 'low']
            missing_columns = [col for col in required_columns if col not in candles.columns]
            if missing_columns:
                error_msg = f"Missing required columns: {missing_columns}"
                logger.error(error_msg)
                return {
                    'mr_high': 0,
                    'mr_low': 0,
                    'mr_size': 0,
                    'mr_value': 0,
                    'is_valid': False,
                    'error': error_msg
                }
            
            # Calculate morning range values
            morning_candle = candles.iloc[0]
            mr_high = morning_candle['high']
            mr_low = morning_candle['low']
            mr_size = mr_high - mr_low
            
            if mr_size <= 0:
                error_msg = "Invalid morning range size (high <= low)"
                logger.error(error_msg)
                return {
                    'mr_high': mr_high,
                    'mr_low': mr_low,
                    'mr_size': mr_size,
                    'mr_value': 0,
                    'is_valid': False,
                    'error': error_msg
                }

            if mr_size < 1:
                error_msg = "Invalid morning range size. Size is less than 1"
                logger.error(error_msg)
                return {
                    'mr_high': mr_high,
                    'mr_low': mr_low,
                    'mr_size': mr_size,
                    'mr_value': 0,
                    'is_valid': False,
                    'error': error_msg
                }
            
            # Calculate 14-day ATR using daily candles
            logger.info("DAILY 14-ATR: {morning_candle['DAILY_ATR_14']}")
            atr_14 = morning_candle['DAILY_ATR_14']
            
            if atr_14 <= 0:
                error_msg = "Invalid ATR value (must be positive)"
                logger.error(error_msg)
                return {
                    'mr_high': mr_high,
                    'mr_low': mr_low,
                    'mr_size': mr_size,
                    'mr_value': 0,
                    'is_valid': False,
                    'error': error_msg
                }
            
            # Calculate MR value
            mr_value = (atr_14 / mr_size) * 1.1
            
            # Validate MR value
            is_valid = mr_value > 3
            
            logger.info(
                f"Morning Range Calculation - "
                f"High: {mr_high:.2f}, "
                f"Low: {mr_low:.2f}, "
                f"Size: {mr_size:.2f}, "
                f"ATR: {atr_14:.2f}, "
                f"MR Value: {mr_value:.2f}, "
                f"Valid: {is_valid}"
            )
            
            return {
                'mr_high': mr_high,
                'mr_low': mr_low,
                'mr_size': mr_size,
                'mr_value': mr_value,
                'is_valid': is_valid,
                'error': None
            }
            
        except Exception as e:
            error_msg = f"Error calculating morning range: {str(e)}"
            logger.error(error_msg)
            return {
                'mr_high': 0,
                'mr_low': 0,
                'mr_size': 0,
                'mr_value': 0,
                'is_valid': False,
                'error': error_msg
            }

    async def load_intraday_data(
        self, 
        instrument_key: str,
        start_date: str,
        end_date: str,
        timeframe: str = '5minute'
    ) -> pd.DataFrame:
        """Load intraday data for multiple instruments.
        
        Args:
            instrument_key: Instrument key or list of instrument keys
            start_date: Start date in ISO format
            end_date: End date in ISO format
            timeframe: Timeframe for candles (default: 5minute)
            
        Returns:
            DataFrame containing candle data
        """
        # Handle single instrument
        if isinstance(instrument_key, str):
            async with self.api_client as client:
                response = await client.get_candles(
                    instrument_key=instrument_key,
                    timeframe=timeframe,
                    start_date=start_date,
                    end_date=end_date
                )
                return self.parse_candles(response)
        
        # Handle multiple instruments
        all_data = []
        async with self.api_client as client:
            for key in instrument_key:
                try:
                    response = await client.get_candles(
                        instrument_key=key,
                        timeframe=timeframe,
                        start_date=start_date,
                        end_date=end_date
                    )
                    df = self.parse_candles(response)
                    df['instrument_key'] = key
                    all_data.append(df)
                except Exception as e:
                    logger.error(f"Error loading data for {key}: {str(e)}")
                    continue
        
        if not all_data:
            raise ApiError("No data loaded for any instrument")
        
        return pd.concat(all_data, ignore_index=True)

    async def load_daily_data(
        self,
        instrument_key: str,
        start_date: str,
        end_date: str
    ) -> pd.DataFrame:
        """Load daily candle data for the specified instrument and date range.

        Args:
            instrument_key (str): Unique identifier for the instrument
            start_date (datetime): Start date for data loading
            end_date (datetime): End date for data loading

        Returns:
            pd.DataFrame: DataFrame containing daily candle data with columns:
                         timestamp, open, high, low, close, volume, open_interest

        Raises:
            ApiError: If API request fails
            ValueError: If data processing fails
        """
        try:
            logger.info(f"Loading daily data for {instrument_key} from {start_date} to {end_date}")
            
            async with ApiClient() as client:
                # Fetch daily candles from API
                response = await client.get_candles(
                    instrument_key=instrument_key,
                    timeframe='day',
                    start_date=start_date,
                    end_date=end_date
                )
                
                # Parse the response into DataFrame
                df = self.parse_candles(response)
                
                if df.empty:
                    logger.warning("No candles found in the response")
                    return df
                
                logger.info(f"Successfully loaded {len(df)} daily candles")
                return df
                
        except ApiError as e:
            logger.error(f"API error loading daily data: {str(e)}")
            raise
        except Exception as e:
            logger.error(f"Error loading daily data: {str(e)}")
            raise ValueError(f"Failed to load daily data: {str(e)}")

    def process_candles(
        self,
        df: pd.DataFrame,
        atr_period: int = 14,
        vwap_period: int = None,
        add_indicators: bool = False
    ) -> pd.DataFrame:
        """Process raw candle data by adding technical indicators and derived values.

        Args:
            df (pd.DataFrame): Raw candle data with OHLCV columns
            atr_period (int, optional): Period for ATR calculation. Defaults to 14.
            vwap_period (int, optional): Period for VWAP calculation. If None, calculates daily VWAP.
            add_indicators (bool, optional): Whether to add technical indicators. Defaults to True.

        Returns:
            pd.DataFrame: Processed DataFrame with additional columns for indicators
        """
        try:
            logger.info("Processing candle data with indicators")
            
            # Make a copy to avoid modifying original data
            processed_df = df.copy()
            
            # Ensure DataFrame is sorted by timestamp
            if 'timestamp' in processed_df.columns:
                processed_df = processed_df.sort_values('timestamp')
            
            if add_indicators:
                # Add ATR
                processed_df['atr'] = self.calculate_atr(processed_df, period=atr_period)
                
                # Calculate VWAP
                if vwap_period:
                    # Rolling VWAP for specified period
                    typical_price = (processed_df['high'] + processed_df['low'] + processed_df['close']) / 3
                    processed_df['vwap'] = (typical_price * processed_df['volume']).rolling(window=vwap_period).sum() / \
                                         processed_df['volume'].rolling(window=vwap_period).sum()
                else:
                    # Daily VWAP
                    processed_df['date'] = processed_df['timestamp'].dt.date
                    typical_price = (processed_df['high'] + processed_df['low'] + processed_df['close']) / 3
                    cumulative_tp_vol = (typical_price * processed_df['volume']).groupby(processed_df['date']).cumsum()
                    cumulative_vol = processed_df['volume'].groupby(processed_df['date']).cumsum()
                    processed_df['vwap'] = cumulative_tp_vol / cumulative_vol
                    processed_df.drop('date', axis=1, inplace=True)
                
                # Add momentum indicators
                processed_df['roc'] = processed_df['close'].pct_change(periods=1)  # Rate of Change
                processed_df['momentum'] = processed_df['close'] - processed_df['close'].shift(1)
                processed_df['rsi'] = self._calculate_rsi(processed_df['close'], period=14)
                
                # Add moving averages
                processed_df['sma_20'] = processed_df['close'].rolling(window=20).mean()
                processed_df['ema_20'] = processed_df['close'].ewm(span=20, adjust=False).mean()
                processed_df['sma_50'] = processed_df['close'].rolling(window=50).mean()
                processed_df['ema_50'] = processed_df['close'].ewm(span=50, adjust=False).mean()
                
                # Add volatility indicators
                processed_df['daily_range'] = processed_df['high'] - processed_df['low']
                processed_df['daily_range_pct'] = processed_df['daily_range'] / processed_df['close'] * 100
                processed_df['bollinger_upper'], processed_df['bollinger_lower'] = self._calculate_bollinger_bands(processed_df['close'])
                
                # Add volume analysis
                processed_df['volume_sma'] = processed_df['volume'].rolling(window=20).mean()
                processed_df['volume_ratio'] = processed_df['volume'] / processed_df['volume_sma']
                processed_df['obv'] = self._calculate_obv(processed_df)
                
                # Add trend indicators
                processed_df['adx'] = self._calculate_adx(processed_df)
                processed_df['macd'], processed_df['macd_signal'] = self._calculate_macd(processed_df['close'])
                
                # Clean up NaN values
                processed_df = processed_df.fillna(method='bfill')
                
                logger.info("Successfully added technical indicators to candle data")
            
            return processed_df
            
        except Exception as e:
            logger.error(f"Error processing candle data: {str(e)}")
            raise

    def _calculate_rsi(self, prices: pd.Series, period: int = 14) -> pd.Series:
        """Calculate Relative Strength Index."""
        delta = prices.diff()
        gain = (delta.where(delta > 0, 0)).rolling(window=period).mean()
        loss = (-delta.where(delta < 0, 0)).rolling(window=period).mean()
        rs = gain / loss
        return 100 - (100 / (1 + rs))

    def _calculate_bollinger_bands(self, prices: pd.Series, period: int = 20, std_dev: float = 2) -> Tuple[pd.Series, pd.Series]:
        """Calculate Bollinger Bands."""
        sma = prices.rolling(window=period).mean()
        std = prices.rolling(window=period).std()
        upper_band = sma + (std * std_dev)
        lower_band = sma - (std * std_dev)
        return upper_band, lower_band

    def _calculate_obv(self, df: pd.DataFrame) -> pd.Series:
        """Calculate On Balance Volume."""
        obv = (np.sign(df['close'].diff()) * df['volume']).fillna(0).cumsum()
        return obv

    def _calculate_adx(self, df: pd.DataFrame, period: int = 14) -> pd.Series:
        """Calculate Average Directional Index."""
        high = df['high']
        low = df['low']
        close = df['close']
        
        plus_dm = high.diff()
        minus_dm = low.diff()
        plus_dm[plus_dm < 0] = 0
        minus_dm[minus_dm > 0] = 0
        
        tr1 = pd.DataFrame(high - low)
        tr2 = pd.DataFrame(abs(high - close.shift(1)))
        tr3 = pd.DataFrame(abs(low - close.shift(1)))
        frames = [tr1, tr2, tr3]
        tr = pd.concat(frames, axis=1, join='inner').max(axis=1)
        atr = tr.rolling(period).mean()
        
        plus_di = 100 * (plus_dm.rolling(period).mean() / atr)
        minus_di = abs(100 * (minus_dm.rolling(period).mean() / atr))
        dx = (abs(plus_di - minus_di) / abs(plus_di + minus_di)) * 100
        adx = dx.rolling(period).mean()
        
        return adx

    def _calculate_macd(self, prices: pd.Series, fast_period: int = 12, slow_period: int = 26, signal_period: int = 9) -> Tuple[pd.Series, pd.Series]:
        """Calculate MACD and Signal line."""
        fast_ema = prices.ewm(span=fast_period, adjust=False).mean()
        slow_ema = prices.ewm(span=slow_period, adjust=False).mean()
        macd = fast_ema - slow_ema
        signal = macd.ewm(span=signal_period, adjust=False).mean()
        return macd, signal

    def get_valid_trading_dates(self, df: pd.DataFrame) -> List[datetime.date]:
        """
        Extract all valid trading dates from a DataFrame of candles.
        
        Args:
            df: DataFrame with candle data (must contain 'timestamp' column)
            
        Returns:
            List of unique trading dates in ascending order
        """
        if df.empty:
            return []
        
        # Reset index if timestamp is the index
        if isinstance(df.index, pd.DatetimeIndex):
            df = df.reset_index()
        
        # Ensure timestamp column exists
        if 'timestamp' not in df.columns:
            logger.error("DataFrame must contain a 'timestamp' column")
            return []
        
        # Extract unique dates
        dates = df['timestamp'].dt.date.unique()
        
        # Sort in ascending order
        dates.sort()
        
        return list(dates)
    
    def is_valid_trading_day(self, date: datetime.date, 
                            market_holidays: Optional[List[datetime.date]] = None,
                            weekend_days: Optional[List[int]] = None) -> bool:
        """
        Check if a given date is a valid trading day.
        
        Args:
            date: Date to check
            market_holidays: List of market holidays (dates)
            weekend_days: List of weekend day numbers (0=Monday, 6=Sunday)
                         Default is [5, 6] for Saturday and Sunday
            
        Returns:
            True if it's a valid trading day, False otherwise
        """
        if weekend_days is None:
            weekend_days = [5, 6]  # Saturday and Sunday
            
        if market_holidays is None:
            market_holidays = []
            
        # Check if it's a weekend
        if date.weekday() in weekend_days:
            return False
            
        # Check if it's a holiday
        if date in market_holidays:
            return False
            
        return True

    # ============================================
    # Method to load Intraday and Daily candles
    # ============================================
    async def load_and_process_candles(
        self,
        instrument_key: str,
        name: str,
        start_date: str,
        end_date: str,
        timeframe: str = '5minute'
    ) -> Tuple[pd.DataFrame]:
        """
        Load and process candles with both intraday and daily indicators.
        Optionally create a Backtrader data feed.
        
        Args:
            instrument_key: Instrument key
            start_date: Start date in ISO format
            end_date: End date in ISO format
            timeframe: Timeframe for candles (default: 5minute)
            
        Returns:
            Tuple containing:
                - DataFrame with processed candles and indicators
                - Optional MorningRangeDataFeed instance
        """
        try:
            # Load intraday candles
            intraday_df = await self.load_intraday_data(
                instrument_key=instrument_key,
                start_date=start_date,
                end_date=end_date,
                timeframe=timeframe
            )
            
            if intraday_df.empty:
                logger.warning("No intraday data loaded")
                return pd.DataFrame(), None
            
            # make start_date to time and make it 50 days back from start_date
            start_date = start_date.split('T')[0]
            start_date = (datetime.strptime(start_date, '%Y-%m-%d') - timedelta(days=50)).strftime('%Y-%m-%d')
            
            # Load daily candles for the same period
            daily_df = await self.load_daily_data(
                instrument_key=instrument_key,
                start_date=start_date + "T09:15:00+05:30",
                end_date=end_date
            )
            
            if daily_df.empty:
                logger.warning("No daily data loaded")
                return pd.DataFrame(), None
            
            # Process candles with both intraday and daily indicators
            processed_df = self.process_candles(intraday_df, daily_df)
            return processed_df
            
        except Exception as e:
            logger.error(f"Error loading and processing candles: {str(e)}")
            raise

# ======================================================================
    # Method to make Information with Intraday and Daily candles together
    # ======================================================================
    def process_candles(self, candles: Union[pd.DataFrame, Dict[str, Any]], daily_candles: Optional[pd.DataFrame] = None) -> pd.DataFrame:
        """
        Process candle data and calculate indicators.
        
        Args:
            candles: DataFrame or dict with candle data
            daily_candles: Optional DataFrame with daily data
            
        Returns:
            Processed DataFrame with indicators
        """
        try:
            # Convert to DataFrame if needed
            if isinstance(candles, dict):
                df = self.parse_candles(candles)
            else:
                df = candles.copy()
            
            # Ensure timestamp is datetime
            df['timestamp'] = pd.to_datetime(df['timestamp'])
            
            # Sort by timestamp
            df = df.sort_values('timestamp')
            
            # Process intraday data
            df = IntradayDataProcessor(self.config).process(df)
            
            # Process daily data if available
            if daily_candles is not None:
                # Process daily data
                processed_daily_df = DailyDataProcessor(self.config).process(daily_candles)
                
                # Merge daily data with intraday data
                processed_daily_df['date'] = processed_daily_df['timestamp'].dt.date
                df['date'] = df['timestamp'].dt.date
                
                # Get all columns from daily data except timestamp and date
                daily_columns = [col for col in processed_daily_df.columns 
                               if col not in ['timestamp', 'date']]
                # remove open, high, low, close, volume, openInterest from daily_columns
                daily_columns.remove('id')
                daily_columns.remove('instrumentKey')
                daily_columns.remove('open')
                daily_columns.remove('high')
                daily_columns.remove('low')
                daily_columns.remove('close')
                daily_columns.remove('volume')
                daily_columns.remove('openInterest')
                daily_columns.remove('timeInterval')
                daily_columns.remove('createdAt')
                
                # Merge the data
                df = df.merge(
                    processed_daily_df[['date'] + daily_columns],
                    on='date',
                    how='left'
                )
                
                # Drop the temporary date column
                df = df.drop('date', axis=1)
            
            # Forward fill NaN values
            df = df.fillna(method='ffill')
            
            return df
            
        except Exception as e:
            logger.error(f"Error processing candles: {str(e)}")
            raise