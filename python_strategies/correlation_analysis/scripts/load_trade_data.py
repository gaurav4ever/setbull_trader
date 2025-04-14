#!/usr/bin/env python3

"""
Script to load and validate trade data from CSV files.
"""

import logging
import argparse
from pathlib import Path
from python_strategies.correlation_analysis.data_loader import DataLoader

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

def main():
    """Main function to load and validate trade data."""
    parser = argparse.ArgumentParser(description='Load and validate trade data from CSV files')
    parser.add_argument('file_path', type=str, help='Path to the CSV file containing trade data')
    args = parser.parse_args()
    
    try:
        # Initialize data loader
        data_loader = DataLoader(args.file_path)
        
        # Load data
        logger.info(f"Loading data from {args.file_path}")
        df = data_loader.load_csv()
        
        # Validate data
        logger.info("Validating data...")
        if data_loader.validate_data(df):
            logger.info("Data validation successful")
        else:
            logger.warning("Data validation failed")
        
        # Clean data
        logger.info("Cleaning data...")
        cleaned_df = data_loader.clean_data(df)
        
        # Print summary
        logger.info(f"Loaded {len(cleaned_df)} trades")
        logger.info(f"Date range: {cleaned_df['Date'].min()} to {cleaned_df['Date'].max()}")
        logger.info(f"Number of unique stocks: {cleaned_df['Name'].nunique()}")
        logger.info(f"Total P&L: {cleaned_df['P&L'].sum():.2f}")
        
        # Print sample of the data
        logger.info("\nSample of the data:")
        print(cleaned_df.head())
        
    except Exception as e:
        logger.error(f"Error processing data: {str(e)}")
        raise

if __name__ == "__main__":
    main() 