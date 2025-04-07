"""
API Client for interacting with the Setbull Trader backend.

This module provides a client for interacting with the Go backend API, including:
- Fetching candle data (intraday and daily)
- Accessing stock information
- Getting filtered stocks from the pipeline
- Managing trading parameters and execution
"""

import requests
import logging
from typing import Dict, List, Optional, Union, Any
import time
from datetime import datetime, timedelta

logger = logging.getLogger(__name__)

class ApiClient:
    """Client for interacting with the Setbull Trader API."""
    
    def __init__(self, base_url: str, timeout: int = 30):
        """
        Initialize the API client.
        
        Args:
            base_url: Base URL of the API (e.g., 'http://localhost:8080/api/v1')
            timeout: Request timeout in seconds
        """
        self.base_url = base_url.rstrip('/')
        self.timeout = timeout
        self.session = requests.Session()
    
    def _handle_response(self, response: requests.Response) -> Dict[str, Any]:
        """
        Handle API response and error cases.
        
        Args:
            response: Response object from requests
            
        Returns:
            Parsed JSON response
            
        Raises:
            requests.HTTPError: If the response status is an error
        """
        try:
            response.raise_for_status()
            return response.json()
        except requests.HTTPError as e:
            # Try to get error details from response
            error_msg = str(e)
            try:
                error_data = response.json()
                if 'error' in error_data:
                    error_msg = f"{error_msg}: {error_data['error']}"
                elif 'message' in error_data:
                    error_msg = f"{error_msg}: {error_data['message']}"
            except (ValueError, KeyError):
                # If we can't parse the error response, use the original error
                pass
            
            logger.error(f"API error: {error_msg} (URL: {response.url}, Status: {response.status_code})")
            raise requests.HTTPError(error_msg, response=response)
        except ValueError:
            # JSON parsing error
            logger.error(f"Invalid JSON response from API: {response.text[:200]}...")
            raise ValueError(f"Invalid JSON response from API: {response.text[:200]}...")
    
    def get_candles(self, 
                   instrument_key: str, 
                   timeframe: str, 
                   start_time: Optional[datetime] = None, 
                   end_time: Optional[datetime] = None) -> Dict[str, Any]:
        """
        Fetch candle data for a specific instrument and timeframe.
        
        Args:
            instrument_key: Unique identifier for the instrument (e.g., 'NSE_EQ|INE0G5901015')
            timeframe: Timeframe for candles ('5minute', 'day')
            start_time: Start time for data range (optional)
            end_time: End time for data range (optional)
            
        Returns:
            Parsed JSON response containing candle data
        """
        endpoint = f"/candles/{instrument_key}/{timeframe}"
        
        params = {}
        if start_time:
            params['start'] = start_time
        if end_time:
            params['end'] = end_time
        
        url = f"{self.base_url}{endpoint}"
        logger.debug(f"Fetching candles from {url} with params {params}")
        
        start = time.time()
        try:
            response = self.session.get(url, params=params, timeout=self.timeout)
            response_time = time.time() - start
            logger.debug(f"Candle data response received in {response_time:.2f}s")
            
            return self._handle_response(response)
        except requests.RequestException as e:
            logger.error(f"Error fetching candle data: {str(e)}")
            raise
    
    def get_stocks(self, selected_only: bool = False) -> Dict[str, Any]:
        """
        Fetch stock data from the API.
        
        Args:
            selected_only: If True, fetch only selected stocks
            
        Returns:
            Parsed JSON response containing stock data
        """
        endpoint = "/stocks/selected" if selected_only else "/stocks"
        url = f"{self.base_url}{endpoint}"
        
        logger.debug(f"Fetching stocks from {url}")
        
        try:
            response = self.session.get(url, timeout=self.timeout)
            return self._handle_response(response)
        except requests.RequestException as e:
            logger.error(f"Error fetching stocks: {str(e)}")
            raise

    def get_health(self) -> Dict[str, Any]:
        """
        Check API health.
        
        Returns:
            Health status information
        """
        url = f"{self.base_url}/health"
        
        try:
            response = self.session.get(url, timeout=self.timeout)
            return self._handle_response(response)
        except requests.RequestException as e:
            logger.error(f"Error checking API health: {str(e)}")
            raise
    
    def get_multi_timeframe_candles(self, 
                                   instrument_key: str,
                                   timeframes: List[str],
                                   start_time: Optional[datetime] = None,
                                   end_time: Optional[datetime] = None) -> Dict[str, Any]:
        """
        Fetch candle data for multiple timeframes in a single request.
        
        Args:
            instrument_key: Unique identifier for the instrument
            timeframes: List of timeframes to fetch ('5minute', 'day', etc.)
            start_time: Start time for data range (optional)
            end_time: End time for data range (optional)
            
        Returns:
            Parsed JSON response containing candle data for all timeframes
        """
        endpoint = f"/candles/{instrument_key}/multi"
        
        payload = {
            "timeframes": timeframes
        }
        
        if start_time:
            payload["start"] = start_time.isoformat()
        if end_time:
            payload["end"] = end_time.isoformat()
        
        url = f"{self.base_url}{endpoint}"
        logger.debug(f"Fetching multi-timeframe candles from {url}")
        
        try:
            response = self.session.post(url, json=payload, timeout=self.timeout)
            return self._handle_response(response)
        except requests.RequestException as e:
            logger.error(f"Error fetching multi-timeframe candle data: {str(e)}")
            raise
    
    def get_daily_candles(self, 
                         instrument_key: str, 
                         start_date: Optional[datetime] = None,
                         end_date: Optional[datetime] = None) -> Dict[str, Any]:
        """
        Convenience method to fetch daily candle data.
        
        Args:
            instrument_key: Unique identifier for the instrument
            start_date: Start date for data range (optional)
            end_date: End date for data range (optional)
            
        Returns:
            Parsed JSON response containing daily candle data
        """
        return self.get_candles(
            instrument_key=instrument_key,
            timeframe='day',
            start_time=start_date,
            end_time=end_date
        )
    
    def get_filtered_stocks(self) -> Dict[str, Any]:
        """
        Get stocks filtered by the pipeline.
        
        Returns:
            Parsed JSON response containing filtered stocks
        """
        url = f"{self.base_url}/filter-pipeline/run"
        logger.debug(f"Fetching filtered stocks from {url}")
        
        try:
            response = self.session.post(url, json={}, timeout=self.timeout)
            return self._handle_response(response)
        except requests.RequestException as e:
            logger.error(f"Error fetching filtered stocks: {str(e)}")
            raise
    
    def get_stock_by_symbol(self, symbol: str) -> Dict[str, Any]:
        """
        Get stock details by symbol.
        
        Args:
            symbol: Stock symbol
            
        Returns:
            Stock details
            
        Raises:
            requests.HTTPError: If the stock is not found
        """
        url = f"{self.base_url}/stocks/universe/{symbol}"
        logger.debug(f"Fetching stock details for {symbol}")
        
        try:
            response = self.session.get(url, timeout=self.timeout)
            return self._handle_response(response)
        except requests.RequestException as e:
            logger.error(f"Error fetching stock details for {symbol}: {str(e)}")
            raise
    
    def get_stock_by_security_id(self, security_id: str) -> Dict[str, Any]:
        """
        Get stock details by security ID.
        
        Args:
            security_id: Stock security ID
            
        Returns:
            Stock details
            
        Raises:
            requests.HTTPError: If the stock is not found
        """
        url = f"{self.base_url}/stocks/security/{security_id}"
        logger.debug(f"Fetching stock details for security ID {security_id}")
        
        try:
            response = self.session.get(url, timeout=self.timeout)
            return self._handle_response(response)
        except requests.RequestException as e:
            logger.error(f"Error fetching stock details for security ID {security_id}: {str(e)}")
            raise
    
    def get_trading_parameters(self, stock_id: str) -> Dict[str, Any]:
        """
        Get trading parameters for a stock.
        
        Args:
            stock_id: Stock ID
            
        Returns:
            Trading parameters
        """
        url = f"{self.base_url}/parameters/stock/{stock_id}"
        logger.debug(f"Fetching trading parameters for stock {stock_id}")
        
        try:
            response = self.session.get(url, timeout=self.timeout)
            return self._handle_response(response)
        except requests.RequestException as e:
            logger.error(f"Error fetching trading parameters for stock {stock_id}: {str(e)}")
            raise
