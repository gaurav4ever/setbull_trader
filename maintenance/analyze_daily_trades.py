#!/usr/bin/env python3

import pandas as pd
import numpy as np
from typing import Dict, List, Tuple

def load_and_analyze_data(file_path: str) -> Dict:
    """Load and analyze the daily trades data."""
    # Load the CSV file
    df = pd.read_csv(file_path)
    
    print(f"Loaded {len(df)} trades from {file_path}")
    print(f"Date range: {df['Date'].min()} to {df['Date'].max()}")
    print(f"Unique stocks: {df['Name'].nunique()}")
    
    # Convert PnL to numeric, handling any non-numeric values
    df['PnL'] = pd.to_numeric(df['PnL'], errors='coerce')
    
    # Basic statistics
    total_pnl = df['PnL'].sum()
    profitable_trades = len(df[df['PnL'] > 0])
    loss_trades = len(df[df['PnL'] < 0])
    flat_trades = len(df[df['PnL'] == 0])
    
    print(f"\nOverall Statistics:")
    print(f"Total PnL: ₹{total_pnl:,.2f}")
    print(f"Profitable trades: {profitable_trades}")
    print(f"Loss trades: {loss_trades}")
    print(f"Flat trades: {flat_trades}")
    print(f"Win rate: {(profitable_trades/len(df)*100):.1f}%")
    
    return df

def analyze_stock_performance(df: pd.DataFrame) -> Dict:
    """Analyze performance by stock."""
    # Group by stock name
    stock_stats = df.groupby('Name').agg({
        'PnL': ['sum', 'count', 'mean'],
        'Status': lambda x: (x == 'PROFIT').sum()
    }).round(2)
    
    # Flatten column names
    stock_stats.columns = ['Total_PnL', 'Trade_Count', 'Avg_PnL', 'Profitable_Trades']
    stock_stats = stock_stats.reset_index()
    
    # Calculate win rate
    stock_stats['Win_Rate'] = (stock_stats['Profitable_Trades'] / stock_stats['Trade_Count'] * 100).round(1)
    
    return stock_stats

def get_top_performers(stock_stats: pd.DataFrame) -> Dict:
    """Get top performers in different categories."""
    results = {}
    
    # 1. Most profitable stocks by total profit amount
    top_profit = stock_stats.nlargest(5, 'Total_PnL')[['Name', 'Total_PnL', 'Trade_Count', 'Win_Rate']]
    results['top_profit_amount'] = top_profit
    
    # 2. Most profitable stocks by number of profitable trades
    top_winning_trades = stock_stats.nlargest(5, 'Profitable_Trades')[['Name', 'Profitable_Trades', 'Trade_Count', 'Total_PnL', 'Win_Rate']]
    results['top_winning_trades'] = top_winning_trades
    
    # 3. Most traded stocks by number of trades
    top_traded = stock_stats.nlargest(5, 'Trade_Count')[['Name', 'Trade_Count', 'Total_PnL', 'Profitable_Trades', 'Win_Rate']]
    results['top_traded'] = top_traded
    
    return results

def generate_analysis_report(df: pd.DataFrame, stock_stats: pd.DataFrame, top_performers: Dict) -> str:
    """Generate a comprehensive analysis report."""
    report = []
    report.append("=" * 80)
    report.append("BB WIDTH BACKTESTING RESULTS ANALYSIS")
    report.append("=" * 80)
    report.append("")
    
    # Overall Performance Summary
    total_pnl = df['PnL'].sum()
    profitable_trades = len(df[df['PnL'] > 0])
    total_trades = len(df)
    win_rate = (profitable_trades / total_trades * 100)
    
    report.append("OVERALL PERFORMANCE SUMMARY")
    report.append("-" * 40)
    report.append(f"Total Trades: {total_trades}")
    report.append(f"Profitable Trades: {profitable_trades}")
    report.append(f"Loss Trades: {len(df[df['PnL'] < 0])}")
    report.append(f"Flat Trades: {len(df[df['PnL'] == 0])}")
    report.append(f"Win Rate: {win_rate:.1f}%")
    report.append(f"Total PnL: ₹{total_pnl:,.2f}")
    report.append(f"Average PnL per Trade: ₹{df['PnL'].mean():,.2f}")
    report.append("")
    
    # 1. Most Profitable Stocks by Profit Amount
    report.append("1. MOST PROFITABLE STOCKS (by Total Profit Amount)")
    report.append("-" * 50)
    for idx, row in top_performers['top_profit_amount'].iterrows():
        report.append(f"{idx+1}. {row['Name']:<15} ₹{row['Total_PnL']:>10,.2f} ({row['Trade_Count']} trades, {row['Win_Rate']}% win rate)")
    report.append("")
    
    # 2. Most Profitable Stocks by Number of Winning Trades
    report.append("2. MOST PROFITABLE STOCKS (by Number of Winning Trades)")
    report.append("-" * 55)
    for idx, row in top_performers['top_winning_trades'].iterrows():
        report.append(f"{idx+1}. {row['Name']:<15} {row['Profitable_Trades']:>2} winning trades ({row['Trade_Count']} total, ₹{row['Total_PnL']:>8,.0f} profit)")
    report.append("")
    
    # 3. Most Traded Stocks
    report.append("3. MOST TRADED STOCKS (by Number of Trades)")
    report.append("-" * 45)
    for idx, row in top_performers['top_traded'].iterrows():
        report.append(f"{idx+1}. {row['Name']:<15} {row['Trade_Count']:>2} trades (₹{row['Total_PnL']:>8,.0f} profit, {row['Win_Rate']}% win rate)")
    report.append("")
    
    # Additional Insights
    report.append("ADDITIONAL INSIGHTS")
    report.append("-" * 20)
    
    # Best performing stock overall
    best_stock = stock_stats.loc[stock_stats['Total_PnL'].idxmax()]
    report.append(f"Best Overall Performer: {best_stock['Name']} (₹{best_stock['Total_PnL']:,.2f} profit)")
    
    # Stock with highest win rate (minimum 3 trades)
    high_volume_stocks = stock_stats[stock_stats['Trade_Count'] >= 3]
    if len(high_volume_stocks) > 0:
        best_win_rate = high_volume_stocks.loc[high_volume_stocks['Win_Rate'].idxmax()]
        report.append(f"Highest Win Rate (3+ trades): {best_win_rate['Name']} ({best_win_rate['Win_Rate']}% win rate)")
    
    # Most consistent performer (lowest standard deviation of PnL)
    stock_pnl_std = df.groupby('Name')['PnL'].std().sort_values()
    if len(stock_pnl_std) > 0:
        most_consistent = stock_pnl_std.index[0]
        report.append(f"Most Consistent Performer: {most_consistent} (lowest PnL volatility)")
    
    # Risk analysis
    avg_loss = df[df['PnL'] < 0]['PnL'].mean()
    avg_profit = df[df['PnL'] > 0]['PnL'].mean()
    if not pd.isna(avg_loss) and not pd.isna(avg_profit):
        risk_reward_ratio = abs(avg_profit / avg_loss)
        report.append(f"Risk-Reward Ratio: {risk_reward_ratio:.2f} (Avg Profit: ₹{avg_profit:.0f}, Avg Loss: ₹{avg_loss:.0f})")
    
    report.append("")
    report.append("=" * 80)
    
    return "\n".join(report)

def main():
    # Load and analyze data
    df = load_and_analyze_data('python_strategies/backtest_results/daily_trades.csv')
    
    # Analyze stock performance
    stock_stats = analyze_stock_performance(df)
    
    # Get top performers
    top_performers = get_top_performers(stock_stats)
    
    # Generate report
    report = generate_analysis_report(df, stock_stats, top_performers)
    
    # Save report
    with open('kb/7.2.1_bb_width_backtesting_results.txt', 'w') as f:
        f.write(report)
    
    print("Analysis completed and saved to kb/7.2.1_bb_width_backtesting_results.txt")
    print("\n" + "="*80)
    print("QUICK SUMMARY:")
    print(f"Total PnL: ₹{df['PnL'].sum():,.2f}")
    print(f"Win Rate: {(len(df[df['PnL'] > 0])/len(df)*100):.1f}%")
    print(f"Best Stock: {stock_stats.loc[stock_stats['Total_PnL'].idxmax(), 'Name']}")
    print("="*80)

if __name__ == "__main__":
    main() 