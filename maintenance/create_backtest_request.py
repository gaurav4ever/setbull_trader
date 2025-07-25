#!/usr/bin/env python3

import json
import mysql.connector
from typing import List, Dict

def get_stock_names_by_instrument_keys(instrument_keys: List[str]) -> List[Dict]:
    """Query database to get stock names and symbols for the given instrument keys."""
    # Database connection
    connection = mysql.connector.connect(
        host='127.0.0.1',
        port=3306,
        user='root',
        password='root1234',
        database='setbull_trader'
    )
    
    cursor = connection.cursor()
    
    # Create placeholders for the IN clause
    placeholders = ','.join(['%s'] * len(instrument_keys))
    
    # Query to get instrument keys, symbols, and names
    query = f"""
    SELECT instrument_key, symbol, name
    FROM stock_universe 
    WHERE instrument_key IN ({placeholders})
    """
    
    cursor.execute(query, instrument_keys)
    results = cursor.fetchall()
    
    # Extract results
    stocks = []
    for row in results:
        if row[0]:  # instrument_key
            stocks.append({
                "key": row[0],
                "symbol": row[1],
                "name": row[2]
            })
    
    cursor.close()
    connection.close()
    
    return stocks

def create_backtest_request(stocks: List[Dict]) -> Dict:
    """Create the backtesting request with the given stocks."""
    backtest_request = {
        "runner_config": {
            "mode": "SINGLE",
            "start_date": "2025-07-01T09:15:00+05:30",
            "end_date": "2025-07-18T15:25:00+05:30",
            "strategies": [
                {
                    "type": "Range",
                    "params": {
                        "range_type": "5MR",
                        "entry_type": "BB_WIDTH_ENTRY",
                        "entry_candle": "9:15",
                        "sl_percentage": 0.5,
                        "target_r": 4.0
                    }
                }
            ],
            "initial_capital": 1000000.0,
            "output_dir": "backtest_results"
        },
        "instrument_configs": []
    }
    
    # Add each stock to the instrument configs
    for stock in stocks:
        backtest_request["instrument_configs"].append({
            "key": stock["key"],
            "name": stock["symbol"],  # Using symbol as name for consistency
            "direction": "BULLISH"
        })
    
    return backtest_request

def main():
    # Read instrument keys from batch_request.json
    with open('maintenance/batch_request.json', 'r') as f:
        batch_data = json.load(f)
    
    instrument_keys = batch_data["instrumentKeys"]
    print(f"Found {len(instrument_keys)} instrument keys in batch_request.json")
    
    # Get stock names from database
    stocks = get_stock_names_by_instrument_keys(instrument_keys)
    print(f"Retrieved {len(stocks)} stocks from database")
    
    # Create backtest request
    backtest_request = create_backtest_request(stocks)
    
    # Save the backtest request to a file
    output_file = 'maintenance/backtest_request.json'
    with open(output_file, 'w') as f:
        json.dump(backtest_request, f, indent=2)
    
    print(f"Backtest request saved to {output_file}")
    print(f"Total instruments in backtest: {len(backtest_request['instrument_configs'])}")
    
    # Print first few stocks as example
    print("\nFirst 5 stocks in the backtest:")
    for i, stock in enumerate(backtest_request['instrument_configs'][:5]):
        print(f"  {i+1}. {stock['name']} ({stock['key']})")

if __name__ == "__main__":
    main() 