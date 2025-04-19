"""
Morning Range Strategy Backtest Runner

This script tests the Morning Range strategy with different entry types
and compares their performance.
"""

import asyncio
import pandas as pd
import matplotlib.pyplot as plt
import pytz
from datetime import datetime
import logging
import os
import pytest
from pathlib import Path

from mr_strategy.backtest.runner import BacktestRunner, BacktestRunConfig, BacktestMode
from mr_strategy.backtest.engine import BacktestConfig
from mr_strategy.strategy.base_strategy import StrategyConfig
from correlation_analysis.correlation_analyzer import CorrelationAnalyzer
from correlation_analysis.correlation_visualizer import CorrelationVisualizer
from correlation_analysis.stock_clusterer import StockClusterer

print(">> Script Started")

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler("backtest_run.log"),
        logging.StreamHandler()
    ]
)

logger = logging.getLogger(__name__)

# Test parameters
INSTRUMENT_CONFIGSS = [
  {
    "name": "FUSION",
    "key": "NSE_EQ|INE139R01012",
    "direction": "BEARISH"
  },
  {
    "name": "EXICOM",
    "key": "NSE_EQ|INE777F01014",
    "direction": "BEARISH"
  },
  {
    "name": "OLAELEC",
    "key": "NSE_EQ|INE0LXG01040",
    "direction": "BEARISH"
  },
  {
    "name": "FOODSIN",
    "key": "NSE_EQ|INE976E01023",
    "direction": "BEARISH"
  },
  {
    "name": "MEDICO",
    "key": "NSE_EQ|INE630Y01024",
    "direction": "BEARISH"
  },
  {
    "name": "JAICORPLTD",
    "key": "NSE_EQ|INE070D01027",
    "direction": "BEARISH"
  },
  {
    "name": "NAVKARCORP",
    "key": "NSE_EQ|INE278M01019",
    "direction": "BEARISH"
  },
  {
    "name": "NACLIND",
    "key": "NSE_EQ|INE295D01020",
    "direction": "BEARISH"
  },
  {
    "name": "DREAMFOLKS",
    "key": "NSE_EQ|INE0JS101016",
    "direction": "BEARISH"
  },
  {
    "name": "SHAREINDIA",
    "key": "NSE_EQ|INE932X01026",
    "direction": "BEARISH"
  }
]
INSTRUMENT_CONFIGS = [
    {
        'key': 'NSE_EQ|INE070D01027',
        "name": "JAICORP",
        'direction': 'BULLISH'
    }
]
START_DATE = "2025-01-01T09:15:00+05:30"
END_DATE = "2025-04-11T15:25:00+05:30"
INITIAL_CAPITAL = 100000.0

# Entry types to test
ENTRY_TYPES = ["1ST_ENTRY"]

def perform_correlation_analysis(csv_file: str) -> dict:
    """Perform correlation analysis on trade data from CSV file."""
    # Read the CSV file
    df = pd.read_csv(csv_file)
    
    # Create a pivot table of P&L values using instrument names
    pnl_df = df.pivot_table(
        index='Date',
        columns='Name',
        values='P&L',
        aggfunc='sum'
    ).fillna(0)
    
    # Initialize correlation analyzer
    analyzer = CorrelationAnalyzer(pnl_df)
    
    # Calculate different types of correlations
    spearman_corr = analyzer.calculate_spearman_correlation()
    binary_corr = analyzer.calculate_binary_correlation()
    r_multiple_corr = analyzer.calculate_r_multiple_correlation()
    
    # Get significant correlations
    significant_spearman = analyzer.get_significant_correlations(spearman_corr, threshold=0.5)
    significant_binary = analyzer.get_significant_correlations(binary_corr, threshold=0.5)
    significant_r_multiple = analyzer.get_significant_correlations(r_multiple_corr, threshold=0.5)
    
    # Create correlation matrices dictionary
    correlation_matrices = {
        'spearman': spearman_corr,
        'binary': binary_corr,
        'r_multiple': r_multiple_corr
    }
    
    # Initialize visualizer
    visualizer = CorrelationVisualizer(correlation_matrices)
    
    # Create output directory for visualizations
    output_dir = Path("backtest_results/correlation_analysis")
    output_dir.mkdir(parents=True, exist_ok=True)
    
    # Save correlation visualizations
    visualizer.save_visualizations(str(output_dir))
    
    # Perform clustering analysis
    print("\n>> Running Clustering Analysis")
    clusterer = StockClusterer(spearman_corr)
    
    # Find optimal number of clusters
    n_clusters = clusterer.find_optimal_clusters()
    
    # Apply clustering
    clusterer.apply_hierarchical_clustering(n_clusters)
    
    # Visualize clusters
    visualizer.visualize_clusters(clusterer, pnl_df, str(output_dir))
    
    # Get cluster members
    cluster_members = clusterer.get_cluster_members()
    print("\nCluster Members:")
    for cluster_id, stocks in cluster_members.items():
        print(f"Cluster {cluster_id}: {', '.join(stocks)}")
    
    # Calculate and print cluster performance
    performance_df = clusterer.calculate_cluster_performance(pnl_df)
    print("\nCluster Performance Metrics:")
    print(performance_df.to_string(index=False))
    
    # Print significant correlations
    print("\nSignificant Spearman Correlations:")
    print(significant_spearman)
    print("\nSignificant Binary Correlations:")
    print(significant_binary)
    print("\nSignificant R-Multiple Correlations:")
    print(significant_r_multiple)
    
    return {
        'correlation_matrices': correlation_matrices,
        'significant_correlations': {
            'spearman': significant_spearman,
            'binary': significant_binary,
            'r_multiple': significant_r_multiple
        },
        'clustering_results': {
            'n_clusters': n_clusters,
            'cluster_members': cluster_members,
            'performance_metrics': performance_df
        }
    }

async def run_entry_type_comparison(instrument_configs):
    print(">> Running Entry Type Comparison")
    """Run backtest to compare different entry types."""    
    
    # Create runner configuration
    runner_config = BacktestRunConfig(
        mode=BacktestMode.SINGLE,
        start_date=START_DATE,
        end_date=END_DATE,
        instruments=instrument_configs,
        strategies=[{
            "type": "MorningRange",
            "params": {
                "range_type": "5MR",
                "entry_type": entry_type,
                "sl_percentage": 0.75,
                "target_r": 7.0
            }
        } for entry_type in ENTRY_TYPES],
        initial_capital=INITIAL_CAPITAL,
        output_dir="backtest_results"
    )
    
    # Create and run backtest runner
    runner = BacktestRunner(runner_config)
    results = await runner.run_backtests()
    
    # Display results
    print_and_visualize_results(results, runner.reports)
    
    # Perform correlation analysis using the CSV file
    csv_file = Path("backtest_results/daily_trades.csv")
    if csv_file.exists():
        print("\n>> Running Correlation Analysis")
        correlation_results = perform_correlation_analysis(str(csv_file))
        results['correlation_results'] = correlation_results
    else:
        print("\n>> Warning: Daily trades CSV file not found. Skipping correlation analysis.")
    
    print(">> Finished Backtest and Correlation Analysis")
    return results

def save_trade_data_to_csv(trade_list, output_dir="backtest_results"):
    """Save trade data to CSV files in the same format as console output.
    
    Args:
        trade_list (list): List of trade dictionaries
        output_dir (str): Directory to save CSV files
    """
    if not trade_list:
        return
    
    # Create output directory if it doesn't exist
    os.makedirs(output_dir, exist_ok=True)
    
    # Create instrument key to name mapping
    instrument_name_map = {inst['key']: inst['name'] for inst in INSTRUMENT_CONFIGS}
    
    # Process trades by date and instrument
    trade_days = {}
    cumulative_pnl = {}
    for trade in trade_list:
        entry_time = trade.get('current_time')
        if entry_time:
            trade_date = pd.to_datetime(entry_time).strftime('%Y-%m-%d')
            instrument_key = trade.get('instrument_key', 'UNKNOWN')
            instrument_name = instrument_name_map.get(instrument_key, instrument_key)
            pnl = trade.get('realized_pnl', 0)
            direction = trade.get('position_type', 'UNKNOWN')
            trade_type = trade.get('trade_type', 'UNKNOWN')
            max_r_multiple = trade.get('max_r_multiple', 0)
            
            if trade_date not in trade_days:
                trade_days[trade_date] = {}
            
            if instrument_name not in trade_days[trade_date]:
                trade_days[trade_date][instrument_name] = {
                    "pnl": 0,
                    "direction": direction,
                    "trade_type": trade_type,
                    "max_r_multiple": max_r_multiple
                }
            
            trade_days[trade_date][instrument_name]["pnl"] += pnl
            
            # Track cumulative P&L per instrument
            if instrument_name not in cumulative_pnl:
                cumulative_pnl[instrument_name] = 0
            cumulative_pnl[instrument_name] += pnl
    
    # Prepare data for CSV
    csv_data = []
    
    # Add trades by date
    for date in sorted(trade_days.keys()):
        # Add trades for each instrument
        for instrument_name, data in trade_days[date].items():
            pnl = data["pnl"]
            direction = data["direction"]
            trade_type = data["trade_type"]
            status = "PROFIT" if pnl > 0 else "LOSS" if pnl < 0 else "FLAT"
            max_r_multiple = data["max_r_multiple"]
            
            csv_data.append({
                'Date': date,
                'Name': instrument_name,
                'P&L': f"{pnl:.2f}",
                'Status': status,
                'Direction': direction,
                'Trade Type': trade_type,
                'Max R Multiple': f"{max_r_multiple:.2f}",
                'Cumulative': f"{cumulative_pnl[instrument_name]:.2f}"
            })
    
    # Convert to DataFrame and save
    df = pd.DataFrame(csv_data)
    output_file = os.path.join(output_dir, "daily_trades.csv")
    df.to_csv(output_file, index=False)
    print(f"\nDaily trades saved to: {output_file}")

def print_and_visualize_results(results, reports):
    """Print and visualize backtest results."""
    
    print("\n=============================================")
    print("MORNING RANGE STRATEGY BACKTEST RESULTS")
    print("=============================================")
    print(f"Instruments: {[f'{inst['key']} ({inst['direction']})' for inst in INSTRUMENT_CONFIGS]}")
    print(f"Period: {START_DATE} to {END_DATE}")
    print("---------------------------------------------")
    
    # Print summary statistics
    print("\nOVERALL PERFORMANCE:")
    summary = reports['single']['summary']
    
    # Calculate additional metrics
    winning_trades = int(summary['winning_trades'])
    losing_trades = int(summary['losing_trades'])
    avg_profit = summary.get('avg_win', 0)
    avg_loss = summary.get('avg_loss', 0)
    total_profit = winning_trades * avg_profit if avg_profit else 0
    total_loss = losing_trades * avg_loss if avg_loss else 0
    overall_pnl = total_profit + total_loss
    profit_percentage = (avg_profit / INITIAL_CAPITAL * 100) if avg_profit else 0
    loss_percentage = (avg_loss / INITIAL_CAPITAL * 100) if avg_loss else 0
    expectancy = (summary['win_rate'] * avg_profit) + ((1 - summary['win_rate']) * avg_loss)
    
    # Print table format
    print("-" * 60)
    print(f"{'Metric':<25} {'Value':<20}")
    print("-" * 60)
    print(f"{'Total Trades':<25} {summary['total_trades']:<20}")
    print(f"{'Winning Trades':<25} {winning_trades:<20}")
    print(f"{'Losing Trades':<25} {losing_trades:<20}")
    print(f"{'Win Rate':<25} {summary['win_rate']:.2%}")
    print(f"{'Profit Factor':<25} {summary['profit_factor']:.2f}")
    print(f"{'Average Profit':<25} {avg_profit:.2f}")
    print(f"{'Average Loss':<25} {avg_loss:.2f}")
    print(f"{'Profit %':<25} {profit_percentage*100:.2f}%")
    print(f"{'Loss %':<25} {loss_percentage*100:.2f}%")
    print(f"{'Expectancy':<25} {expectancy:.2f}")
    print(f"{'Total Profit':<25} {total_profit:.2f}")
    print(f"{'Total Loss':<25} {total_loss:.2f}")
    print(f"{'Overall PNL':<25} {overall_pnl:.2f}")
    print(f"{'Total Return':<25} {summary['total_return']:.2f}")
    print("-" * 60)
    
    # Print daily P&L information
    trade_list = results.get('trade_list', []) or results.get('trades', [])
    if trade_list:
        # Save trade data to CSV files
        save_trade_data_to_csv(trade_list)
        
        print("\nDAILY PROFIT/LOSS BREAKDOWN:")
        print("-" * 100)
        print(f"{'Date':<12} {'Name':<15} {'P&L':<12} {'Status':<10} {'Direction':<15} {'Trade Type':<12} {'Cumulative':<12}")
        print("-" * 100)
        
        # Process trades by date and instrument
        trade_days = {}
        cumulative_pnl = {}
        for trade in trade_list:
            entry_time = trade.get('current_time')
            if entry_time:
                trade_date = pd.to_datetime(entry_time).strftime('%Y-%m-%d')
                instrument_key = trade.get('instrument_key', 'UNKNOWN')
                pnl = trade.get('realized_pnl', 0)
                direction = trade.get('position_type', 'UNKNOWN')
                trade_type = trade.get('trade_type', 'UNKNOWN')
                
                if trade_date not in trade_days:
                    trade_days[trade_date] = {}
                
                if instrument_key not in trade_days[trade_date]:
                    trade_days[trade_date][instrument_key] = {
                        "pnl": 0,
                        "direction": direction,
                        "trade_type": trade_type
                    }
                
                trade_days[trade_date][instrument_key]["pnl"] += pnl
                
                # Track cumulative P&L per instrument
                if instrument_key not in cumulative_pnl:
                    cumulative_pnl[instrument_key] = 0
                cumulative_pnl[instrument_key] += pnl
        
        # Sort dates and print
        for date in sorted(trade_days.keys()):
            # Print date header
            print(f"{date:<12} {'':<15} {'':<12} {'':<10} {'':<10} {'':<12}")
            print("-" * 100)
            
            # Print trades for each instrument on this date
            for instrument_key, data in trade_days[date].items():
                pnl = data["pnl"]
                direction = data["direction"]
                trade_type = data["trade_type"]
                status = "PROFIT" if pnl > 0 else "LOSS" if pnl < 0 else "FLAT"
                print(f"{'':<12} {instrument_key:<15} {pnl:<12.2f} {status:<10} {direction:<10} {trade_type:<10} {cumulative_pnl[instrument_key]:<12.2f}")
            
            # Print date total
            date_total = sum(data["pnl"] for data in trade_days[date].values())
            date_status = "PROFIT" if date_total > 0 else "LOSS" if date_total < 0 else "FLAT"
            print("-" * 100)
        
        # Print overall totals
        print("\nOVERALL TOTALS:")
        print("-" * 100)
        for instrument_key in sorted(set(key for day in trade_days.values() for key in day.keys())):
            instrument_total = sum(
                day[instrument_key]["pnl"]
                for day in trade_days.values()
                if instrument_key in day
            )
            instrument_status = "PROFIT" if instrument_total > 0 else "LOSS" if instrument_total < 0 else "FLAT"
            print(f"{'':<12} {instrument_key:<15} {instrument_total:<12.2f} {instrument_status:<10} {'':<10} {cumulative_pnl[instrument_key]:<12.2f}")
        
        # Print grand total
        grand_total = sum(
            sum(data["pnl"] for data in day.values())
            for day in trade_days.values()
        )
        grand_status = "PROFIT" if grand_total > 0 else "LOSS" if grand_total < 0 else "FLAT"
        print("-" * 100)
        print(f"{'':<12} {'GRAND TOTAL':<15} {grand_total:<12.2f} {grand_status:<10} {'':<10} {grand_total:<12.2f}")
        print("-" * 100)

if __name__ == "__main__":
    asyncio.run(run_entry_type_comparison(INSTRUMENT_CONFIGS))
