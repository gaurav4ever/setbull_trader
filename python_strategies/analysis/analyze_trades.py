from analysis.intraday_data_analysis import IntradayDataAnalysis
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
        csv_path = os.path.join('/Users/gaurav/setbull_projects/setbull_trader/python_strategies/backtest_results', 'daily_trades.csv')
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
        output_path = "/Users/gaurav/setbull_projects/setbull_trader/python_strategies/backtest_results/strategy_results/backtest_analysis.csv"
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

    except Exception as e:
        print(f"Error during analysis: {str(e)}")
    finally:
        analyzer.close()

if __name__ == "__main__":
    analyze_trades() 