#!/usr/bin/env python3
"""
Stock Group Management Script

This script processes stocks from a text file, looks them up in the NSE Upstox JSON file,
creates them via the API, and adds them to a specified stock group.

Usage:
    python add_stocks_to_group.py

Requirements:
    - requests library: pip install requests
    - Python 3.6+

Author: Setbull Trader Team
Date: 2025
"""

import json
import requests
import time
import logging
from typing import Dict, List, Optional, Tuple
from pathlib import Path
import datetime

# Configuration
def load_config():
    """Load configuration from config.json"""
    try:
        with open('config.json', 'r') as f:
            return json.load(f)
    except Exception as e:
        print(f"Warning: Could not load config.json, using defaults: {e}")
        return {
            "api": {"base_url": "http://localhost:8083/api/v1", "timeout": 30},
            "stock_group": {"id": "d63c7112-2e10-443a-bb99-221fb9238e9e"},
            "files": {
                "nse_upstox_json": "../../nse_upstox.json",
                "stocks_input": "1_Buying_28008.txt"
            },
            "logging": {"level": "INFO", "file": "stock_processing.log"}
        }

config = load_config()
API_BASE_URL = config["api"]["base_url"]
STOCK_GROUP_ENTRY_TYPE = config["stock_group"]["entry_type"]
NSE_UPSTOX_JSON_PATH = config["files"]["nse_upstox_json"]
STOCKS_FILE_PATH = config["files"]["stocks_input"]

# Logging configuration
logging.basicConfig(
    level=getattr(logging, config["logging"]["level"]),
    format=config["logging"]["format"],
    handlers=[
        logging.FileHandler(config["logging"]["file"]),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)

def log_api_request(method: str, url: str, payload: dict = None, params: dict = None):
    """Log API request details"""
    timestamp = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S.%f")[:-3]
    logger.info(f"[{timestamp}] API REQUEST:")
    logger.info(f"  Method: {method}")
    logger.info(f"  URL: {url}")
    if params:
        logger.info(f"  Params: {json.dumps(params, indent=2)}")
    if payload:
        logger.info(f"  Payload: {json.dumps(payload, indent=2)}")
    logger.info("-" * 80)

def log_api_response(status_code: int, response_text: str, response_time: float = None):
    """Log API response details"""
    timestamp = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S.%f")[:-3]
    logger.info(f"[{timestamp}] API RESPONSE:")
    logger.info(f"  Status Code: {status_code}")
    if response_time:
        logger.info(f"  Response Time: {response_time:.3f}s")
    
    try:
        response_json = json.loads(response_text)
        logger.info(f"  Response Body: {json.dumps(response_json, indent=2)}")
    except json.JSONDecodeError:
        logger.info(f"  Response Body: {response_text}")
    
    logger.info("-" * 80)


class StockProcessor:
    """Handles stock processing operations"""
    
    def __init__(self):
        self.nse_data = None
        self.created_stock_ids = []
        self.failed_stocks = []
        self.skipped_stocks = []
        
    def load_nse_data(self) -> bool:
        """Load NSE Upstox JSON data"""
        logger.info(f"📂 Loading NSE data from: {NSE_UPSTOX_JSON_PATH}")
        try:
            with open(NSE_UPSTOX_JSON_PATH, 'r') as f:
                self.nse_data = json.load(f)
            logger.info(f"✅ Successfully loaded {len(self.nse_data)} stocks from NSE Upstox JSON")
            return True
        except Exception as e:
            logger.error(f"❌ Failed to load NSE data: {e}")
            return False
    
    def load_stocks_from_file(self) -> List[str]:
        """Load stock symbols from the text file"""
        logger.info(f"📂 Loading stocks from: {STOCKS_FILE_PATH}")
        try:
            with open(STOCKS_FILE_PATH, 'r') as f:
                stocks = [line.strip() for line in f if line.strip()]
            logger.info(f"✅ Successfully loaded {len(stocks)} stocks from {STOCKS_FILE_PATH}")
            return stocks
        except Exception as e:
            logger.error(f"❌ Failed to load stocks from file: {e}")
            return []
    
    def find_stock_in_nse_data(self, symbol: str) -> Optional[Dict]:
        """Find stock information in NSE data by trading symbol"""
        if not self.nse_data:
            return None
            
        for stock in self.nse_data:
            if stock.get('trading_symbol') == symbol:
                return stock
        return None
    
    def create_stock_via_api(self, symbol: str, name: str, securityId: str) -> Optional[str]:
        """Create stock via API and return stock ID"""
        url = f"{API_BASE_URL}/stocks"
        payload = {
            "symbol": symbol,
            "name": name,
            "securityId": securityId,
            "isSelected": False
        }
        
        # Log the request
        log_api_request("POST", url, payload=payload)
        
        try:
            start_time = time.time()
            response = requests.post(url, json=payload, timeout=config["api"]["timeout"])
            response_time = time.time() - start_time
            
            # Log the response
            log_api_response(response.status_code, response.text, response_time)
            
            if response.status_code == 201:
                response_data = response.json()
                if response_data.get('success') and response_data.get('data'):
                    stock_data = response_data['data']
                    stock_id = stock_data.get('id')
                    logger.info(f"✅ Successfully created stock: {symbol} ({name}) with ID: {stock_id}")
                    return stock_id
                else:
                    logger.error(f"❌ Invalid response format for stock {symbol}: {response_data}")
                    return None
            elif response.status_code == 409:
                # Stock already exists, try to get its ID
                logger.info(f"⚠️  Stock {symbol} already exists, fetching ID...")
                return self.get_existing_stock_id(symbol)
            else:
                logger.error(f"❌ Failed to create stock {symbol}: {response.status_code} - {response.text}")
                return None
        except Exception as e:
            logger.error(f"❌ Error creating stock {symbol}: {e}")
            return None
    
    def get_existing_stock_id(self, symbol: str) -> Optional[str]:
        """Get existing stock ID by symbol"""
        url = f"{API_BASE_URL}/stocks"
        params = {"symbol": symbol}
        
        # Log the request
        log_api_request("GET", url, params=params)
        
        try:
            start_time = time.time()
            response = requests.get(url, params=params, timeout=config["api"]["timeout"])
            response_time = time.time() - start_time
            
            # Log the response
            log_api_response(response.status_code, response.text, response_time)
            
            if response.status_code == 200:
                stocks = response.json()
                if stocks and len(stocks) > 0:
                    stock_id = stocks[0].get('id')
                    logger.info(f"✅ Found existing stock {symbol} with ID: {stock_id}")
                    return stock_id
                else:
                    logger.warning(f"⚠️  Stock {symbol} not found in existing stocks")
                    return None
            else:
                logger.error(f"❌ Failed to fetch existing stock {symbol}: {response.status_code} - {response.text}")
                return None
        except Exception as e:
            logger.error(f"❌ Error fetching existing stock {symbol}: {e}")
            return None
    
    def add_stocks_to_group(self, stock_ids: List[str]) -> bool:
        """Add stocks to the specified group"""
        if not stock_ids:
            logger.warning("⚠️  No stock IDs to add to group")
            return True
            
        url = f"{API_BASE_URL}/groups"
        payload = {
            "entryType": "BB_RANGE",
            "stockIds": stock_ids
        }
        
        # Log the request
        logger.info(f"🔄 Creating group with {len(stock_ids)} stocks...")
        log_api_request("POST", url, payload=payload)
        
        try:
            start_time = time.time()
            response = requests.put(url, json=payload, timeout=config["api"]["timeout"])
            response_time = time.time() - start_time
            
            # Log the response
            log_api_response(response.status_code, response.text, response_time)
            
            if response.status_code == 201:
                logger.info(f"✅ Successfully created group with {len(stock_ids)} stocks")
                return True
            else:
                logger.error(f"❌ Failed to create group with stocks: {response.status_code} - {response.text}")
                return False
        except Exception as e:
            logger.error(f"❌ Error creating group with stocks: {e}")
            return False
    
    def process_stocks(self) -> bool:
        """Main processing function"""
        logger.info("🚀 Starting stock processing...")
        logger.info(f"📊 Configuration:")
        logger.info(f"  - API Base URL: {API_BASE_URL}")
        logger.info(f"  - Entry Type: BB_RANGE")
        logger.info(f"  - NSE Data Path: {NSE_UPSTOX_JSON_PATH}")
        logger.info(f"  - Stocks File: {STOCKS_FILE_PATH}")
        logger.info("=" * 80)
        
        # Load data
        if not self.load_nse_data():
            return False
            
        stocks = self.load_stocks_from_file()
        if not stocks:
            return False
        
        logger.info(f"📋 Processing {len(stocks)} stocks...")
        logger.info("=" * 80)
        
        # Process each stock
        for i, symbol in enumerate(stocks, 1):
            logger.info(f"🔄 Processing stock {i}/{len(stocks)}: {symbol}")
            
            # Find stock in NSE data
            nse_stock = self.find_stock_in_nse_data(symbol)
            if not nse_stock:
                logger.warning(f"⚠️  Stock {symbol} not found in NSE data")
                self.skipped_stocks.append(symbol)
                continue
            
            logger.info(f"📈 Found stock in NSE data:")
            logger.info(f"  - Symbol: {nse_stock['trading_symbol']}")
            logger.info(f"  - Name: {nse_stock['name']}")
            logger.info(f"  - ISIN: {nse_stock.get('isin', 'N/A')}")
            logger.info(f"  - Instrument Key: {nse_stock.get('instrument_key', 'N/A')}")
            logger.info(f"  - Security ID: {nse_stock['exchange_token']}")
            
            # Create stock via API
            stock_id = self.create_stock_via_api(
                symbol=nse_stock['trading_symbol'],
                name=nse_stock['name'],
                securityId=nse_stock['exchange_token']
            )
            
            if stock_id:
                self.created_stock_ids.append(stock_id)
                logger.info(f"✅ Stock {symbol} processed successfully")
            else:
                self.failed_stocks.append(symbol)
                logger.error(f"❌ Stock {symbol} processing failed")
            
            logger.info("-" * 80)
            
            # Add small delay to avoid overwhelming the API
            time.sleep(config["processing"]["delay_between_requests"])
        
        logger.info("=" * 80)
        logger.info(f"📦 Creating group with {len(self.created_stock_ids)} stocks...")
        
        # Add all created stocks to group
        if self.created_stock_ids:
            success = self.add_stocks_to_group(self.created_stock_ids)
            if not success:
                logger.error("❌ Failed to create group with stocks")
                return False
        else:
            logger.warning("⚠️  No stocks were successfully created")
        
        # Print summary
        self.print_summary()
        return True
    
    def print_summary(self):
        """Print processing summary"""
        logger.info("=" * 80)
        logger.info("📊 PROCESSING SUMMARY")
        logger.info("=" * 80)
        total_processed = len(self.created_stock_ids) + len(self.failed_stocks) + len(self.skipped_stocks)
        logger.info(f"📈 Total stocks processed: {total_processed}")
        logger.info(f"✅ Successfully created: {len(self.created_stock_ids)}")
        logger.info(f"❌ Failed to create: {len(self.failed_stocks)}")
        logger.info(f"⚠️  Skipped (not found in NSE data): {len(self.skipped_stocks)}")
        
        if self.created_stock_ids:
            logger.info(f"🆔 Created stock IDs: {', '.join(self.created_stock_ids[:5])}{'...' if len(self.created_stock_ids) > 5 else ''}")
        
        if self.failed_stocks:
            logger.info(f"❌ Failed stocks: {', '.join(self.failed_stocks)}")
        
        if self.skipped_stocks:
            logger.info(f"⚠️  Skipped stocks: {', '.join(self.skipped_stocks)}")
        
        logger.info("=" * 80)


def main():
    """Main function"""
    logger.info("🚀 Stock Group Management Script Started")
    logger.info(f"📅 Started at: {datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    
    # Validate file paths
    if not Path(NSE_UPSTOX_JSON_PATH).exists():
        logger.error(f"❌ NSE Upstox JSON file not found: {NSE_UPSTOX_JSON_PATH}")
        return False
    
    if not Path(STOCKS_FILE_PATH).exists():
        logger.error(f"❌ Stocks file not found: {STOCKS_FILE_PATH}")
        return False
    
    logger.info(f"✅ All required files found")
    
    # Process stocks
    processor = StockProcessor()
    success = processor.process_stocks()
    
    if success:
        logger.info("🎉 Stock processing completed successfully!")
    else:
        logger.error("💥 Stock processing failed!")
    
    logger.info(f"📅 Completed at: {datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    return success


if __name__ == "__main__":
    main() 