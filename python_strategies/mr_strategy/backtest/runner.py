"""
Backtest Runner and Reports for Morning Range Strategy.

This module provides batch backtest execution capabilities and comprehensive
reporting functionality.
"""

from dataclasses import dataclass
from enum import Enum
from typing import Dict, List, Optional, Union, Tuple
import pandas as pd
import numpy as np
from datetime import datetime, timedelta
import logging
import json
import asyncio
from pathlib import Path
import matplotlib.pyplot as plt
import seaborn as sns
from concurrent.futures import ThreadPoolExecutor
import pytz
import os

from ..strategy.mr_strategy_base import MorningRangeStrategy
from ..strategy.base_strategy import BaseStrategy, StrategyConfig
from ..backtest.engine import BacktestEngine, BacktestConfig
from ..backtest.simulator import BacktestSimulator, SimulationConfig
from ..backtest.metrics import PerformanceAnalyzer
from ..data.data_processor import CandleProcessor

logger = logging.getLogger(__name__)

class BacktestMode(Enum):
    """Backtest execution modes."""
    SINGLE = "single"
    BATCH = "batch"
    OPTIMIZATION = "optimization"
    WALK_FORWARD = "walk_forward"

@dataclass
class BacktestRunConfig:
    """Configuration for backtest runs."""
    mode: BacktestMode
    start_date: str
    end_date: str
    instruments: List[str]
    strategies: List[Dict]
    initial_capital: float
    batch_size: int = 100
    parallel_runs: int = 4
    optimization_params: Dict = None
    walk_forward_windows: List[Tuple[datetime, datetime]] = None
    output_dir: str = "backtest_results"

class BacktestRunner:
    """Runner for executing and managing backtests."""
    
    def __init__(self, config: BacktestRunConfig):
        """Initialize the Backtest Runner."""
        self.config = config
        self.data_processor = CandleProcessor()
        self.performance_analyzer = PerformanceAnalyzer()
        self.results: Dict = {}
        self.reports: Dict = {}
        
        # Create output directory
        self.output_dir = Path(config.output_dir)
        self.output_dir.mkdir(parents=True, exist_ok=True)
        
        logger.info(f"Initialized BacktestRunner in {config.mode.value} mode")

    async def run_backtests(self) -> Dict:
        """Execute backtests based on configured mode."""
        if self.config.mode == BacktestMode.SINGLE:
            return await self._run_single_backtest()
        elif self.config.mode == BacktestMode.BATCH:
            return await self._run_batch_backtests()
        elif self.config.mode == BacktestMode.OPTIMIZATION:
            return await self._run_optimization()
        elif self.config.mode == BacktestMode.WALK_FORWARD:
            return await self._run_walk_forward()
        else:
            raise ValueError(f"Unsupported backtest mode: {self.config.mode}")

    async def _run_single_backtest(self) -> Dict:
        """Run a single backtest."""
        engine = self._create_backtest_engine()
        
        # Load data for each instrument
        all_data = {}
        for instrument in self.config.instruments:
            data = await self.data_processor.load_intraday_data(
                instrument_key=instrument,
                start_date=self.config.start_date,
                end_date=self.config.end_date
            )
            all_data[instrument] = data
        
        # Run backtest with loaded data
        logger.info(f"Running backtest for {len(all_data)} instruments")
        results = await engine.run_backtest(data=all_data)
        
        # Generate and save reports
        self.results["single"] = results
        self.reports["single"] = self._generate_backtest_report(results)
        
        # Save results
        self._save_results("single")
        
        return self.results["single"]

    async def _run_batch_backtests(self) -> Dict:
        """Run multiple backtests in batch."""
        all_results = {}
        
        # Split instruments into batches
        instrument_batches = [
            self.config.instruments[i:i + self.config.batch_size]
            for i in range(0, len(self.config.instruments), self.config.batch_size)
        ]
        
        # Run batches in parallel
        with ThreadPoolExecutor(max_workers=self.config.parallel_runs) as executor:
            futures = []
            for batch_id, instruments in enumerate(instrument_batches):
                config = self._create_batch_config(instruments)
                engine = self._create_backtest_engine(config)
                
                future = executor.submit(
                    asyncio.run,
                    engine.run_backtest()
                )
                futures.append((batch_id, future))
            
            # Collect results
            for batch_id, future in futures:
                try:
                    results = future.result()
                    all_results[f"batch_{batch_id}"] = results
                except Exception as e:
                    logger.error(f"Error in batch {batch_id}: {str(e)}")
        
        # Aggregate results
        self.results["batch"] = self._aggregate_batch_results(all_results)
        self.reports["batch"] = self._generate_batch_report(all_results)
        
        # Save results
        self._save_results("batch")
        
        return self.results["batch"]

    async def _run_optimization(self) -> Dict:
        """Run parameter optimization."""
        if not self.config.optimization_params:
            raise ValueError("Optimization parameters not provided")
        
        optimization_results = {}
        param_combinations = self._generate_param_combinations()
        
        # Run optimization in parallel
        with ThreadPoolExecutor(max_workers=self.config.parallel_runs) as executor:
            futures = []
            for param_set_id, params in enumerate(param_combinations):
                config = self._create_optimization_config(params)
                engine = self._create_backtest_engine(config)
                
                future = executor.submit(
                    asyncio.run,
                    engine.run_backtest()
                )
                futures.append((param_set_id, params, future))
            
            # Collect results
            for param_set_id, params, future in futures:
                try:
                    results = future.result()
                    optimization_results[f"param_set_{param_set_id}"] = {
                        "params": params,
                        "results": results
                    }
                except Exception as e:
                    logger.error(f"Error in optimization set {param_set_id}: {str(e)}")
        
        # Find optimal parameters
        self.results["optimization"] = self._find_optimal_parameters(optimization_results)
        self.reports["optimization"] = self._generate_optimization_report(optimization_results)
        
        # Save results
        self._save_results("optimization")
        
        return self.results["optimization"]

    async def _run_walk_forward(self) -> Dict:
        """Run walk-forward analysis."""
        if not self.config.walk_forward_windows:
            raise ValueError("Walk-forward windows not provided")
        
        walk_forward_results = {}
        
        # Run analysis for each window
        for window_id, (train_period, test_period) in enumerate(self.config.walk_forward_windows):
            # Train on training period
            train_config = self._create_walk_forward_config(train_period[0], train_period[1])
            train_engine = self._create_backtest_engine(train_config)
            train_results = await train_engine.run_backtest()
            
            # Test on test period
            test_config = self._create_walk_forward_config(test_period[0], test_period[1])
            test_engine = self._create_backtest_engine(test_config)
            test_results = await test_engine.run_backtest()
            
            walk_forward_results[f"window_{window_id}"] = {
                "train": train_results,
                "test": test_results
            }
        
        # Analyze walk-forward results
        self.results["walk_forward"] = self._analyze_walk_forward_results(walk_forward_results)
        self.reports["walk_forward"] = self._generate_walk_forward_report(walk_forward_results)
        
        # Save results
        self._save_results("walk_forward")
        
        return self.results["walk_forward"]

    def _create_backtest_engine(self, config: Optional[BacktestConfig] = None) -> BacktestEngine:
        """Create backtest engine instance."""
        if config is None:
            # Convert dictionary strategies to StrategyConfig objects
            strategy_configs = []
            for strategy_dict in self.config.strategies:
                strategy_configs.append(
                    StrategyConfig(
                        instrument_key=self.config.instruments[0],  # Use first instrument as default
                        range_type=strategy_dict["params"]["range_type"],
                        entry_type=strategy_dict["params"]["entry_type"],
                        sl_percentage=strategy_dict["params"]["sl_percentage"],
                        target_r=strategy_dict["params"]["target_r"],
                        buffer_ticks=5,  # Default value
                        tick_size=0.05  # Default value
                    )
                )
            
            config = BacktestConfig(
                start_date=self.config.start_date,
                end_date=self.config.end_date,
                instruments=self.config.instruments,
                strategies=strategy_configs,
                initial_capital=self.config.initial_capital,
                position_size_type="RISK_PERCENTAGE",
                max_positions=1,
                enable_parallel=True,
                cache_data=True
            )
        
        return BacktestEngine(config)

    def _generate_backtest_report(self, results: Dict) -> Dict:
        """Generate comprehensive backtest report."""
        # Initialize default summary
        default_summary = {
            "total_trades": 0,
            "win_rate": 0.0,
            "profit_factor": 0.0,
            "total_return": 0.0,
            "max_drawdown": 0.0,
            "sortino_ratio": 0.0,
            "avg_trade": 0.0,
            "avg_win": 0.0,
            "avg_loss": 0.0,
            "largest_win": 0.0,
            "largest_loss": 0.0,
            "winning_trades": 0,
            "losing_trades": 0,
            "consecutive_wins": 0,
            "consecutive_losses": 0
        }
        
        # Get trade list safely
        trade_list = results.get("trade_list", [])
        
        # Get metrics safely
        metrics = results.get("metrics", {})
        if not metrics:
            metrics = {
                "overall": self.performance_analyzer.calculate_base_metrics(trade_list),
                "entry": self.performance_analyzer.calculate_entry_metrics(trade_list),
                "range": self.performance_analyzer.calculate_range_metrics(trade_list)
            }
        
        # Extract summary from overall metrics
        overall_metrics = metrics.get("overall", {})
        if overall_metrics:
            summary = {
                "total_trades": overall_metrics.get("total_trades", 0),
                "win_rate": overall_metrics.get("win_rate", 0.0),
                "profit_factor": overall_metrics.get("profit_factor", 0.0),
                "total_return": overall_metrics.get("net_pnl", 0.0),
                "max_drawdown": overall_metrics.get("max_drawdown", 0.0),
                "sortino_ratio": overall_metrics.get("sortino_ratio", 0.0),
                "avg_trade": overall_metrics.get("average_r", 0.0),
                "avg_win": overall_metrics.get("average_win", 0.0),
                "avg_loss": overall_metrics.get("average_loss", 0.0),
                "largest_win": overall_metrics.get("largest_win", 0.0),
                "largest_loss": overall_metrics.get("largest_loss", 0.0),
                "winning_trades": overall_metrics.get("winning_trades", 0),
                "losing_trades": overall_metrics.get("losing_trades", 0),
                "consecutive_wins": overall_metrics.get("consecutive_wins", 0),
                "consecutive_losses": overall_metrics.get("consecutive_losses", 0)
            }
        else:
            summary = default_summary
        
        # Generate equity curve safely
        equity_curve = self._generate_equity_curve(trade_list)
        
        # Generate recommendations safely
        recommendations = self._generate_recommendations(summary)
        
        report = {
            "summary": summary,
            "performance_metrics": metrics,
            "equity_curve": equity_curve,
            "recommendations": recommendations
        }
        
        return report

    def _generate_batch_report(self, batch_results: Dict) -> Dict:
        """Generate report for batch backtest results."""
        consolidated_metrics = self._consolidate_batch_metrics(batch_results)
        
        report = {
            "summary": {
                "total_batches": len(batch_results),
                "successful_batches": len([r for r in batch_results.values() if r["summary"]["total_trades"] > 0]),
                "total_trades": sum(r["summary"]["total_trades"] for r in batch_results.values()),
                "average_win_rate": np.mean([r["summary"]["win_rate"] for r in batch_results.values()]),
                "total_return": sum(r["summary"]["total_return"] for r in batch_results.values())
            },
            "batch_metrics": consolidated_metrics,
            "performance_distribution": self._analyze_performance_distribution(batch_results),
            "correlation_analysis": self._analyze_strategy_correlations(batch_results),
            "recommendations": self._generate_batch_recommendations(batch_results)
        }
        
        return report

    def _generate_optimization_report(self, optimization_results: Dict) -> Dict:
        """Generate report for optimization results."""
        report = {
            "summary": {
                "total_combinations": len(optimization_results),
                "best_parameters": self._find_best_parameters(optimization_results),
                "parameter_sensitivity": self._analyze_parameter_sensitivity(optimization_results),
                "performance_surface": self._generate_performance_surface(optimization_results)
            },
            "detailed_results": {
                param_set: {
                    "parameters": results["params"],
                    "metrics": self.performance_analyzer.calculate_base_metrics(results["results"]["trade_list"])
                }
                for param_set, results in optimization_results.items()
            },
            "visualization": self._generate_optimization_plots(optimization_results),
            "recommendations": self._generate_optimization_recommendations(optimization_results)
        }
        
        return report

    def _generate_walk_forward_report(self, walk_forward_results: Dict) -> Dict:
        """Generate report for walk-forward analysis."""
        report = {
            "summary": {
                "total_windows": len(walk_forward_results),
                "in_sample_performance": self._analyze_in_sample_performance(walk_forward_results),
                "out_of_sample_performance": self._analyze_out_of_sample_performance(walk_forward_results),
                "robustness_metrics": self._calculate_robustness_metrics(walk_forward_results)
            },
            "window_analysis": {
                window_id: {
                    "train_metrics": self.performance_analyzer.calculate_base_metrics(results["train"]["trade_list"]),
                    "test_metrics": self.performance_analyzer.calculate_base_metrics(results["test"]["trade_list"]),
                    "performance_degradation": self._calculate_performance_degradation(
                        results["train"]["trade_list"],
                        results["test"]["trade_list"]
                    )
                }
                for window_id, results in walk_forward_results.items()
            },
            "stability_analysis": self._analyze_strategy_stability(walk_forward_results),
            "recommendations": self._generate_walk_forward_recommendations(walk_forward_results)
        }
        
        return report

    def _save_results(self, mode: str):
        """Save backtest results and reports."""
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        
        # Save results
        results_file = self.output_dir / f"results_{mode}_{timestamp}.json"
        with open(results_file, "w") as f:
            json.dump(self.results[mode], f, indent=2, default=str)
        
        # Save reports
        report_file = self.output_dir / f"report_{mode}_{timestamp}.json"
        with open(report_file, "w") as f:
            json.dump(self.reports[mode], f, indent=2, default=str)
        
        # Generate and save plots
        # self._save_visualization(mode, timestamp)
        
        logger.info(f"Saved {mode} results and reports to {self.output_dir}")

    def _save_visualization(self, mode: str, timestamp: str):
        """Generate and save visualization plots."""
        plot_dir = self.output_dir / "plots" / mode / timestamp
        plot_dir.mkdir(parents=True, exist_ok=True)
        
        # Generate plots based on mode
        if mode == "single":
            self._plot_equity_curve(plot_dir)
            self._plot_drawdown(plot_dir)
            self._plot_trade_distribution(plot_dir)
        elif mode == "batch":
            self._plot_batch_performance(plot_dir)
            self._plot_correlation_matrix(plot_dir)
        elif mode == "optimization":
            self._plot_parameter_sensitivity(plot_dir)
            self._plot_performance_surface(plot_dir)
        elif mode == "walk_forward":
            self._plot_walk_forward_performance(plot_dir)
            self._plot_stability_metrics(plot_dir)

    def _plot_equity_curve(self, trade_list: List[Dict]) -> Dict:
        """Generate equity curve data from trade list.

        Args:
            trade_list (List[Dict]): List of trade dictionaries with PnL information

        Returns:
            Dict: Dictionary containing equity curve data points and metadata
        """
        if not trade_list:
            return {
                "timestamps": [],
                "equity_values": [],
                "drawdowns": [],
                "max_drawdown": 0.0,
                "final_equity": self.config.initial_capital
            }

        equity = self.config.initial_capital
        equity_points = [(pd.Timestamp(trade_list[0]["entry_time"]), equity)]
        
        for trade in trade_list:
            equity += trade["pnl"]
            equity_points.append((pd.Timestamp(trade["exit_time"]), equity))
        
        # Convert to DataFrame for easier calculations
        df = pd.DataFrame(equity_points, columns=["timestamp", "equity"])
        df = df.sort_values("timestamp")
        
        # Calculate drawdown
        df["peak"] = df["equity"].cummax()
        df["drawdown"] = (df["equity"] - df["peak"]) / df["peak"] * 100
        max_drawdown = abs(df["drawdown"].min())
        
        return {
            "timestamps": df["timestamp"].tolist(),
            "equity_values": df["equity"].tolist(),
            "drawdowns": df["drawdown"].tolist(),
            "max_drawdown": max_drawdown,
            "final_equity": equity
        }

    def _generate_equity_curve(self, trade_list: List[Dict]) -> Dict:
        """Generate equity curve data from trade list.

        Args:
            trade_list (List[Dict]): List of trade dictionaries with PnL information

        Returns:
            Dict: Dictionary containing equity curve data points and metadata
        """
        if not trade_list:
            return {
                "timestamps": [],
                "equity_values": [],
                "drawdowns": [],
                "max_drawdown": 0.0,
                "final_equity": self.config.initial_capital
            }

        equity = self.config.initial_capital
        equity_points = [(pd.Timestamp(trade_list[0]["entry_time"]), equity)]
        
        for trade in trade_list:
            equity += trade["pnl"]
            equity_points.append((pd.Timestamp(trade["exit_time"]), equity))
        
        # Convert to DataFrame for easier calculations
        df = pd.DataFrame(equity_points, columns=["timestamp", "equity"])
        df = df.sort_values("timestamp")
        
        # Calculate drawdown
        df["peak"] = df["equity"].cummax()
        df["drawdown"] = (df["equity"] - df["peak"]) / df["peak"] * 100
        max_drawdown = abs(df["drawdown"].min())
        
        return {
            "timestamps": df["timestamp"].tolist(),
            "equity_values": df["equity"].tolist(),
            "drawdowns": df["drawdown"].tolist(),
            "max_drawdown": max_drawdown,
            "final_equity": equity
        }

    def _generate_trade_list(self, results: Dict) -> List[Dict]:
        """Generate a list of trades from backtest results.
        
        Args:
            results (Dict): Backtest results dictionary
            
        Returns:
            List[Dict]: List of trade dictionaries with entry/exit details
        """
        trade_list = []
        
        for strategy_id, strategy_trades in results.items():
            for trade in strategy_trades:
                if trade["action"] == "exit":
                    trade_list.append({
                        "strategy_id": strategy_id,
                        "entry_time": trade["result"]["entry_time"],
                        "exit_time": trade["result"]["exit_time"],
                        "entry_price": trade["result"]["entry_price"],
                        "exit_price": trade["result"]["exit_price"],
                        "position_size": trade["result"]["position_size"],
                        "pnl": trade["result"]["realized_pnl"],
                        "r_multiple": trade["result"].get("r_multiple", 0),
                        "exit_reason": trade["result"]["status"]
                    })
        
        return sorted(trade_list, key=lambda x: x["entry_time"])

    def _generate_recommendations(self, results: Dict) -> List[str]:
        """Generate strategy improvement recommendations."""
        recommendations = []
        
        # # Analyze win rate
        # if results["summary"]["win_rate"] < 0.5:
        #     recommendations.append("Consider reviewing entry criteria to improve win rate")
        
        # # Analyze risk-reward
        # if results["summary"]["profit_factor"] < 1.5:
        #     recommendations.append("Review risk-reward ratios and take profit levels")
        
        # # Analyze drawdown
        # if results["summary"]["max_drawdown"] > results["summary"]["total_return"] * 0.3:
        #     recommendations.append("Consider implementing stricter risk management rules")
        
        # # Analyze trade frequency
        # avg_trades_per_day = results["summary"]["total_trades"] / len(results["equity_curve"])
        # if avg_trades_per_day < 0.5:
        #     recommendations.append("Consider relaxing entry criteria to increase trade frequency")
        # elif avg_trades_per_day > 5:
        #     recommendations.append("Consider implementing trade filters to reduce false signals")
        
        return recommendations

def print_and_visualize_results(results, reports):
    """Print and visualize backtest results."""
    
    print("\n=============================================")
    print("MORNING RANGE STRATEGY BACKTEST RESULTS")
    print("=============================================")
    print(f"Instrument: {INSTRUMENT_KEY}")
    print(f"Period: {START_DATE} to {END_DATE}")
    print("---------------------------------------------")
    
    # Print summary statistics
    print("\nOVERALL PERFORMANCE:")
    summary = reports['single']['summary']
    
    # Print table format
    print("-" * 60)
    print(f"{'Metric':<25} {'Value':<20}")
    print("-" * 60)
    print(f"{'Total Trades':<25} {summary['total_trades']:<20}")
    print(f"{'Winning Trades':<25} {summary['winning_trades']:<20}")
    print(f"{'Losing Trades':<25} {summary['losing_trades']:<20}")
    print(f"{'Win Rate':<25} {summary['win_rate']:.2%}")
    print(f"{'Profit Factor':<25} {summary['profit_factor']:.2f}")
    print(f"{'Average Profit':<25} {summary['average_win']:.2f}")
    print(f"{'Average Loss':<25} {summary['average_loss']:.2f}")
    print(f"{'Profit %':<25} {summary['profit_percentage']:.2f}%")
    print(f"{'Loss %':<25} {summary['loss_percentage']:.2f}%")
    print(f"{'Expectancy':<25} {summary['expectancy']:.2f}")
    print(f"{'Total Profit':<25} {summary['total_profit']:.2f}")
    print(f"{'Total Loss':<25} {summary['total_loss']:.2f}")
    print(f"{'Overall PNL':<25} {summary['overall_pnl']:.2f}")
    print(f"{'Total Return':<25} {summary['total_return']:.2f}")
    print(f"{'Max Drawdown':<25} {summary['max_drawdown']:.2f}")
    print("-" * 60)
    
    # Extract and compare strategy results
    strategy_results = reports['single']['performance_metrics']
    
    print("\nENTRY TYPE COMPARISON:")
    print("-" * 60)
    print(f"{'Entry Type':<15} {'Total Trades':<12} {'Win Rate':<10} {'Avg Profit':<12} {'Avg Loss':<12}")
    print("-" * 60)
    
    for strategy_id, metrics in strategy_results.items():
        entry_type = strategy_id.split('_')[-1]
        print(f"{entry_type:<15} {metrics['total_trades']:<12} {metrics['win_rate']:<12} "
              f"{metrics['average_win']:<12} {metrics['average_loss']:<12}")
    
    # Visualize equity curves if available
    if 'equity_curve' in results and not results['equity_curve'].empty:
        plt.figure(figsize=(12, 6))
        
        # Group by strategy_id and plot
        for strategy_id in results['equity_curve']['strategy_id'].unique():
            strategy_data = results['equity_curve'][results['equity_curve']['strategy_id'] == strategy_id]
            entry_type = strategy_id.split('_')[-1]
            plt.plot(strategy_data['timestamp'], strategy_data['equity'], label=entry_type)
        
        plt.title('Equity Curve Comparison by Entry Type')
        plt.xlabel('Date')
        plt.ylabel('Equity')
        plt.legend()
        plt.grid(True)
        
        # Save the plot
        output_dir = "backtest_results/plots"
        os.makedirs(output_dir, exist_ok=True)
        plt.savefig(f"{output_dir}/equity_comparison.png")
        plt.close()
        
        print(f"\nEquity curve comparison saved to {output_dir}/equity_comparison.png")
    
    # Print recommendations
    if 'recommendations' in reports['single']:
        print("\nRECOMMENDATIONS:")
        for rec in reports['single']['recommendations']:
            print(f"- {rec}")
    
    # Print daily P&L information
    trade_list = results.get('trade_list', []) or results.get('trades', [])
    if trade_list:
        print("\nDAILY PROFIT/LOSS BREAKDOWN:")
        print("-" * 60)
        print(f"{'Date':<12} {'Profit/Loss':<15} {'Status':<10}")
        print("-" * 60)
        
        # Process trades by date
        trade_days = {}
        for trade in trade_list:
            # Extract date in YYYY-MM-DD format
            entry_time = trade.get('entry_time')
            if entry_time:
                trade_date = pd.to_datetime(entry_time).strftime('%Y-%m-%d')
                pnl = trade.get('realized_pnl', 0)
                
                if trade_date not in trade_days:
                    trade_days[trade_date] = 0
                
                trade_days[trade_date] += pnl
        
        # Sort dates and print
        for date in sorted(trade_days.keys()):
            pnl = trade_days[date]
            status = "PROFIT" if pnl > 0 else "LOSS" if pnl < 0 else "FLAT"
            print(f"{date:<12} {pnl:<15.2f} {status:<10}")
            
        print("-" * 60)
