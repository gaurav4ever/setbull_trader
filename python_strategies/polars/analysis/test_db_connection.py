#!/usr/bin/env python3
"""
Test Database Connection and Symbol Lookup
==========================================

This script tests the database connection and symbol lookup functionality
to help debug the issue with symbols not being found.

Author: Gaurav Sharma - CEO, Setbull Trader
Date: 2025-01-30
"""

import mysql.connector
import pandas as pd
import sys

def test_database_connection():
    """Test basic database connection."""
    try:
        connection = mysql.connector.connect(
            host='127.0.0.1',
            port=3306,
            user='root',
            password='root1234',
            database='setbull_trader',
            autocommit=True
        )
        print("✅ Database connection successful")
        return connection
    except mysql.connector.Error as err:
        print(f"❌ Database connection failed: {err}")
        return None

def test_stock_universe_table(connection):
    """Test stock_universe table."""
    try:
        query = "SELECT COUNT(*) as count FROM stock_universe"
        df = pd.read_sql(query, connection)
        print(f"✅ stock_universe table has {df['count'].iloc[0]} records")
        
        # Check sample records
        query = "SELECT symbol, instrument_key, name FROM stock_universe LIMIT 5"
        df = pd.read_sql(query, connection)
        print("Sample records from stock_universe:")
        print(df)
        
        return True
    except Exception as e:
        print(f"❌ Error testing stock_universe table: {e}")
        return False

def test_stock_candle_data_table(connection):
    """Test stock_candle_data table."""
    try:
        query = "SELECT COUNT(*) as count FROM stock_candle_data WHERE time_interval = '1min'"
        df = pd.read_sql(query, connection)
        print(f"✅ stock_candle_data table has {df['count'].iloc[0]} 1min records")
        
        # Check sample records
        query = """
        SELECT DISTINCT instrument_key, time_interval, COUNT(*) as count
        FROM stock_candle_data 
        WHERE time_interval = '1min'
        GROUP BY instrument_key, time_interval
        LIMIT 5
        """
        df = pd.read_sql(query, connection)
        print("Sample records from stock_candle_data:")
        print(df)
        
        return True
    except Exception as e:
        print(f"❌ Error testing stock_candle_data table: {e}")
        return False

def test_symbol_lookup(connection, symbols):
    """Test symbol lookup functionality."""
    try:
        print(f"\n🔍 Testing symbol lookup for: {symbols}")
        
        # Test 1: Check if symbols exist in stock_universe
        placeholders = ','.join(['%s'] * len(symbols))
        query = f"""
        SELECT symbol, instrument_key, name
        FROM stock_universe
        WHERE symbol IN ({placeholders})
        """
        
        print(f"Query 1: {query}")
        print(f"Params: {symbols}")
        
        df = pd.read_sql(query, connection, params=symbols)
        if df.empty:
            print("❌ No symbols found in stock_universe table")
        else:
            print("✅ Symbols found in stock_universe:")
            print(df)
        
        # Test 2: Check if instruments exist in stock_candle_data
        for symbol in symbols:
            print(f"\n🔍 Checking instrument_key '{symbol}' in stock_candle_data:")
            
            # Exact match
            query = """
            SELECT DISTINCT instrument_key, time_interval, COUNT(*) as count
            FROM stock_candle_data
            WHERE instrument_key = %s AND time_interval = '1min'
            GROUP BY instrument_key, time_interval
            """
            
            df = pd.read_sql(query, connection, params=(symbol,))
            if df.empty:
                print(f"❌ No exact match found for '{symbol}'")
                
                # Try partial match
                query = """
                SELECT DISTINCT instrument_key, time_interval, COUNT(*) as count
                FROM stock_candle_data
                WHERE instrument_key LIKE %s AND time_interval = '1min'
                GROUP BY instrument_key, time_interval
                LIMIT 5
                """
                
                df = pd.read_sql(query, connection, params=(f"%{symbol}%",))
                if df.empty:
                    print(f"❌ No partial match found for '{symbol}'")
                else:
                    print(f"✅ Partial matches found for '{symbol}':")
                    print(df)
            else:
                print(f"✅ Exact match found for '{symbol}':")
                print(df)
        
        return True
    except Exception as e:
        print(f"❌ Error testing symbol lookup: {e}")
        return False

def test_time_intervals(connection):
    """Test what time intervals are available in stock_candle_data."""
    try:
        print("\n🔍 Checking available time intervals in stock_candle_data:")
        
        query = """
        SELECT time_interval, COUNT(*) as count
        FROM stock_candle_data
        GROUP BY time_interval
        ORDER BY count DESC
        """
        
        df = pd.read_sql(query, connection)
        if df.empty:
            print("❌ No data found in stock_candle_data table")
        else:
            print("✅ Available time intervals:")
            print(df)
            
            # Check if we have any data for the symbols we want
            print("\n🔍 Checking if we have any data for our test symbols:")
            test_symbols = ['RELIANCE', 'TCS', 'INFY', 'HDFCBANK', 'ICICIBANK']
            
            for symbol in test_symbols:
                # Get the instrument_key from stock_universe
                query = "SELECT instrument_key FROM stock_universe WHERE symbol = %s"
                df_key = pd.read_sql(query, connection, params=(symbol,))
                
                if not df_key.empty:
                    instrument_key = df_key['instrument_key'].iloc[0]
                    print(f"\nChecking {symbol} (instrument_key: {instrument_key}):")
                    
                    # Check all time intervals for this instrument
                    query = """
                    SELECT time_interval, COUNT(*) as count
                    FROM stock_candle_data
                    WHERE instrument_key = %s
                    GROUP BY time_interval
                    ORDER BY count DESC
                    """
                    
                    df_data = pd.read_sql(query, connection, params=(instrument_key,))
                    if df_data.empty:
                        print(f"  ❌ No data found for {symbol}")
                    else:
                        print(f"  ✅ Data found for {symbol}:")
                        print(f"    {df_data}")
        
        return True
    except Exception as e:
        print(f"❌ Error testing time intervals: {e}")
        return False

def main():
    """Main test function."""
    print("Database Connection and Symbol Lookup Test")
    print("=" * 50)
    
    # Test database connection
    connection = test_database_connection()
    if not connection:
        sys.exit(1)
    
    # Test tables
    test_stock_universe_table(connection)
    test_stock_candle_data_table(connection)
    
    # Test symbol lookup
    test_symbols = ['RELIANCE', 'TCS', 'INFY', 'HDFCBANK', 'ICICIBANK']
    test_symbol_lookup(connection, test_symbols)
    
    # Test time intervals
    test_time_intervals(connection)
    
    # Close connection
    connection.close()
    print("\n✅ Test completed")

if __name__ == "__main__":
    main() 