from .intraday_data_analysis import IntradayDataAnalysis
import pandas as pd
import os

def analyze_trades():
    # Initialize the analyzer
    analyzer = IntradayDataAnalysis()
    
    try:
        # Connect to database
        analyzer.connect_db()
        
        # Create tables if they don't exist
        analyzer.create_tables()
        
        # Load data from CSV
        csv_path = os.path.join('/Users/gauravsharma/setbull_projects/setbull_trader/python_strategies/backtest_results', 'daily_trades.csv')
        analyzer.load_data_from_csv(csv_path)
        
        # Update stock analysis
        analyzer.analyze_stock_performance()

        print("\n=== 1st Entry: Top Performing Stocks (min 5 trades) ===")
        entry1st_top = analyzer.get_1st_entry_top_stocks(min_trades=5)
        print(entry1st_top.to_string())

        print("\n=== 2:30 Entry: Top Performing Stocks (min 5 trades) ===")
        entry230_top = analyzer.get_2_30_entry_top_stocks(min_trades=5)
        print(entry230_top.to_string())

        # Export backtest analysis CSV
        output_dir = "/Users/gauravsharma/setbull_projects/setbull_trader/python_strategies/backtest_results/strategy_results"
        os.makedirs(output_dir, exist_ok=True)
        output_path = os.path.join(output_dir, "backtest_analysis.csv")
        analyzer.export_backtest_analysis_csv(output_path)

        # New: Top performing stocks per EntryTimeString
        print("\n=== Top Performing Stocks by EntryTimeString (min 5 trades) ===")
        entry_time_top = analyzer.get_entry_time_top_stocks(min_trades=5)
        for entry_time, df in entry_time_top.items():
            print(f"\n--- EntryTimeString: {entry_time} ---")
            if not df.empty:
                print(df.to_string(index=False))
            else:
                print("No data for this entry time.")

        # NEW: Comprehensive Analysis (from analyze_daily_trades.py)
        print("\n" + "="*80)
        print("COMPREHENSIVE BB WIDTH BACKTESTING ANALYSIS")
        print("="*80)
        
        # Generate and save comprehensive report
        analyzer.save_comprehensive_report()
        
        # Display top performers analysis
        top_performers = analyzer.get_top_performers_analysis()
        
        print("\n1. MOST PROFITABLE STOCKS (by Total Profit Amount):")
        print("-" * 50)
        for idx, row in top_performers['top_profit_amount'].iterrows():
            print(f"{idx+1}. {row['name']:<15} ₹{row['total_pnl']:>10,.2f} ({row['trade_count']} trades, {row['win_rate']}% win rate)")
        
        print("\n2. MOST PROFITABLE STOCKS (by Number of Winning Trades):")
        print("-" * 55)
        for idx, row in top_performers['top_winning_trades'].iterrows():
            print(f"{idx+1}. {row['name']:<15} {row['profitable_trades']:>2} winning trades ({row['trade_count']} total, ₹{row['total_pnl']:>8,.0f} profit)")
        
        print("\n3. MOST TRADED STOCKS (by Number of Trades):")
        print("-" * 45)
        for idx, row in top_performers['top_traded'].iterrows():
            print(f"{idx+1}. {row['name']:<15} {row['trade_count']:>2} trades (₹{row['total_pnl']:>8,.0f} profit, {row['win_rate']}% win rate)")
        
        # Display overall statistics
        stock_stats = analyzer.get_comprehensive_stock_analysis()
        total_pnl = stock_stats['total_pnl'].sum()
        total_trades = stock_stats['trade_count'].sum()
        profitable_trades = stock_stats['profitable_trades'].sum()
        win_rate = (profitable_trades / total_trades * 100) if total_trades > 0 else 0
        
        print(f"\nOVERALL SUMMARY:")
        print(f"Total PnL: ₹{total_pnl:,.2f}")
        print(f"Total Trades: {total_trades}")
        print(f"Win Rate: {win_rate:.1f}%")
        print(f"Best Stock: {top_performers['top_profit_amount'].iloc[0]['name']}")
        print("="*80)

    except Exception as e:
        print(f"Error during analysis: {str(e)}")
    finally:
        analyzer.close()

if __name__ == "__main__":
    analyze_trades() 