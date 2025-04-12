import debugpy
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List, Optional
import asyncio
import logging
from datetime import datetime
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

class BacktestResponse(BaseModel):
    success: bool
    results: dict
    error: Optional[str] = None

@app.post("/backtest/run", response_model=BacktestResponse)
async def run_backtest(request: BacktestRequest):
    """
    Run MR Strategy backtest with the provided parameters.
    """
    try:
        logger.info(f"Starting backtest with parameters: {request.dict()}")
        
        # Run the backtest
        results = await run_entry_type_comparison()
        
        # Format the results
        response = BacktestResponse(success=True, results=results['metrics'], error=None)
        # convert response to json
        
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