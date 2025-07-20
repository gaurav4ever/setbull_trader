#!/usr/bin/env python3

import json

def extract_symbols_from_backtest_request(file_path: str) -> list:
    """Extract all symbols from the backtest request JSON file."""
    with open(file_path, 'r') as f:
        data = json.load(f)
    
    symbols = []
    for instrument in data.get('instrument_configs', []):
        symbol = instrument.get('name')
        if symbol:
            symbols.append(symbol)
    
    return symbols

def generate_python_command(symbols: list) -> str:
    """Generate the Python command with all symbols."""
    symbols_str = ' '.join(symbols)
    
    command = f"""python /Users/gauravsharma/setbull_projects/setbull_trader/python_strategies/polars/analysis/analyze_bb_width_intraday.py --symbols {symbols_str} --update-database --lookback-days 30"""
    
    return command

def main():
    # Extract symbols from backtest request
    symbols = extract_symbols_from_backtest_request('maintenance/backtest_request.json')
    
    print(f"Found {len(symbols)} symbols in backtest_request.json")
    print("\nFirst 10 symbols:")
    for i, symbol in enumerate(symbols[:10]):
        print(f"  {i+1}. {symbol}")
    
    if len(symbols) > 10:
        print(f"  ... and {len(symbols) - 10} more symbols")
    
    # Generate the Python command
    command = generate_python_command(symbols)
    
    print(f"\n{'='*80}")
    print("GENERATED PYTHON COMMAND:")
    print(f"{'='*80}")
    print(command)
    print(f"{'='*80}")
    
    # Save the command to a file for easy copying
    with open('maintenance/bb_width_command.txt', 'w') as f:
        f.write(command)
    
    print(f"\nCommand saved to: maintenance/bb_width_command.txt")
    print("You can copy and paste the command above to run the analysis for all symbols.")

if __name__ == "__main__":
    main() 