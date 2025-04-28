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
        
        # Perform various analyses
        print("\n=== Opening Condition Analysis ===")
        opening_analysis = analyzer.get_opening_condition_analysis()
        print(opening_analysis.to_string())
        
        print("\n=== Trend Analysis ===")
        trend_analysis = analyzer.get_trend_analysis()
        print(trend_analysis.to_string())
        
        print("\n=== Stocks Performing Against Trend ===")
        against_trend = analyzer.get_stocks_performing_against_trend()
        print(against_trend.to_string())
        
        print("\n=== Stocks Performing With Trend ===")
        with_trend = analyzer.get_stocks_performing_with_trend()
        print(with_trend.to_string())
        
        print("\n=== Market Reversal Analysis ===")
        reversal_analysis = analyzer.get_market_reversal_analysis()
        print(reversal_analysis.to_string())
        
        print("\n=== Pattern Recognition Analysis ===")
        pattern_analysis = analyzer.get_pattern_recognition()
        print(pattern_analysis.to_string())
        
        print("\n=== Mamba Move Analysis ===")
        mamba_analysis = analyzer.get_mamba_move_analysis()
        print(mamba_analysis.to_string())
        
        print("\n=== High Potential Stocks ===")
        high_potential = analyzer.get_high_potential_stocks()
        print(high_potential.to_string())
        
        # Display trade counts by opening type
        print("\n=== Trade Counts by Opening Type ===")
        trade_counts = pd.read_sql("""
            SELECT 
                name,
                direction,
                oah_trade_count,
                oal_trade_count,
                oam_trade_count,
                (oah_trade_count + oal_trade_count + oam_trade_count) as total_trades
            FROM stock_analysis
            ORDER BY total_trades DESC
        """, analyzer.conn)
        print(trade_counts.to_string())
        
    except Exception as e:
        print(f"Error during analysis: {str(e)}")
    finally:
        analyzer.close()

if __name__ == "__main__":
    analyze_trades() 