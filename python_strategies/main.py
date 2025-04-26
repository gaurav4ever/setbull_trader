import debugpy
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List, Optional
import asyncio
import logging
from datetime import datetime
import httpx
from utils.utils import convert_numpy_types
# Import the backtest function from test_mr_strategy
from test_mr_strategy import run_entry_type_comparison, print_and_visualize_results

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

class BacktestRequest(BaseModel):
    instrument_key: str
    start_date: str
    end_date: str
    initial_capital: float
    entry_types: List[str]
    instrument_configs: List[dict]

class BacktestResponse(BaseModel):
    success: bool
    results: dict
    error: Optional[str] = None

@app.post("/backtest/run/single", response_model=BacktestResponse)
async def run_single_backtest(request: BacktestRequest):
    """
    Run a single backtest with the provided parameters.
    """
    results = await run_entry_type_comparison(instrument_configs=request.instrument_configs)
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
        
        # Fetch top 10 filtered stocks from the filter pipeline API
        async with httpx.AsyncClient() as client:
            response = await client.get("http://localhost:8080/api/v1/filter-pipeline/fetch/top-10")
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail="Failed to fetch filtered stocks")
            
            filtered_stocks = response.json()
            if not filtered_stocks.get("success", False):
                raise HTTPException(status_code=500, detail="Failed to fetch filtered stocks")
            
            # Extract instrument configs from filtered stocks
            instrument_configs = []
            for stock in filtered_stocks.get("data", []):
                instrument_configs.append({
                    "key": stock["instrument_key"],
                    "name": stock["symbol"],
                    "direction": "BEARISH"
                })
                instrument_configs.append({
                    "key": stock["instrument_key"],
                    "name": stock["symbol"],
                    "direction": "BULLISH"
                })
            
            # Run the backtest with the filtered stocks
            results = await run_entry_type_comparison(instrument_configs=instrument_configs)
        
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