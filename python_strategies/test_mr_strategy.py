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

from mr_strategy.backtest.runner import BacktestRunner, BacktestRunConfig, BacktestMode
from mr_strategy.backtest.engine import BacktestConfig
from mr_strategy.strategy.base_strategy import StrategyConfig

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
INSTRUMENT_KEY = ["NSE_EQ|INE070D01027", "NSE_EQ|INE139R01012", "NSE_EQ|INE133E01013"]
START_DATE = "2025-04-01T09:15:00+05:30"
END_DATE = "2025-04-11T15:25:00+05:30"
INITIAL_CAPITAL = 100000.0

# Entry types to test
ENTRY_TYPES = ["1ST_ENTRY"]

async def run_entry_type_comparison():
    print(">> Running Entry Type Comparison")
    """Run backtest to compare different entry types."""    
    
    # Create runner configuration
    runner_config = BacktestRunConfig(
        mode=BacktestMode.SINGLE,
        start_date=START_DATE,
        end_date=END_DATE,
        instruments=INSTRUMENT_KEY,
        strategies=[{
            "type": "MorningRange",
            "params": {
                "range_type": "5MR",
                "entry_type": entry_type,
                "sl_percentage": 0.5,
                "target_r": 5.0
            }
        } for entry_type in ENTRY_TYPES],
        initial_capital=INITIAL_CAPITAL,
        output_dir="backtest_results"
    )
    
    # Create and run backtest runner
    runner = BacktestRunner(runner_config)
    results = await runner.run_backtests()
    
    # Display results
    # logger.info(f"results: {results}, runner.reports: {runner.reports}")
    print_and_visualize_results(results, runner.reports)
    
    print(">> Finished Backtest")
    return results

def print_and_visualize_results(results, reports):
    """Print and visualize backtest results."""
    
    print("\n=============================================")
    print("MORNING RANGE STRATEGY BACKTEST RESULTS")
    print("=============================================")
    print(f"Instrument: {INSTRUMENT_KEY}")
    print(f"Period: {START_DATE} to {END_DATE}")
    print("---------------------------------------------")
    
    # Print summary statistics
    print("\nOVERALL PERFORMANCE:")
    summary = reports['single']['summary']
    
    # Calculate additional metrics
    winning_trades = int(summary['total_trades'])
    losing_trades = summary['total_trades'] - winning_trades
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
    print(f"{'Profit %':<25} {profit_percentage:.2f}%")
    print(f"{'Loss %':<25} {loss_percentage:.2f}%")
    print(f"{'Expectancy':<25} {expectancy:.2f}")
    print(f"{'Total Profit':<25} {total_profit:.2f}")
    print(f"{'Total Loss':<25} {total_loss:.2f}")
    print(f"{'Overall PNL':<25} {overall_pnl:.2f}")
    print(f"{'Total Return':<25} {summary['total_return']:.2f}")
    print(f"{'Max Drawdown':<25} {summary['max_drawdown']:.2f}")
    print("-" * 60)
    
    # Extract and compare strategy results
    strategy_results = reports['single']['performance_metrics']
    
    print("\nENTRY TYPE COMPARISON:")
    print("-" * 60)
    print(f"{'Entry Type':<15} {'Total Trades':<12} {'Win Rate':<10} {'Avg Profit':<12} {'Avg Loss':<12}")
    print("-" * 60)
    
    for strategy_id, metrics in strategy_results.items():
        entry_type = strategy_id.split('_')[-1]
        print(f"{entry_type:<15} {metrics['total_trades']:<12} {metrics['win_rate']:<12} "
              f"{metrics['average_win']:<12} {metrics['average_loss']:<12}")
    
    # Visualize equity curves if available
    if 'equity_curve' in results and not results['equity_curve'].empty:
        plt.figure(figsize=(12, 6))
        
        # Group by strategy_id and plot
        for strategy_id in results['equity_curve']['strategy_id'].unique():
            strategy_data = results['equity_curve'][results['equity_curve']['strategy_id'] == strategy_id]
            entry_type = strategy_id.split('_')[-1]
            plt.plot(strategy_data['timestamp'], strategy_data['equity'], label=entry_type)
        
        plt.title('Equity Curve Comparison by Entry Type')
        plt.xlabel('Date')
        plt.ylabel('Equity')
        plt.legend()
        plt.grid(True)
        
        # Save the plot
        output_dir = "backtest_results/plots"
        os.makedirs(output_dir, exist_ok=True)
        plt.savefig(f"{output_dir}/equity_comparison.png")
        plt.close()
        
        print(f"\nEquity curve comparison saved to {output_dir}/equity_comparison.png")
    
    # Print recommendations
    if 'recommendations' in reports['single']:
        print("\nRECOMMENDATIONS:")
        for rec in reports['single']['recommendations']:
            print(f"- {rec}")
    
    # Print daily P&L information
    trade_list = results.get('trade_list', []) or results.get('trades', [])
    if trade_list:
        print("\nDAILY PROFIT/LOSS BREAKDOWN:")
        print("-" * 60)
        print(f"{'Date':<12} {'Profit/Loss':<15} {'Status':<10}")
        print("-" * 60)
        
        # Process trades by date
        trade_days = {}
        for trade in trade_list:
            # Extract date in YYYY-MM-DD format
            entry_time = trade.get('current_time')
            if entry_time:
                trade_date = pd.to_datetime(entry_time).strftime('%Y-%m-%d')
                pnl = trade.get('realized_pnl', 0)
                
                if trade_date not in trade_days:
                    trade_days[trade_date] = 0
                
                trade_days[trade_date] += pnl
        
        # Sort dates and print
        for date in sorted(trade_days.keys()):
            pnl = trade_days[date]
            status = "PROFIT" if pnl > 0 else "LOSS" if pnl < 0 else "FLAT"
            print(f"{date:<12} {pnl:<15.2f} {status:<10}")
            
        print("-" * 60)

if __name__ == "__main__":
    asyncio.run(run_entry_type_comparison())
