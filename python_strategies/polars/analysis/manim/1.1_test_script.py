import polars as pl
import numpy as np
import matplotlib.pyplot as plt
from matplotlib.animation import FuncAnimation
import matplotlib.patches as mpatches

# --- Configuration (matches the logic in your script) ---
MAX_SQUEEZE_RATIO = 0.6
MAX_VOLUME_RATIO = 0.8
BULLISH_THRESHOLD = 0.7
BEARISH_THRESHOLD = 0.7 # Still defined, but animation focuses on Bullish for iterative removal
TOP_N = 5
TOTAL_CANDIDATES = 200

# --- Instagram-specific Animation Configuration ---
TARGET_DURATION_SECONDS = 10
FPS = 25 # Frames per second for a smoother look and to hit 10s target
TOTAL_FRAMES = TARGET_DURATION_SECONDS * FPS

# --- Phase Timing for Iterative Removal ---
# Divide total frames into: Initial View, Iterative Filtering, Final Highlight
PHASE_INITIAL_END_FRAME = int(TOTAL_FRAMES * 0.1)  # ~10% for initial overview (25 frames)
PHASE_FILTERING_END_FRAME = int(TOTAL_FRAMES * 0.8) # ~70% for iterative filtering (175 frames)
                                                  # This leaves 250 - 25 - 175 = 50 frames for final highlight
FILTERING_FRAMES_DURATION = PHASE_FILTERING_END_FRAME - PHASE_INITIAL_END_FRAME

def generate_mock_data(num_records: int) -> pl.DataFrame:
    """Generates a Polars DataFrame with random data mimicking your input file."""
    np.random.seed(42)  # for reproducible results
    data = {
        "symbol": [f"SYM{i:03}" for i in range(num_records)],
        "squeeze_ratio": np.random.uniform(0.1, 1.0, num_records),
        "volume_ratio": np.random.uniform(0.2, 1.5, num_records),
        "breakout_readiness": np.random.uniform(0.0, 1.0, num_records),
        "latest_close": np.random.uniform(20, 300, num_records)
    }
    return pl.DataFrame(data)

def animate_ranking_process_instagram_iterative():
    """Creates and saves an animation visualizing the iterative filtering process, optimized for Instagram."""
    
    # --- 1. Data Preparation ---
    df = generate_mock_data(TOTAL_CANDIDATES)
    df = df.with_columns(
        (1 - pl.col("breakout_readiness")).alias("breakdown_readiness")
    )

    # Pre-calculate the boolean mask for stocks that ultimately pass the BULLISH filter
    # This is done once outside the animation loop for efficiency.
    is_ultimately_bullish_mask = (
        (df["squeeze_ratio"] <= MAX_SQUEEZE_RATIO) & 
        (df["volume_ratio"] <= MAX_VOLUME_RATIO) & 
        (df["breakout_readiness"] >= BULLISH_THRESHOLD)
    ).to_numpy()

    # Determine the top N bullish candidates (these are the 'final stocks')
    bullish_df = df.filter(is_ultimately_bullish_mask) # Filter using the pre-calculated mask
    if not bullish_df.is_empty():
        bullish_ranked = bullish_df.with_columns([
            pl.col("squeeze_ratio").rank(method='min').alias("squeeze_rank"),
            pl.col("volume_ratio").rank(method='min').alias("volume_rank"),
            pl.col("breakout_readiness").rank(method='min', descending=True).alias("readiness_rank")
        ]).with_columns(
            (pl.col("squeeze_rank") + pl.col("volume_rank") + pl.col("readiness_rank")).alias("composite_rank")
        ).sort("composite_rank").head(TOP_N)
        top_bullish_symbols = bullish_ranked["symbol"].to_list()
    else:
        top_bullish_symbols = []

    # Calculate sizes once, outside the update function as it's constant
    sizes = (df["breakout_readiness"] * 150).to_numpy() 

    # --- 2. Animation Setup ---
    fig, ax = plt.subplots(figsize=(10, 10), dpi=100) # Instagram square format
    fig.patch.set_facecolor('#1a1a1a') # Dark background for figure
    ax.set_facecolor('#1a1a1a')         # Dark background for axes

    # Initial scatter plot creation - dummy data, will be updated
    scatter = ax.scatter([0], [0], s=[0], c=[0], cmap="viridis", alpha=1.0, edgecolors='w', linewidths=0.5)

    # Text objects for dynamic updates
    criteria_text_obj = ax.text(0.98, 0.65, '', transform=ax.transAxes, fontsize=11,
                                verticalalignment='top', horizontalalignment='right', color='white', 
                                bbox=dict(boxstyle='round,pad=0.5', fc='#333333', ec='none', alpha=0.8))
    title_obj = ax.set_title("", color='white', fontsize=18, pad=20)

    legend = None # Initialize legend as None

    # --- Animation Frames Logic ---
    # Modified update function to accept df and sizes as arguments
    def update(frame, df, sizes): # Added df and sizes to the function signature
        nonlocal legend # Allow modification of legend object from outer scope
        
        ax.clear() # Clear axes for fresh plot in each frame
        ax.set_facecolor('#1a1a1a') # Re-set background color after clearing

        # Plot styling
        ax.set_xlim(0, 1.6)
        ax.set_ylim(0, 1.6)
        ax.spines['top'].set_visible(False)
        ax.spines['right'].set_visible(False)
        ax.spines['bottom'].set_color('white')
        ax.spines['left'].set_color('white')
        ax.tick_params(axis='x', colors='white', labelsize=10)
        ax.tick_params(axis='y', colors='white', labelsize=10)
        ax.set_xlabel("Squeeze Ratio (Lower is Tighter)", color='white', fontsize=13)
        ax.set_ylabel("Volume Ratio (Lower is Quieter)", color='white', fontsize=13)
        
        # Determine current phase of the animation
        if frame < PHASE_INITIAL_END_FRAME:
            # Phase 0: Initial overview, fading in all points
            title_obj.set_text("Analyzing All Potential Candidates")
            title_obj.set_color('white')
            
            # Fade in alpha for more dynamic start
            current_alpha = 0.3 + (0.7 * (frame / PHASE_INITIAL_END_FRAME))
            
            colors_data = df["breakout_readiness"].to_numpy() # Use readiness for initial color variation
            colors = plt.cm.viridis(colors_data) # Apply viridis colormap
            # edgecolors can be a single value 'w' as it's broadcast for all points
            edgecolors = 'w' 
            linewidths = 0.5
            
            criteria_text_obj.set_text('') # No specific criteria text in this phase
            if legend: legend.remove()

        elif frame < PHASE_FILTERING_END_FRAME:
            # Phase 1: Iterative filtering (one by one removal)
            title_obj.set_text("Applying Filters: Identifying Top Stocks...")
            title_obj.set_color('white') # Neutral title color during filtering

            # Calculate how many stocks should have been "processed" by this frame
            progress_in_filtering_phase = (frame - PHASE_INITIAL_END_FRAME) / FILTERING_FRAMES_DURATION
            stocks_processed_count = int(progress_in_filtering_phase * TOTAL_CANDIDATES)
            
            # Initialize colors and alphas for all points
            current_alpha = np.full(TOTAL_CANDIDATES, 0.7) # Default active alpha
            colors = np.full(TOTAL_CANDIDATES, '#CCCCCC', dtype='U7') # Default neutral color
            # Initialize edgecolors as object array to hold string colors or transparent tuples
            edgecolors = np.array(['w'] * TOTAL_CANDIDATES, dtype=object) 
            linewidths = np.full(TOTAL_CANDIDATES, 0.5)

            # Iterate through stocks to update their appearance based on processing status
            for i in range(TOTAL_CANDIDATES):
                if i < stocks_processed_count:
                    # This stock has been "processed"
                    if not is_ultimately_bullish_mask[i]:
                        # This stock FAILED the bullish criteria
                        current_alpha[i] = 0.1 # Fade out
                        colors[i] = '#990000' # Dark red for removed
                        edgecolors[i] = (0.0, 0.0, 0.0, 0.0) # Changed None to transparent RGBA tuple
                    else:
                        # This stock PASSED the bullish criteria so far
                        current_alpha[i] = 0.8
                        colors[i] = '#2ECC71' # Green for passing
                        edgecolors[i] = 'w'
                else:
                    # This stock has not yet been processed (still active/unfiltered)
                    current_alpha[i] = 0.7
                    colors[i] = '#CCCCCC' # Neutral color
                    edgecolors[i] = 'w'

            # Update criteria text to reflect current stage (or just a general message)
            criteria_text = f'Squeeze Ratio ≤ {MAX_SQUEEZE_RATIO}\nVolume Ratio ≤ {MAX_VOLUME_RATIO}\nBreakout Readiness ≥ {BULLISH_THRESHOLD}'
            criteria_text_obj.set_text(criteria_text)
            
            if legend: legend.remove() # Remove legend during iterative filtering

        else:
            # Phase 2: Final highlight (top N bullish candidates)
            title_obj.set_text(f"Top {TOP_N} Bullish Candidates!")
            title_obj.set_color('#2ECC71') # Green for final bullish

            is_top_bullish_final_mask = df["symbol"].is_in(top_bullish_symbols).to_numpy()
            
            current_alpha = np.where(is_top_bullish_final_mask, 1.0, 0.1) # Highlight top N, fade others
            colors = np.where(is_top_bullish_final_mask, '#FFFFFF', '#444444') # White for top N, dark for others
            
            # FIX: Manually construct the (N, 4) edgecolors array to avoid np.where broadcasting issues
            edgecolors_array = np.zeros((TOTAL_CANDIDATES, 4), dtype=float) # Initialize with transparent black
            edgecolors_array[is_top_bullish_final_mask] = (1.0, 1.0, 1.0, 1.0) # Set white for top bullish
            edgecolors_array[~is_top_bullish_final_mask] = (0.0, 0.0, 0.0, 0.0) # Set transparent for others
            edgecolors = edgecolors_array
            
            linewidths = 0.5
            
            # Legend for final phase
            if legend: legend.remove()
            top_patch = mpatches.Patch(color='#FFFFFF', label=f'Top {TOP_N} Ranked')
            remaining_patch = mpatches.Patch(color='#444444', label='Other Candidates')
            legend = ax.legend(handles=[top_patch, remaining_patch], loc='upper right', frameon=False, fontsize=10)
            plt.setp(legend.get_texts(), color='white')

            # Final criteria text
            criteria_text = f'Squeeze Ratio ≤ {MAX_SQUEEZE_RATIO}\nVolume Ratio ≤ {MAX_VOLUME_RATIO}\nBreakout Readiness ≥ {BULLISH_THRESHOLD}'
            criteria_text_obj.set_text(criteria_text)

        # Update the scatter plot with the calculated properties for the current frame
        scatter = ax.scatter(df["squeeze_ratio"], df["volume_ratio"], s=sizes, c=colors, 
                             alpha=current_alpha, edgecolors=edgecolors, linewidths=linewidths)
        
        # Return all artists that have been modified or created in this frame
        artists_to_return = [scatter, title_obj, criteria_text_obj]
        if legend:
            artists_to_return.extend(legend.get_patches())
            artists_to_return.extend(legend.get_texts())
        return artists_to_return

    # --- 3. Create and Save Animation ---
    print("Generating animation... This may take a few moments.")
    # Pass df and sizes to the update function using fargs
    ani = FuncAnimation(fig, update, frames=TOTAL_FRAMES, fargs=(df, sizes),
                        interval=1000/FPS, repeat=False, blit=False)

    # Save the animation
    output_filename = "iterative_squeeze_candidates_animation.mp4"
    ani.save(output_filename, writer='ffmpeg', fps=FPS, 
             progress_callback=lambda i, n: print(f'Saving frame {i+1} of {n}'))
    
    print(f"\nAnimation saved successfully as '{output_filename}'!")

if __name__ == "__main__":
    animate_ranking_process_instagram_iterative()