import pandas as pd
from datetime import datetime, timedelta
from typing import List, Dict, Tuple

def calculate_success_percentage(df: pd.DataFrame, entries: int) -> Dict[str, float]:
    stock_performance = {}

    for stock in df['Name'].unique():
        stock_df = df[df['Name'] == stock].sort_values('Date', ascending=True)

        last_n_dates = stock_df['Date'].drop_duplicates().tail(entries)

        if len(last_n_dates) == 0:
            continue

        filtered_df = stock_df[stock_df['Date'].isin(last_n_dates)]

        profitable_days = 0
        for date in last_n_dates:
            day_trades = filtered_df[filtered_df['Date'] == date]
            if any(day_trades['P&L'] > 0):
                profitable_days += 1

        success_rate = (profitable_days / len(last_n_dates)) * 100
        stock_performance[stock] = success_rate

    return stock_performance

def generate_report(df: pd.DataFrame, direction: str, entries: List[int], file):
    """
    Generate a performance report for a specific direction (LONG/SHORT)
    """
    # Filter trades by direction
    direction_trades = df[df['Direction'] == direction]
    
    file.write(f"\n\n{direction} DIRECTION PERFORMANCE REPORT")
    file.write("\n" + "=" * 50 + "\n")
    
    results = {}
    for n in entries:
        success_rates = calculate_success_percentage(direction_trades, n)
        # Sort stocks by success rate in descending order
        sorted_stocks = sorted(success_rates.items(), key=lambda x: x[1], reverse=True)
        results[f'Last {n} Entries'] = sorted_stocks
    
    for period, stocks in results.items():
        file.write(f"\n{period} Success Rate Ranking:")
        file.write("\n" + "-" * 30 + "\n")
        file.write(f"{'Stock':<15} {'Success Rate (%)':<15}\n")
        file.write("-" * 30 + "\n")
        for stock, rate in stocks:
            file.write(f"{stock:<15} {rate:.2f}%\n")
        if stocks:
            top_stock = stocks[0]
            file.write(f"\n🏆 Top Performer in {period}: {top_stock[0]} with {top_stock[1]:.2f}% success rate\n")

def main():
    # Read the CSV file
    df = pd.read_csv('trading_data.csv')
    
    # Convert Date column to datetime
    df['Date'] = pd.to_datetime(df['Date'])
    
    # Define time periods
    entries = [5, 10, 15]
    
    # Generate timestamp for unique filename
    timestamp = datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
    filename = f"stock_performance_report_{timestamp}.txt"
    
    with open(filename, "w") as f:
        f.write("STOCK PERFORMANCE ANALYSIS REPORT")
        f.write("\n" + "=" * 50 + "\n")
        
        # Generate LONG direction report
        generate_report(df, "LONG", entries, f)
        
        # Add separator
        f.write("\n\n" + "=" * 80 + "\n\n")
        
        # Generate SHORT direction report
        generate_report(df, "SHORT", entries, f)
    
    print(f"\n✅ Reports have been saved to: {filename}")

if __name__ == "__main__":
    main() 