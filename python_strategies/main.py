import debugpy
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from typing import List, Optional, Dict, Any
import asyncio
import logging
from datetime import datetime
import httpx
from utils.utils import convert_numpy_types

# Import the backtest modules
from mr_strategy.backtest.comparison import run_entry_type_comparison
from mr_strategy.backtest.runner import BacktestRunner

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
    start_date: str
    end_date: str
    initial_capital: float

class SingleBacktestRequest(BaseModel):
    instrument_key: str
    name: str
    start_date: str
    end_date: str
    initial_capital: float
    strategy_params: Dict[str, Any]

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
                    "direction": stock["trend"]
                })
            
            # Run backtest for each instrument
            all_results = {}
            for config in instrument_configs:
                # Create backtest runner with default parameters
                runner = BacktestRunner({
                    "initial_capital": request.initial_capital,
                    "commission": 0.001,
                    "slippage": 0.001,
                    "strategy_params": {
                        "mr_start_time": "09:15:00",
                        "mr_end_time": "09:20:00",
                        "market_close_time": "15:20:00",
                        "stop_loss_pct": 0.005,
                        "target_pct": 0.02,
                        "position_size": 1,
                        "risk_per_trade": 50,
                        "use_daily_indicators": True,
                        "min_mr_size_pct": 0.002,
                        "max_mr_size_pct": 0.01,
                        "min_risk_reward": 2.0,
                        "max_trade_duration": 600,
                        "breakeven_r": 1.0,
                        "trail_activation_r": 2.0,
                        "trail_step_pct": 0.002
                    },
                    'ema': {
                        'periods': [5, 9, 50],
                        'column_prefix': 'EMA'
                    },
                    'rsi': {
                        'period': 14,
                        'column_prefix': 'RSI'
                    },
                    'atr': {
                        'period': 14,
                        'column_prefix': 'ATR'
                    }
                })
                name = config["name"]
                try:
                    logger.info(f"Running backtest for {name}")
                    result = await runner.run_single_backtest(
                        instrument_key=config["key"],
                        name=name,
                        start_date=request.start_date,
                        end_date=request.end_date,
                        timeframe="5minute"
                    )
                    all_results[name] = result
                except Exception as e:
                    logger.exception(f"Error running backtest for {name}")
                    all_results[name] = {"error": str(e)}
            
            # Calculate aggregate statistics
            total_trades = sum(result.get("total_trades", 0) for result in all_results.values() if isinstance(result, dict))
            winning_trades = sum(result.get("winning_trades", 0) for result in all_results.values() if isinstance(result, dict))
            total_pnl = sum(result.get("total_pnl", 0) for result in all_results.values() if isinstance(result, dict))
            
            # Format the final response
            final_results = {
                "individual_results": all_results,
                "aggregate_stats": {
                    "total_trades": total_trades,
                    "winning_trades": winning_trades,
                    "win_rate": (winning_trades / total_trades * 100) if total_trades > 0 else 0,
                    "total_pnl": total_pnl,
                    "average_pnl_per_trade": total_pnl / total_trades if total_trades > 0 else 0
                }
            }
            
            response = BacktestResponse(success=True, results=final_results, error=None)
            
            logger.info("Backtest completed successfully")
            return response
            
    except Exception as e:
        error_msg = f"Error running backtest: {str(e)}"
        logger.error(error_msg)
        raise HTTPException(status_code=500, detail=error_msg)

@app.post("/backtest/single", response_model=BacktestResponse)
async def run_single_backtest(request: SingleBacktestRequest):
    """
    Run a single backtest with specific strategy parameters.
    """
    try:
        logger.info(f"Starting single backtest with parameters: {request.dict()}")
        
        # Create backtest runner
        runner = BacktestRunner({
            "initial_capital": request.initial_capital,
            "commission": 0.001,
            "slippage": 0.001,
            "strategy_params": request.strategy_params,
            'ema': {
                           'periods': [5, 9, 50],
                           'column_prefix': 'EMA'
                       },
                       'rsi': {
                           'period': 14,
                           'column_prefix': 'RSI'
                       },
                       'atr': {
                           'period': 14,
                           'column_prefix': 'ATR'
                       }
        })
        
        # Run single backtest
        results = await runner.run_single_backtest(
            instrument_key=request.instrument_key,
            name=request.name,
            start_date=request.start_date,
            end_date=request.end_date,
            timeframe="5minute"
        )
        
        # Format the results
        response = BacktestResponse(success=True, results=results, error=None)
        
        logger.info("Single backtest completed successfully")
        return response
        
    except Exception as e:
        error_msg = f"Error running single backtest: {str(e)}"
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