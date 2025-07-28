#!/usr/bin/env python3
"""
Convert bb_width_backtest_data.txt to SQL INSERT statements for trades table
"""

import csv
import sys
from datetime import datetime

def convert_to_sql():
    # Read the input file
    input_file = "pkg/database/backups/bb_width_backtest_data.txt"
    output_file = "pkg/database/backups/bb_width_backtest_data.sql"
    
    # Start ID counter (you may want to adjust this based on your existing data)
    
    with open(input_file, 'r') as infile, open(output_file, 'w') as outfile:
        # Write SQL header
        outfile.write("-- BB Width Backtest Data SQL Insert Statements\n")
        outfile.write("-- Generated from bb_width_backtest_data.txt\n")
        outfile.write("-- Date: " + datetime.now().strftime("%Y-%m-%d %H:%M:%S") + "\n\n")
        
        # Read each line and convert to SQL
        for line_num, line in enumerate(infile, 1):
            line = line.strip()
            if not line:
                continue
                
            # Split the CSV line
            parts = line.split(',')
            
            if len(parts) < 27:  # Ensure we have enough columns
                print(f"Warning: Line {line_num} has insufficient columns: {len(parts)}")
                continue
            
            try:
                # Extract the relevant fields for trades table
                # Column mapping based on the data structure:
                # 0: date, 1: name, 2: pnl, 3: status, 4: direction, 5: RETEST_ENTRY, 6: max_r_multiple, 7: cumulative_pnl, 8: opening_type, 9: trend, 15: time
                date = parts[0]
                name = parts[1]
                pnl = float(parts[2])
                status = parts[3]
                direction = parts[4]
                trade_type = parts[14]  # This is the time (e.g., "12:20") - column 15 (0-indexed)
                max_r_multiple = float(parts[6])
                cumulative_pnl = float(parts[7])
                opening_type = parts[8]
                trend = parts[9]
                
                # Create SQL INSERT statement
                sql = f"""INSERT INTO `trades` (`date`,`name`,`pnl`,`status`,`direction`,`trade_type`,`max_r_multiple`,`cumulative_pnl`,`opening_type`,`trend`,`created_at`,`updated_at`) VALUES ('{date}','{name}',{pnl},'{status}','{direction}','{trade_type}',{max_r_multiple},{cumulative_pnl},'{opening_type}','{trend}','{datetime.now().strftime("%Y-%m-%d %H:%M:%S")}','{datetime.now().strftime("%Y-%m-%d %H:%M:%S")}');"""
                
                outfile.write(sql + "\n")
                
            except (ValueError, IndexError) as e:
                print(f"Error processing line {line_num}: {e}")
                print(f"Line content: {line}")
                continue
    
    print(f"Conversion completed. Output written to {output_file}")

if __name__ == "__main__":
    convert_to_sql()
