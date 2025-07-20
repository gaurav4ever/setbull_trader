#!/usr/bin/env python3
import re
import mysql.connector
import json
from typing import List

def extract_stock_symbols(file_path: str) -> List[str]:
    """Extract stock symbols from the file."""
    with open(file_path, 'r') as f:
        content = f.read()
    
    # Extract all NSE: symbols using regex
    symbols = re.findall(r'NSE:([A-Z0-9._]+)', content)
    return symbols

def get_instrument_keys(symbols: List[str]) -> List[str]:
    """Query database to get instrument keys for the given symbols."""
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
    placeholders = ','.join(['%s'] * len(symbols))
    
    # Query to get instrument keys
    query = f"""
    SELECT instrument_key 
    FROM stock_universe 
    WHERE symbol IN ({placeholders})
    """
    
    cursor.execute(query, symbols)
    results = cursor.fetchall()
    
    # Extract instrument keys
    instrument_keys = [row[0] for row in results if row[0]]
    
    cursor.close()
    connection.close()
    
    return instrument_keys

def create_request_json(instrument_keys: List[str]) -> str:
    """Create the request JSON with instrument keys."""
    request_data = {
        "instrumentKeys": instrument_keys,
        "fromDate": "2025-07-01",
        "toDate": "2025-07-18",
        "interval": "1minute"
    }
    
    return json.dumps(request_data, indent=2)

def main():
    # Extract symbols from file
    symbols = extract_stock_symbols('maintenance/stocks_20thjul.txt')
    print(f"Found {len(symbols)} stock symbols:")
    for symbol in symbols:
        print(f"  - {symbol}")
    
    print(f"\nQuerying database for instrument keys...")
    
    # Get instrument keys from database
    instrument_keys = get_instrument_keys(symbols)
    print(f"Found {len(instrument_keys)} instrument keys:")
    for key in instrument_keys:
        print(f"  - {key}")
    
    # Create request JSON
    request_json = create_request_json(instrument_keys)
    
    print(f"\nRequest JSON:")
    print(request_json)
    
    # Save to file
    with open('maintenance/batch_request.json', 'w') as f:
        f.write(request_json)
    
    print(f"\nRequest saved to maintenance/batch_request.json")

if __name__ == "__main__":
    main() 
    # python maintenance/get_instrument_keys.py