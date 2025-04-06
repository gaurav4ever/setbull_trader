#!/usr/bin/env python3

import json
import time
import logging
import requests
from typing import List

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler("stock_data_processing.log"),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger("stock_data_processor")

def read_instrument_keys(file_path: str) -> List[str]:
    """Read instrument keys from the file and return as a list."""
    try:
        with open(file_path, 'r') as file:
            content = file.read()
            # Parse the content - remove quotes and split by commas
            keys = [key.strip('"') for key in content.replace('\n', '').split(',') if key.strip()]
            logger.info(f"Successfully read {len(keys)} instrument keys from {file_path}")
            return keys
    except Exception as e:
        logger.error(f"Error reading instrument keys from {file_path}: {e}")
        return []

def process_in_batches(instrument_keys: List[str], batch_size: int = 5, delay: int = 2):
    """Process instrument keys in batches and make API calls."""
    api_url = "http://localhost:8080/api/v1/stocks/universe/daily-candles"
    headers = {'Content-Type': 'application/json'}
    
    total_batches = (len(instrument_keys) + batch_size - 1) // batch_size
    
    for i in range(0, len(instrument_keys), batch_size):
        batch = instrument_keys[i:i+batch_size]
        batch_num = (i // batch_size) + 1
        
        logger.info(f"Processing batch {batch_num}/{total_batches} with {len(batch)} instruments")
        logger.debug(f"Batch content: {batch}")
        
        payload = {
            "days": 100,
            "instrumentKeys": batch
        }
        
        try:
            response = requests.post(api_url, headers=headers, data=json.dumps(payload))
            
            if response.status_code == 200:
                logger.info(f"Batch {batch_num} processed successfully")
            else:
                logger.error(f"Batch {batch_num} failed with status code {response.status_code}: {response.text}")
        
        except Exception as e:
            logger.error(f"Error processing batch {batch_num}: {e}")
        
        # Wait before next batch unless it's the last batch
        if i + batch_size < len(instrument_keys):
            logger.info(f"Waiting {delay} seconds before next batch...")
            time.sleep(delay)

def main():
    """Main function to orchestrate the process."""
    file_path = "/Users/gaurav/setbull_projects/setbull_trader/maintenance/stocks_data_to_fill.txt"
    logger.info("Starting stock data processing")
    
    instrument_keys = read_instrument_keys(file_path)
    
    if instrument_keys:
        logger.info(f"Processing {len(instrument_keys)} instrument keys in batches of 5")
        process_in_batches(instrument_keys)
        logger.info("Stock data processing completed")
    else:
        logger.error("No instrument keys found. Exiting.")

if __name__ == "__main__":
    main()
