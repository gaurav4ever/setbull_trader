import polars as pl
import argparse
import os

def rank_candidates(input_file: str, output_dir: str, max_squeeze_ratio: float, max_volume_ratio: float, 
                   bullish_threshold: float, bearish_threshold: float, top_n: int):
    """
    Reads squeeze candidates from a CSV, filters them for both bullish and bearish setups,
    ranks them, and saves the top candidates to separate files.
    """
    # --- 1. Load Data ---
    try:
        df = pl.read_csv(input_file)
        print(f"Successfully loaded {len(df)} candidates from '{input_file}'.")
    except Exception as e:
        print(f"Error loading input file '{input_file}': {e}")
        return

    # --- 2. Calculate Breakdown Readiness ---
    df = df.with_columns(
        pl.col("breakout_readiness").map_elements(lambda x: 1 - x if x is not None else None).alias("breakdown_readiness")
    )

    # --- 3. Filter and Rank Bullish Candidates ---
    print(f"\n--- Analyzing Bullish Candidates (Breakout Readiness >= {bullish_threshold}) ---")
    bullish_df = df.filter(
        (pl.col("squeeze_ratio") <= max_squeeze_ratio) &
        (pl.col("volume_ratio") <= max_volume_ratio) &
        (pl.col("breakout_readiness") >= bullish_threshold)
    )
    
    bullish_count = len(bullish_df)
    print(f"Found {bullish_count} bullish candidates after filtering.")
    
    if not bullish_df.is_empty():
        # Rank bullish candidates
        bullish_df = bullish_df.with_columns([
            pl.col("squeeze_ratio").rank(method='min').alias("squeeze_rank"),
            pl.col("volume_ratio").rank(method='min').alias("volume_rank"),
            pl.col("breakout_readiness").rank(method='min', descending=True).alias("readiness_rank")
        ]).with_columns(
            (pl.col("squeeze_rank") + pl.col("volume_rank") + pl.col("readiness_rank")).alias("composite_rank")
        )
        
        bullish_ranked = bullish_df.sort("composite_rank").head(top_n)
        
        # Save bullish candidates
        bullish_output = os.path.join(output_dir, "bullish_candidates.csv")
        bullish_ranked.write_csv(bullish_output)
        print(f"Saved top {len(bullish_ranked)} bullish candidates to '{bullish_output}'")
        
        print("\n--- Top Bullish Candidates ---")
        print(bullish_ranked.select([
            "symbol", "composite_rank", "squeeze_ratio", "volume_ratio", 
            "breakout_readiness", "latest_close"
        ]))
    else:
        print("No bullish candidates found matching criteria.")

    # --- 4. Filter and Rank Bearish Candidates ---
    print(f"\n--- Analyzing Bearish Candidates (Breakdown Readiness >= {bearish_threshold}) ---")
    bearish_df = df.filter(
        (pl.col("squeeze_ratio") <= max_squeeze_ratio) &
        (pl.col("volume_ratio") <= max_volume_ratio) &
        (pl.col("breakdown_readiness") >= bearish_threshold)
    )
    
    bearish_count = len(bearish_df)
    print(f"Found {bearish_count} bearish candidates after filtering.")
    
    if not bearish_df.is_empty():
        # Rank bearish candidates
        bearish_df = bearish_df.with_columns([
            pl.col("squeeze_ratio").rank(method='min').alias("squeeze_rank"),
            pl.col("volume_ratio").rank(method='min').alias("volume_rank"),
            pl.col("breakdown_readiness").rank(method='min', descending=True).alias("readiness_rank")
        ]).with_columns(
            (pl.col("squeeze_rank") + pl.col("volume_rank") + pl.col("readiness_rank")).alias("composite_rank")
        )
        
        bearish_ranked = bearish_df.sort("composite_rank").head(top_n)
        
        # Save bearish candidates
        bearish_output = os.path.join(output_dir, "bearish_candidates.csv")
        bearish_ranked.write_csv(bearish_output)
        print(f"Saved top {len(bearish_ranked)} bearish candidates to '{bearish_output}'")
        
        print("\n--- Top Bearish Candidates ---")
        print(bearish_ranked.select([
            "symbol", "composite_rank", "squeeze_ratio", "volume_ratio", 
            "breakdown_readiness", "latest_close"
        ]))
    else:
        print("No bearish candidates found matching criteria.")

    # --- 5. Summary ---
    print(f"\n--- Summary ---")
    print(f"Total candidates analyzed: {len(df)}")
    print(f"Bullish candidates: {bullish_count}")
    print(f"Bearish candidates: {bearish_count}")
    print(f"Output directory: {output_dir}")


def main():
    """Main function to parse arguments and run the ranking script."""
    parser = argparse.ArgumentParser(
        description="Filter and rank volatility squeeze candidates for both bullish and bearish setups.",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter
    )
    
    parser.add_argument("input_file", type=str, help="Path to the input CSV file from the analysis script.")
    parser.add_argument("output_dir", type=str, help="Directory to save the ranked output CSV files.")
    
    # Filtering criteria
    parser.add_argument("--max-squeeze-ratio", type=float, default=0.6, help="Maximum squeeze ratio (lower is tighter).")
    parser.add_argument("--max-volume-ratio", type=float, default=0.8, help="Maximum volume ratio (5-day avg / 50-day avg).")
    parser.add_argument("--bullish-threshold", type=float, default=0.7, help="Minimum breakout readiness for bullish candidates.")
    parser.add_argument("--bearish-threshold", type=float, default=0.7, help="Minimum breakdown readiness for bearish candidates.")
    parser.add_argument("--top-n", type=int, default=5, help="Number of top-ranked candidates to display and save for each direction.")

    args = parser.parse_args()
    
    # Ensure output directory exists - handle both file and directory cases
    try:
        # If output_dir is actually a file, use its directory
        if os.path.isfile(args.output_dir):
            output_dir = os.path.dirname(args.output_dir)
            if not output_dir:
                output_dir = "."  # Use current directory if no directory path
        else:
            output_dir = args.output_dir
            
        os.makedirs(output_dir, exist_ok=True)
    except Exception as e:
        print(f"Error creating output directory: {e}")
        return
    
    rank_candidates(
        input_file=args.input_file,
        output_dir=output_dir,
        max_squeeze_ratio=args.max_squeeze_ratio,
        max_volume_ratio=args.max_volume_ratio,
        bullish_threshold=args.bullish_threshold,
        bearish_threshold=args.bearish_threshold,
        top_n=args.top_n
    )

if __name__ == "__main__":
    main() 