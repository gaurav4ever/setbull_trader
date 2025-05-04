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

        # Perform 1st entry analysis and present month on month basis, for each month, present top 10 stocks, logic should be like get_1st_entry_top_stocks
        print("\n=== 1st Entry: Top Performing Stocks (min 5 trades) by month ===")
        entry1st_top = analyzer.get_monthly_1st_entry_top_stocks(min_trades=5)
        print(entry1st_top.to_string())

        # 1st entry: month-on-month, trend filtered (Bullish->LONG, Bearish->SHORT)
        print("\n=== 1st Entry: Top 10 Stocks by Month (Trend Filtered) ===")
        entry1st_trend = entry1st_top.copy()
        # Only keep rows where (trend == 'Bullish' and direction == 'LONG') or (trend == 'Bearish' and direction == 'SHORT')
        if 'trend' in entry1st_trend.columns:
            entry1st_trend = entry1st_trend[((entry1st_trend['trend'] == 'Bullish') & (entry1st_trend['direction'] == 'LONG')) |
                                            ((entry1st_trend['trend'] == 'Bearish') & (entry1st_trend['direction'] == 'SHORT'))]
        print(entry1st_trend.to_string())

        print("\n=== 2:30 Entry: Top Performing Stocks (min 5 trades) ===")
        entry230_top = analyzer.get_2_30_entry_top_stocks(min_trades=5)
        print(entry230_top.to_string())

        # Perform 2nd entry analysis and present month on month basis, for each month, present top 10 stocks, logic should be like get_2_30_entry_top_stocks
        print("\n=== 2:30 Entry: Top Performing Stocks (min 5 trades) by month ===")
        entry230_top = analyzer.get_monthly_2_30_entry_top_stocks(min_trades=5)
        print(entry230_top.to_string())

        # 2:30 entry: month-on-month, trend filtered (Bullish->LONG, Bearish->SHORT)
        print("\n=== 2:30 Entry: Top 10 Stocks by Month (Trend Filtered) ===")
        entry230_trend = entry230_top.copy()
        if 'trend' in entry230_trend.columns:
            entry230_trend = entry230_trend[((entry230_trend['trend'] == 'Bullish') & (entry230_trend['direction'] == 'LONG')) |
                                            ((entry230_trend['trend'] == 'Bearish') & (entry230_trend['direction'] == 'SHORT'))]
        print(entry230_trend.to_string())

        
        
    except Exception as e:
        print(f"Error during analysis: {str(e)}")
    finally:
        analyzer.close()

if __name__ == "__main__":
    analyze_trades() 