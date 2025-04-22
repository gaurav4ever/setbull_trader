import pandas as pd
from datetime import datetime, timedelta
from typing import List, Dict, Tuple

def calculate_success_percentage(df: pd.DataFrame, entries: int) -> Dict[str, float]:
    """
    Calculate success percentage for each stock based on the last N entries.
    Success percentage = (Number of profitable days / Total number of days) * 100
    """
    # Sort by date and get the last N entries
    sorted_df = df.sort_values('Date')
    period_trades = sorted_df.tail(entries)
    
    # Group by stock and calculate success metrics
    stock_performance = {}
    for stock in period_trades['Name'].unique():
        stock_trades = period_trades[period_trades['Name'] == stock]
        # Get unique dates for this stock
        unique_dates = stock_trades['Date'].unique()
        total_days = len(unique_dates)
        
        # Count profitable days (days with at least one profitable trade)
        profitable_days = 0
        for date in unique_dates:
            day_trades = stock_trades[stock_trades['Date'] == date]
            if any(day_trades['P&L'] > 0):  # If any trade on this day was profitable
                profitable_days += 1
        
        if total_days > 0:
            success_percentage = (profitable_days / total_days) * 100
            stock_performance[stock] = success_percentage
    
    return stock_performance

def main():
    # Read the CSV file
    df = pd.read_csv('trading_data.csv')
    
    # Convert Date column to datetime
    df['Date'] = pd.to_datetime(df['Date'])
    
    # Calculate success percentages for different time periods
    entries = [5, 10, 15]
    results = {}
    
    for n in entries:
        success_rates = calculate_success_percentage(df, n)
        # Sort stocks by success rate in descending order
        sorted_stocks = sorted(success_rates.items(), key=lambda x: x[1], reverse=True)
        results[f'Last {n} Entries'] = sorted_stocks
    
    # Print results
    print("\nStock Performance Analysis")
    print("=" * 50)
    
    for period, stocks in results.items():
        print(f"\n{period} Success Rate Ranking:")
        print("-" * 30)
        print(f"{'Stock':<15} {'Success Rate (%)':<15}")
        print("-" * 30)
        for stock, rate in stocks:
            print(f"{stock:<15} {rate:.2f}%")

if __name__ == "__main__":
    main() 