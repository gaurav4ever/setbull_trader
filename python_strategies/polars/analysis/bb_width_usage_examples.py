#!/usr/bin/env python3
"""
BB Width Intraday Analysis - Usage Examples
===========================================

This script demonstrates how to use the BB width analyzer with different parameters.

Author: Gaurav Sharma - CEO, Setbull Trader
Date: 2025-01-30
"""

import subprocess
import sys
import os

def run_command(command, description):
    """Run a command and display the description."""
    print(f"\n{'='*60}")
    print(f"EXAMPLE: {description}")
    print(f"{'='*60}")
    print(f"Command: {command}")
    print(f"{'='*60}")
    
    try:
        result = subprocess.run(command, shell=True, capture_output=True, text=True)
        print("STDOUT:")
        print(result.stdout)
        if result.stderr:
            print("STDERR:")
            print(result.stderr)
        print(f"Return Code: {result.returncode}")
    except Exception as e:
        print(f"Error running command: {e}")

def main():
    """Run various usage examples."""
    
    # Get the script directory
    script_dir = os.path.dirname(os.path.abspath(__file__))
    analyzer_script = os.path.join(script_dir, "analyze_bb_width_intraday.py")
    
    print("BB Width Intraday Analysis - Usage Examples")
    print("=" * 60)
    
    # Example 1: Analyze all instruments
    command1 = f"python {analyzer_script} --output-file all_instruments_bb_analysis.csv"
    run_command(command1, "Analyze all instruments in database")
    
    # Example 2: Analyze specific symbols
    command2 = f"python {analyzer_script} --symbols RELIANCE TCS INFY --output-file top_stocks_bb_analysis.csv"
    run_command(command2, "Analyze specific symbols (RELIANCE, TCS, INFY)")
    
    # Example 3: Analyze with lookback period
    command3 = f"python {analyzer_script} --symbols HDFCBANK --lookback-days 30 --output-file hdfc_30days_bb_analysis.csv"
    run_command(command3, "Analyze HDFCBANK with 30-day lookback")
    
    # Example 4: Analyze with custom BB parameters
    command4 = f"python {analyzer_script} --symbols ICICIBANK --bb-period 14 --bb-std 1.5 --output-file icici_custom_bb_analysis.csv"
    run_command(command4, "Analyze ICICIBANK with custom BB parameters (14-period, 1.5 std)")
    
    # Example 5: Generate detailed report
    command5 = f"python {analyzer_script} --symbols RELIANCE TCS --detailed-report --output-file detailed_analysis.csv"
    run_command(command5, "Generate detailed report with all daily statistics")
    
    # Example 6: Analyze with custom market hours
    command6 = f"python {analyzer_script} --symbols WIPRO --market-start 09:30 --market-end 15:00 --output-file wipro_custom_hours.csv"
    run_command(command6, "Analyze WIPRO with custom market hours (9:30 AM to 3:00 PM)")
    
    # Example 7: Verbose analysis
    command7 = f"python {analyzer_script} --symbols TATAMOTORS --verbose --output-file tata_verbose_analysis.csv"
    run_command(command7, "Analyze TATAMOTORS with verbose logging")
    
    # Example 8: Multiple symbols with lookback
    command8 = f"python {analyzer_script} --symbols RELIANCE TCS INFY HDFCBANK ICICIBANK --lookback-days 60 --output-file top5_60days_analysis.csv"
    run_command(command8, "Analyze top 5 stocks with 60-day lookback")
    
    print(f"\n{'='*60}")
    print("USAGE SUMMARY")
    print(f"{'='*60}")
    print("""
Available Command Line Options:
-----------------------------
--symbols SYMBOL1 SYMBOL2 ...    : Analyze specific symbols
--lookback-days DAYS             : Number of days to look back (default: all data)
--bb-period PERIOD               : Bollinger Bands period (default: 20)
--bb-std STD_DEV                 : Bollinger Bands standard deviations (default: 2.0)
--market-start HH:MM             : Market start time (default: 09:15)
--market-end HH:MM               : Market end time (default: 15:30)
--output-file FILENAME           : Output CSV filename (default: bb_width_analysis.csv)
--detailed-report                : Generate detailed report with all daily statistics
--verbose                        : Enable verbose logging

Examples:
---------
1. Analyze all instruments:
   python analyze_bb_width_intraday.py

2. Analyze specific symbols:
   python analyze_bb_width_intraday.py --symbols RELIANCE TCS INFY

3. Analyze with lookback period:
   python analyze_bb_width_intraday.py --symbols HDFCBANK --lookback-days 30

4. Generate detailed report:
   python analyze_bb_width_intraday.py --symbols RELIANCE --detailed-report

5. Custom BB parameters:
   python analyze_bb_width_intraday.py --symbols ICICIBANK --bb-period 14 --bb-std 1.5

Output Files:
-------------
- Main CSV: Contains summary with lowest BB width day for each instrument
- Detailed CSV: Contains all daily statistics (when --detailed-report is used)
- Logs: Detailed execution logs in output/logs/ directory
    """)

if __name__ == "__main__":
    main() 