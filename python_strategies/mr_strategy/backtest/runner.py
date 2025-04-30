"""
Backtest Runner and Reports for Range Strategy.

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
            data_feed = await self.data_processor.load_and_process_candles(
                instrument_key=instrument.get('key'),
                name=instrument.get('name'),
                start_date=self.config.start_date,
                end_date=self.config.end_date
            )
            all_data[instrument.get('key')] = data_feed
        
        # Run backtest with loaded data
        logger.info(f"Running backtest for {len(all_data)} instruments")
        results = await engine.run_backtest(all_data_feed=all_data)
        
        # Generate and save reports
        self.results["single"] = results
        self.reports["single"] = self._generate_backtest_report(results)
        
        # Save results
        # self._save_results("single")
        
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
                for instrument in self.config.instruments:
                    strategy_configs.append(
                        StrategyConfig(
                            instrument_key=instrument,  # Use first instrument as default
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
        total_trades = 0
        winning_trades = 0
        losing_trades = 0
        avg_win = 0
        avg_loss = 0
        total_profit = 0
        total_loss = 0
        net_pnl = 0
        for key, value in metrics.items():
            total_trades += value["total_trades"]
            winning_trades += value["winning_trades"]
            losing_trades += value["losing_trades"]
            avg_win += value["average_win"]
            avg_loss += value["average_loss"]
            net_pnl += value["net_pnl"]
            total_profit += value["total_profit"]
            total_loss += value["total_loss"]

        summary = {
                "total_trades": total_trades,
                "win_rate": winning_trades / total_trades,
                "profit_factor": net_pnl / abs(net_pnl),
                "total_return": net_pnl,
                "avg_trade": avg_win + avg_loss,
                "avg_win": avg_win,
                "avg_loss": avg_loss,
                "winning_trades": winning_trades,
                "losing_trades": losing_trades
            }
        
        # Generate equity curve safely
        # equity_curve = self._generate_equity_curve(trade_list)
        
        # Generate recommendations safely
        # recommendations = self._generate_recommendations(summary)
        
        # Generate instrument-specific reports
        instrument_reports = {}
        for instrument_key, instrument_data in results.get('instruments', {}).items():
            instrument_reports[instrument_key] = {
                'direction': instrument_data['direction'],
                'metrics': instrument_data['metrics'],
                'signals': len(instrument_data['signals']),
                'trades': len(instrument_data['trades'])
                # 'equity_curve': self._generate_equity_curve(instrument_data['trades'])
            }
        
        report = {
            "summary": summary,
            "performance_metrics": metrics,
            # "equity_curve": equity_curve,
            # "recommendations": recommendations,
            "instruments": instrument_reports
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
        
        # Save instrument-specific results
        instruments_dir = self.output_dir / "instruments"
        instruments_dir.mkdir(exist_ok=True)
        
        for instrument_key, instrument_data in self.results[mode].get('instruments', {}).items():
            instrument_file = instruments_dir / f"{instrument_key}_{timestamp}.json"
            with open(instrument_file, "w") as f:
                json.dump(instrument_data, f, indent=2, default=str)
        
        # Generate and save plots
        # self._save_visualization(mode, timestamp)
        
        logger.info(f"Saved {mode} results and reports to {self.output_dir}")

    def _save_visualization(self, mode: str, timestamp: str):
        """Generate and save visualization plots."""
        plot_dir = self.output_dir / "plots" / mode / timestamp
        plot_dir.mkdir(parents=True, exist_ok=True)
        
        # Generate plots based on mode
        if mode == "single":
            # Plot overall equity curve
            self._plot_equity_curve(plot_dir)
            
            # Plot instrument-specific equity curves
            instruments_dir = plot_dir / "instruments"
            instruments_dir.mkdir(exist_ok=True)
            
            for instrument_key, instrument_data in self.results[mode].get('instruments', {}).items():
                self._plot_instrument_equity_curve(instrument_key, instrument_data, instruments_dir)
            
            # Plot drawdown
            self._plot_drawdown(plot_dir)
            
            # Plot trade distribution
            self._plot_trade_distribution(plot_dir)
            
            # Plot instrument-specific trade distributions
            for instrument_key, instrument_data in self.results[mode].get('instruments', {}).items():
                self._plot_instrument_trade_distribution(instrument_key, instrument_data, instruments_dir)
        
        elif mode == "batch":
            self._plot_batch_performance(plot_dir)
            self._plot_correlation_matrix(plot_dir)
        
        elif mode == "optimization":
            self._plot_parameter_sensitivity(plot_dir)
            self._plot_performance_surface(plot_dir)
        
        elif mode == "walk_forward":
            self._plot_walk_forward_performance(plot_dir)
            self._plot_stability_metrics(plot_dir)

    def _plot_instrument_equity_curve(self, instrument_key: str, instrument_data: Dict, plot_dir: Path):
        """Plot equity curve for a specific instrument."""
        if 'equity_curve' not in instrument_data:
            return
        
        plt.figure(figsize=(12, 6))
        plt.plot(instrument_data['equity_curve'].index, instrument_data['equity_curve'])
        plt.title(f'Equity Curve - {instrument_key} ({instrument_data["direction"]})')
        plt.xlabel('Date')
        plt.ylabel('Equity')
        plt.grid(True)
        
        # Save the plot
        plt.savefig(plot_dir / f"equity_curve_{instrument_key}.png")
        plt.close()

    def _plot_instrument_trade_distribution(self, instrument_key: str, instrument_data: Dict, plot_dir: Path):
        """Plot trade distribution for a specific instrument."""
        if 'trades' not in instrument_data:
            return
        
        trades_df = pd.DataFrame(instrument_data['trades'])
        if trades_df.empty:
            return
        
        plt.figure(figsize=(12, 6))
        plt.hist(trades_df['realized_pnl'], bins=20)
        plt.title(f'Trade Distribution - {instrument_key} ({instrument_data["direction"]})')
        plt.xlabel('P&L')
        plt.ylabel('Frequency')
        plt.grid(True)
        
        # Save the plot
        plt.savefig(plot_dir / f"trade_distribution_{instrument_key}.png")
        plt.close()

    def _generate_equity_curve(self, trade_list: List[Dict]) -> Dict:
        """
        Generate equity curve from trade list.
        
        Args:
            trade_list: List of trade dictionaries, each containing:
                - instrument_key: str
                - entry_time: datetime
                - exit_time: datetime
                - entry_price: float
                - exit_price: float
                - position_size: float
                - realized_pnl: float
                - status: str
                
        Returns:
            Dictionary containing:
                - equity_curve: pd.Series with timestamp index and equity values
                - drawdown_curve: pd.Series with timestamp index and drawdown values
                - instrument_curves: Dict[str, pd.Series] mapping instrument keys to their equity curves
        """
        if not trade_list:
            return {
                'equity_curve': pd.Series(),
                'drawdown_curve': pd.Series(),
                'instrument_curves': {}
            }
        
        # Convert trade list to DataFrame
        trades_df = pd.DataFrame(trade_list)
        
        # Convert timestamps to datetime
        trades_df['entry_time'] = pd.to_datetime(trades_df['entry_time'])
        trades_df['exit_time'] = pd.to_datetime(trades_df['exit_time'])
        
        # Sort trades by exit time
        trades_df = trades_df.sort_values('exit_time')
        
        # Group trades by instrument
        instrument_groups = trades_df.groupby('instrument_key')
        
        # Calculate equity curves for each instrument
        instrument_curves = {}
        for instrument_key, instrument_trades in instrument_groups:
            # Calculate cumulative P&L for this instrument
            instrument_pnl = instrument_trades['realized_pnl'].cumsum()
            
            # Create equity curve for this instrument
            instrument_curve = self.config.initial_capital + instrument_pnl
            
            # Set index to exit times
            instrument_curve.index = instrument_trades['exit_time']
            
            # Store instrument curve
            instrument_curves[instrument_key] = instrument_curve
        
        # Calculate portfolio equity curve
        if instrument_curves:
            # Combine all instrument curves
            portfolio_curve = pd.concat(instrument_curves.values())
            
            # Sort by timestamp
            portfolio_curve = portfolio_curve.sort_index()
            
            # Calculate cumulative sum
            portfolio_curve = portfolio_curve.groupby(portfolio_curve.index).sum()
            
            # Add initial capital
            portfolio_curve = self.config.initial_capital + portfolio_curve.cumsum()
        else:
            portfolio_curve = pd.Series([self.config.initial_capital])
            portfolio_curve.index = [trades_df['exit_time'].min()]
        
        # Calculate drawdown curve
        rolling_max = portfolio_curve.expanding().max()
        drawdown_curve = (portfolio_curve - rolling_max) / rolling_max
        
        return {
            'equity_curve': portfolio_curve,
            'drawdown_curve': drawdown_curve,
            'instrument_curves': instrument_curves
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
    print("RANGE STRATEGY BACKTEST RESULTS")
    print("=============================================")
    print(f"Instruments: {[f'{inst['key']} ({inst['direction']})' for inst in results['instruments'].keys()]}")
    print(f"Period: {results['start_date']} to {results['end_date']}")
    print("---------------------------------------------")
    
    # Print summary statistics
    print("\nOVERALL PERFORMANCE:")
    summary = reports['single']['summary']
    
    # Calculate additional metrics
    winning_trades = int(summary['winning_trades'])
    losing_trades = int(summary['losing_trades'])
    avg_profit = summary.get('avg_win', 0)
    avg_loss = summary.get('avg_loss', 0)
    total_profit = winning_trades * avg_profit if avg_profit else 0
    total_loss = losing_trades * avg_loss if avg_loss else 0
    overall_pnl = total_profit + total_loss
    profit_percentage = (avg_profit / results['initial_capital'] * 100) if avg_profit else 0
    loss_percentage = (avg_loss / results['initial_capital'] * 100) if avg_loss else 0
    expectancy = (summary['win_rate'] * avg_profit) + ((1 - summary['win_rate']) * avg_loss)
    
    # Print table format
    print("-" * 60)
    print(f"{'Metric':<25} {'Value':<20}")
    print("-" * 60)
    print(f"{'Total Trades':<25} {summary['total_trades']:<20}")
    print(f"{'Winning Trades':<25} {winning_trades:<20}")
    print(f"{'Losing Trades':<25} {losing_trades:<20}")
    print(f"{'Win Rate':<25} {summary['win_rate']:.2%}")
    print(f"{'Profit Factor':<25} {summary['profit_factor']:.2f}")
    print(f"{'Average Profit':<25} {avg_profit:.2f}")
    print(f"{'Average Loss':<25} {avg_loss:.2f}")
    print(f"{'Profit %':<25} {profit_percentage*100:.2f}%")
    print(f"{'Loss %':<25} {loss_percentage*100:.2f}%")
    print(f"{'Expectancy':<25} {expectancy:.2f}")
    print(f"{'Total Profit':<25} {total_profit:.2f}")
    print(f"{'Total Loss':<25} {total_loss:.2f}")
    print(f"{'Overall PNL':<25} {overall_pnl:.2f}")
    print(f"{'Total Return':<25} {summary['total_return']:.2f}")
    print(f"{'Max Drawdown':<25} {summary['max_drawdown']:.2f}")
    print("-" * 60)
    
    # Print instrument-specific results
    print("\nINSTRUMENT-SPECIFIC PERFORMANCE:")
    for instrument_key, instrument_data in results['instruments'].items():
        print(f"\n{instrument_key} ({instrument_data['direction']}):")
        print("-" * 60)
        print(f"{'Metric':<25} {'Value':<20}")
        print("-" * 60)
        print(f"{'Total Trades':<25} {instrument_data['metrics']['total_trades']:<20}")
        print(f"{'Win Rate':<25} {instrument_data['metrics']['win_rate']:.2%}")
        print(f"{'Profit Factor':<25} {instrument_data['metrics']['profit_factor']:.2f}")
        print(f"{'Average R':<25} {instrument_data['metrics']['average_r']:.2f}")
        print(f"{'Max Drawdown':<25} {instrument_data['metrics']['max_drawdown']:.2f}")
        print(f"{'Sharpe Ratio':<25} {instrument_data['metrics']['sharpe_ratio']:.2f}")
        print("-" * 60)
    
    # Print overall portfolio results
    print("\nOVERALL PORTFOLIO PERFORMANCE:")
    portfolio_metrics = results['portfolio']['metrics']
    print("-" * 60)
    print(f"{'Metric':<25} {'Value':<20}")
    print("-" * 60)
    print(f"{'Total Trades':<25} {portfolio_metrics['total_trades']:<20}")
    print(f"{'Win Rate':<25} {portfolio_metrics['win_rate']:.2%}")
    print(f"{'Profit Factor':<25} {portfolio_metrics['profit_factor']:.2f}")
    print(f"{'Average R':<25} {portfolio_metrics['average_r']:.2f}")
    print(f"{'Max Drawdown':<25} {portfolio_metrics['max_drawdown']:.2f}")
    print(f"{'Sharpe Ratio':<25} {portfolio_metrics['sharpe_ratio']:.2f}")
    print("-" * 60)
    
    # Visualize equity curves
    if 'equity_curve' in results['portfolio'] and not results['portfolio']['equity_curve'].empty:
        plt.figure(figsize=(12, 6))
        
        # Plot individual instrument equity curves
        for instrument_key, instrument_data in results['instruments'].items():
            if 'equity_curve' in instrument_data:
                plt.plot(
                    instrument_data['equity_curve'].index,
                    instrument_data['equity_curve'],
                    label=f"{instrument_key} ({instrument_data['direction']})"
                )
        
        # Plot portfolio equity curve
        plt.plot(
            results['portfolio']['equity_curve'].index,
            results['portfolio']['equity_curve'],
            label='Portfolio',
            linestyle='--',
            linewidth=2
        )
        
        plt.title('Equity Curve Comparison')
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
    
    # Print daily P&L information
    trade_list = results.get('trades', [])
    if trade_list:
        print("\nDAILY PROFIT/LOSS BREAKDOWN:")
        print("-" * 60)
        print(f"{'Date':<12} {'Instrument':<15} {'Profit/Loss':<15} {'Status':<10} {'Direction':<10}")
        print("-" * 60)
        
        # Process trades by date and instrument
        trade_days = dict()
        for trade in trade_list:
            # Extract date in YYYY-MM-DD format
            entry_time = trade.get('current_time')
            if entry_time:
                trade_date = pd.to_datetime(entry_time).strftime('%Y-%m-%d')
                instrument_key = trade.get('instrument_key', 'UNKNOWN')
                pnl = trade.get('realized_pnl', 0)
                direction = trade.get('position_type', 'UNKNOWN')
                
                if trade_date not in trade_days:
                    trade_days[trade_date] = {}
                
                if instrument_key not in trade_days[trade_date]:
                    trade_days[trade_date][instrument_key] = {
                        "pnl": 0,
                        "direction": direction
                    }
                
                trade_days[trade_date][instrument_key]["pnl"] += pnl
        
        # Sort dates and print
        for date in sorted(trade_days.keys()):
            print(f"\n{date}:")
            for instrument_key, data in trade_days[date].items():
                pnl = data["pnl"]
                direction = data["direction"]
                status = "PROFIT" if pnl > 0 else "LOSS" if pnl < 0 else "FLAT"
                print(f"{'':<12} {instrument_key:<15} {pnl:<15.2f} {status:<10} {direction:<10}")
            
        print("-" * 60)
