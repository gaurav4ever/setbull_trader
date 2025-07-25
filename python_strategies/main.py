import debugpy
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List, Optional
import asyncio
import logging
from datetime import datetime
import httpx
from mr_strategy.backtest.runner import BacktestMode, BacktestRunConfig
from utils.utils import convert_numpy_types
# Import the backtest function from test_mr_strategy
from test_mr_strategy import run_entry_type_comparison, print_and_visualize_results
from analysis.analyze_trades import analyze_trades

# Allow debugger to attach on port 5678
debugpy.listen(("0.0.0.0", 5678))
print("✅ Waiting for debugger attach...")
debugpy.wait_for_client()  # Optional: will pause execution until debugger is attached

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler("backtest_server.log"),
        logging.StreamHandler()
    ]
)

logger = logging.getLogger(__name__)

app = FastAPI(
    title="MR Strategy Backtest Server",
    description="Simple server to run MR Strategy backtests",
    version="1.0.0"
)

class SingleBacktestRequest(BaseModel):
    instrument_configs: List[dict]
    runner_config: dict
class BacktestRequest(BaseModel):
    runner_config: dict
class BacktestResponse(BaseModel):
    success: bool
    results: dict
    error: Optional[str] = None

@app.get("/backtest/analyze/trades")
async def analyze_all_trades():
    analyze_trades()

@app.post("/backtest/run/single", response_model=BacktestResponse)
async def run_single_backtest(request: SingleBacktestRequest):
    """
    Run a single backtest with the provided parameters.
    """
    json_data = request.runner_config
    strategies = json_data["strategies"]
    instruments = request.instrument_configs
    # convert strategies to dictionary
    strategies = [dict(strategy) for strategy in strategies]
    instruments = [dict(instrument) for instrument in instruments]
    config = BacktestRunConfig(
        mode=BacktestMode[json_data["mode"]],
        start_date=json_data["start_date"],
        end_date=json_data["end_date"],
        instruments=instruments,
        strategies=strategies,
        initial_capital=json_data["initial_capital"],
        output_dir=json_data.get("output_dir", "backtest_results")
    )
    results = await run_entry_type_comparison(instrument_configs=request.instrument_configs, runner_config=config)
    # Format the results
    response = BacktestResponse(success=True, results=results['metrics'], error=None)
        
    logger.info("Backtest completed successfully")
    return response

@app.post("/backtest/run", response_model=BacktestResponse)
async def run_backtest(request: BacktestRequest):
    """
    Run MR Strategy backtest with the provided parameters.
    """
    try:
        logger.info(f"Starting backtest with parameters: {request.dict()}")
        
        # 1. Fetch top 10 filtered stocks from the filter pipeline API
        async with httpx.AsyncClient() as client:
            response = await client.get("http://localhost:8083/api/v1/filter-pipeline/fetch/top-10")
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail="Failed to fetch filtered stocks")
            
            filtered_stocks = response.json()
            if not filtered_stocks.get("success", False):
                raise HTTPException(status_code=500, detail="Failed to fetch filtered stocks")
            
            # Extract instrument configs from filtered stocks
            instrument_configs = []
            instrument_keys = []
            for stock in filtered_stocks.get("data", []):
                instrument_configs.append({
                    "key": stock["instrument_key"],
                    "name": stock["symbol"],
                    "direction": "BULLISH"
                })
                instrument_keys.append(stock["instrument_key"])
            
            logger.info(f"Fetched {len(instrument_keys)} stocks for data ingestion")
            
            # ENABLE_DATA_INGESTION = True  # Set to False to skip data ingestion
            ENABLE_DATA_INGESTION = False
            
            if ENABLE_DATA_INGESTION:
                # 2. Ingest 1min candle data for last 30 trading days with batch request of 4 days
                from datetime import datetime, timedelta
                import calendar
                
                # Calculate date range for last 30 trading days
                end_date = datetime.now().date()
                start_date = end_date - timedelta(days=45)  # Add buffer for weekends/holidays
                
                # Create batches of 4 days each
                batch_size = 4
                current_date = start_date
                batch_count = 0
                
                while current_date <= end_date:
                    batch_end_date = min(current_date + timedelta(days=batch_size-1), end_date)
                    
                    batch_request = {
                        "instrumentKeys": instrument_keys,
                        "fromDate": current_date.strftime("%Y-%m-%d"),
                        "toDate": batch_end_date.strftime("%Y-%m-%d"),
                        "interval": "1minute"
                    }
                    
                    logger.info(f"Ingesting batch {batch_count + 1}: {current_date} to {batch_end_date}")
                    
                    try:
                        batch_response = await client.post(
                            "http://localhost:8083/api/v1/historical-data/batch-store",
                            json=batch_request,
                            timeout=300  # 5 minutes timeout for batch operations
                        )
                        
                        if batch_response.status_code != 200:
                            logger.warning(f"Batch {batch_count + 1} failed with status {batch_response.status_code}")
                        else:
                            batch_result = batch_response.json()
                            if batch_result.get("success", False):
                                logger.info(f"Batch {batch_count + 1} completed successfully")
                            else:
                                logger.warning(f"Batch {batch_count + 1} failed: {batch_result.get('message', 'Unknown error')}")
                        
                    except Exception as batch_error:
                        logger.error(f"Error in batch {batch_count + 1}: {str(batch_error)}")
                    
                    current_date = batch_end_date + timedelta(days=1)
                    batch_count += 1
                
                logger.info(f"Data ingestion completed for {len(instrument_keys)} stocks in {batch_count} batches")
            else:
                logger.info("Data ingestion skipped - ENABLE_DATA_INGESTION is False")
            
            # 3. Once data ingested then perform rest of the backtesting
            json_data = request.runner_config
            strategies = json_data["strategies"]
            # convert strategies to dictionary
            strategies = [dict(strategy) for strategy in strategies]
            instruments = [dict(instrument) for instrument in instrument_configs]
            config = BacktestRunConfig(
                mode=BacktestMode[json_data["mode"]],
                start_date=json_data["start_date"],
                end_date=json_data["end_date"],
                instruments=instruments,
                strategies=strategies,
                initial_capital=json_data["initial_capital"],
                output_dir=json_data.get("output_dir", "backtest_results")
            )
            
            # Run the backtest with the filtered stocks
            results = await run_entry_type_comparison(instrument_configs=instrument_configs, runner_config=config)
        
        # Format the results
        response = BacktestResponse(success=True, results=results['metrics'], error=None)
        
        logger.info("Backtest completed successfully")
        return response
        
    except Exception as e:
        error_msg = f"Error running backtest: {str(e)}"
        logger.error(error_msg)
        raise HTTPException(status_code=500, detail=error_msg)

@app.get("/health")
async def health_check():
    """
    Health check endpoint
    """
    return {
        "status": "healthy",
        "timestamp": datetime.now().isoformat()
    }

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=3000) 