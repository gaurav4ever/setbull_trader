"""
Range Strategy Backtest Runner

This script tests the Range strategy with different entry types
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
    "key": "NSE_EQ|INE188A01015",
    "name": "FACT",
    "direction": "BULLISH"
  },
  {
    "key": "NSE_EQ|INE027A01015",
    "name": "RCF",
    "direction": "BULLISH"
  },
  {
    "key": "NSE_EQ|INE503A01015",
    "name": "DCBBANK",
    "direction": "BULLISH"
  },
  {
    "key": "NSE_EQ|INE510A01028",
    "name": "ENGINERSIN",
    "direction": "BULLISH"
  },
  {
    "name": "PARADEEP",
    "key": "NSE_EQ|INE088F01024",
    "direction": "BULLISH"
  },
  {
    "name": "GPTINFRA",
    "key": "NSE_EQ|INE390G01014",
    "direction": "BULLISH"
  },
  {
    "name": "BALUFORGE",
    "key": "NSE_EQ|INE011E01029",
    "direction": "BULLISH"
  },
  {
    "name": "YATHARTH",
    "key": "NSE_EQ|INE0JO301016",
    "direction": "BULLISH"
  },
  {
    "name": "AVANTIFEED",
    "key": "NSE_EQ|INE871C01038",
    "direction": "BULLISH"
  },
  {
    "name": "SUPRIYA",
    "key": "NSE_EQ|INE07RO01027",
    "direction": "BULLISH"
  },
  {
    "name": "GRMOVER",
    "key": "NSE_EQ|INE192H01020",
    "direction": "BULLISH"
  },
  {
    "name": "TDPOWERSYS",
    "key": "NSE_EQ|INE419M01027",
    "direction": "BULLISH"
  },
  {
    "name": "AVALON",
    "key": "NSE_EQ|INE0LCL01028",
    "direction": "BULLISH"
  },
  {
    "name": "NACLIND",
    "key": "NSE_EQ|INE295D01020",
    "direction": "BULLISH"
  },
  {
    "name": "DBREALTY",
    "key": "NSE_EQ|INE879I01012",
    "direction": "BULLISH"
  },
  {
    "name": "POONAWALLA",
    "key": "NSE_EQ|INE511C01022",
    "direction": "BULLISH"
  }
]
START_DATE = "2025-03-01T09:15:00+05:30"
END_DATE = "2025-05-02T15:25:00+05:30"
INITIAL_CAPITAL = 100000.0

# Entry types to test
ENTRY_TYPES = ["1ST_ENTRY", "2_30_ENTRY"]

async def run_entry_type_comparison(instrument_configs, runner_config):
    print(">> Running Entry Type Comparison")
    """Run backtest to compare different entry types."""    
    
    # Create and run backtest runner
    # convert runner_config to BacktestRunConfig object
    runner = BacktestRunner(runner_config)
    results = await runner.run_backtests()
    
    # Display results
    print_and_visualize_results(results, runner.reports, instrument_configs)
    trades = results.get('trades')
    # create a dataframe with the above values
    df = pd.DataFrame(trades)
    # create a series of PNL values for each instrument
    pnl_series = df.groupby('instrument_key')['realized_pnl'].apply(list)
    # print the pnl series
    print("PNL Series: ", pnl_series)


    print(">> Finished Backtest")
    results['pnl_series'] = pnl_series
    return results

def save_trade_data_to_csv(trade_list, instrument_configs, output_dir="backtest_results"):
    """Save trade data to CSV files and update existing rows if duplicate (by Date, Name, Direction).
    
    Args:
        trade_list (list): List of trade dictionaries
        output_dir (str): Directory to save CSV files
    """
    if not trade_list:
        return

    os.makedirs(output_dir, exist_ok=True)

    instrument_name_map = {inst['key']: inst['name'] for inst in instrument_configs}
    trade_days = {}
    cumulative_pnl = {}
    new_csv_data = []

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
            opening_type = trade.get('opening_type', 'UNKNOWN')
            trend = trade.get('trend', "UNKNOWN")
            gap_up = trade.get('gap_up', False)
            gap_down = trade.get('gap_down', False)
            prev_day_buying_indication = trade.get('prev_day_buying_indication', False)
            prev_day_selling_indication = trade.get('prev_day_selling_indication', False)
            entry_type = trade.get('entry_type', 'UNKNOWN')
            entry_time_string = trade.get('entry_time_string', 'UNKNOWN')
            entry_price = trade.get('entry_price', 0)
            exit_price = trade.get('exit_price', 0)
            initial_position_size = trade.get('initial_position_size', 0)
            stop_loss = trade.get('stop_loss', 0)
            breakeven_level = trade.get('breakeven_level', 0)
            breakout_even_to_cost = trade.get('breakout_even_to_cost', 0)
            risk_amount = trade.get('risk_amount', 0)
            duration = trade.get('duration', 0)
            max_r_multiple = trade.get('max_r_multiple', 0)
            mr_value = trade.get('mr_value', 0)
            exit_reason = trade.get('exit_reason', 'UNKNOWN')

            if trade_date not in trade_days:
                trade_days[trade_date] = {}

            if instrument_name not in trade_days[trade_date]:
                trade_days[trade_date][instrument_name] = {
                    "pnl": 0,
                    "direction": direction,
                    "trade_type": trade_type
                }
            trade_days[trade_date][instrument_name]["pnl"] += pnl

            if instrument_name not in cumulative_pnl:
                cumulative_pnl[instrument_name] = 0
            cumulative_pnl[instrument_name] += pnl

            new_csv_data.append({
                'Date': trade_date,
                'Name': instrument_name,
                'PnL': f"{pnl:.2f}",
                'Status': "PROFIT" if pnl > 0 else "LOSS" if pnl < 0 else "FLAT",
                'Direction': direction,
                'TradeType': trade_type,
                'RMultiple': f"{max_r_multiple:.2f}",
                'Cumulative': f"{cumulative_pnl[instrument_name]:.2f}",
                'OpeningType': opening_type,
                'Trend': trend,
                'GapUp': gap_up,
                'GapDown': gap_down,
                'PrevDayBuyingIndication': prev_day_buying_indication,
                'PrevDaySellingIndication': prev_day_selling_indication,
                'EntryType': entry_type,
                'EntryTimeString': entry_time_string,
                'EntryPrice': entry_price,
                'ExitPrice': exit_price,
                'InitialPositionSize': initial_position_size,
                'StopLoss': stop_loss,
                'BreakevenLevel': breakeven_level,
                'BreakoutEvenToCost': breakout_even_to_cost,
                'RiskAmount': risk_amount,
                'Duration': duration,
                'MaxRMultiple': max_r_multiple,
                'MRValue': mr_value,
                'ExitReason': exit_reason
            })

    new_df = pd.DataFrame(new_csv_data)
    output_file = os.path.join(output_dir, "daily_trades.csv")

    # Load existing CSV if available
    if os.path.exists(output_file):
        existing_df = pd.read_csv(output_file)
        # Drop duplicates based on Date, Name, Direction, and keep new version
        combined_df = pd.concat([existing_df, new_df])
        combined_df.drop_duplicates(subset=['Date', 'Name', 'Direction', 'EntryType'], keep='last', inplace=True)
    else:
        combined_df = new_df

    combined_df.sort_values(by=['Date', 'Name'], inplace=True)
    combined_df.to_csv(output_file, index=False)
    print(f"\n✅ Daily trades saved to: {output_file}")

def print_and_visualize_results(results, reports, instrument_configs):
    """Print and visualize backtest results."""
    
    print("\n=============================================")
    print("RANGE STRATEGY BACKTEST RESULTS")
    print("=============================================")
    print(f"Instruments: {[f'{inst['name']} ({inst['direction']})' for inst in instrument_configs]}")
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
        save_trade_data_to_csv(trade_list, instrument_configs)
        
        print("\nDAILY PROFIT/LOSS BREAKDOWN:")
        print("-" * 100)
        print(f"{'Date':<12} {'Name':<15} {'P&L':<12} {'Status':<10} {'Direction':<15} {'Trade Type':<12} {'Cumulative':<12}")
        print("-" * 100)
        
        # Process trades by date and instrument
        trade_days = {}
        cumulative_pnl = {}
        for trade in trade_list:
            entry_time = trade.get('entry_time_string')
            current_time = trade.get('current_time')
            if entry_time:
                trade_date_v1 = pd.to_datetime(entry_time).strftime('%Y-%m-%d %H:%M')
                trade_date_v2 = pd.to_datetime(current_time).strftime('%Y-%m-%d')
                trade_date = trade_date_v1 + " " + trade_date_v2
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
    # Create runner configuration
    runner_config = BacktestRunConfig(
        mode=BacktestMode.SINGLE,
        start_date=START_DATE,
        end_date=END_DATE,
        instruments=INSTRUMENT_CONFIGSS,
        strategies=[{
              "type": "Range",
              "params": {
                "range_type": "5MR",
                "entry_type": "1ST_ENTRY",
                "sl_percentage": 0.5,
                "target_r": 7.0
            }
        }],
        initial_capital=INITIAL_CAPITAL,
        output_dir="backtest_results"
    )
    asyncio.run(run_entry_type_comparison(INSTRUMENT_CONFIGSS, runner_config))
