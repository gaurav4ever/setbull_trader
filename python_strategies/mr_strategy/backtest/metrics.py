"""
Performance Analytics Framework for Morning Range Strategy.

This module provides comprehensive performance metrics and analysis tools
for evaluating strategy performance.
"""

from dataclasses import dataclass
from enum import Enum
from typing import Dict, List, Optional, Union, Tuple
import pandas as pd
import numpy as np
from datetime import datetime, timedelta
import logging
from scipy import stats

logger = logging.getLogger(__name__)

class MetricCategory(Enum):
    """Categories of performance metrics."""
    RETURNS = "returns"
    RISK = "risk"
    ENTRY = "entry"
    EXIT = "exit"
    RANGE = "range"
    TIMING = "timing"
    VOLUME = "volume"

@dataclass
class BaseMetrics:
    """Base performance metrics."""
    total_trades: int = 0
    winning_trades: int = 0
    losing_trades: int = 0
    win_rate: float = 0.0
    profit_factor: float = 0.0
    avg_profit: float = 0.0
    avg_loss: float = 0.0
    largest_win: float = 0.0
    largest_loss: float = 0.0
    avg_trade: float = 0.0
    avg_bars_in_trade: float = 0.0
    total_return: float = 0.0
    annualized_return: float = 0.0
    sharpe_ratio: float = 0.0
    sortino_ratio: float = 0.0
    max_drawdown: float = 0.0
    max_drawdown_duration: int = 0
    recovery_factor: float = 0.0
    risk_adjusted_return: float = 0.0

@dataclass
class EntryMetrics:
    """Entry-specific performance metrics."""
    avg_entry_efficiency: float = 0.0
    missed_entries: int = 0
    false_signals: int = 0
    entry_time_distribution: Dict[str, float] = None
    avg_entry_slippage: float = 0.0
    successful_entry_patterns: Dict[str, int] = None
    failed_entry_patterns: Dict[str, int] = None
    avg_entry_range_position: float = 0.0
    entry_volume_profile: Dict[str, float] = None

@dataclass
class RangeMetrics:
    """Range-specific performance metrics."""
    avg_range_size: float = 0.0
    range_success_rate: float = 0.0
    range_failure_rate: float = 0.0
    range_breakout_distribution: Dict[str, float] = None
    range_volume_profile: Dict[str, float] = None
    range_duration_stats: Dict[str, float] = None
    range_quality_score: float = 0.0

class PerformanceAnalyzer:
    """Core performance analysis engine."""
    
    def __init__(self, risk_free_rate: float = 0.05):
        """Initialize Performance Analyzer."""
        self.risk_free_rate = risk_free_rate
        self.trades: List[Dict] = []
        self.daily_returns: pd.Series = None
        self.equity_curve: pd.Series = None
        
        logger.info("Initialized PerformanceAnalyzer")

    def calculate_base_metrics(self, trades: List[Dict]) -> BaseMetrics:
        """Calculate base performance metrics."""
        if not trades:
            return BaseMetrics()
        
        # Convert trades to DataFrame
        df = pd.DataFrame(trades)
        
        # Calculate basic statistics
        total_trades = len(trades)
        winning_trades = len(df[df['realized_pnl'] > 0])
        losing_trades = len(df[df['realized_pnl'] <= 0])
        
        # Calculate returns
        total_profit = df[df['realized_pnl'] > 0]['realized_pnl'].sum()
        total_loss = abs(df[df['realized_pnl'] <= 0]['realized_pnl'].sum())
        
        metrics = BaseMetrics(
            total_trades=total_trades,
            winning_trades=winning_trades,
            losing_trades=losing_trades,
            win_rate=winning_trades / total_trades if total_trades > 0 else 0,
            profit_factor=total_profit / total_loss if total_loss > 0 else float('inf'),
            avg_profit=df[df['realized_pnl'] > 0]['realized_pnl'].mean(),
            avg_loss=df[df['realized_pnl'] <= 0]['realized_pnl'].mean(),
            largest_win=df['realized_pnl'].max(),
            largest_loss=df['realized_pnl'].min(),
            avg_trade=df['realized_pnl'].mean(),
            avg_bars_in_trade=df['duration'].mean(),
            total_return=df['realized_pnl'].sum(),
            annualized_return=self._calculate_annualized_return(df),
            sharpe_ratio=self._calculate_sharpe_ratio(df),
            sortino_ratio=self._calculate_sortino_ratio(df),
            max_drawdown=self._calculate_max_drawdown(df),
            max_drawdown_duration=self._calculate_max_drawdown_duration(df),
            recovery_factor=self._calculate_recovery_factor(df),
            risk_adjusted_return=self._calculate_risk_adjusted_return(df)
        )
        
        return metrics

    def calculate_entry_metrics(self, trades: List[Dict]) -> EntryMetrics:
        """Calculate entry-specific metrics."""
        if not trades:
            return EntryMetrics()
        
        df = pd.DataFrame(trades)
        
        # Calculate entry efficiency
        df['entry_efficiency'] = df.apply(
            lambda x: self._calculate_entry_efficiency(
                x['entry_price'],
                x['exit_price'],
                x['position_type']
            ),
            axis=1
        )
        
        # Analyze entry patterns
        entry_patterns = self._analyze_entry_patterns(df)
        
        metrics = EntryMetrics(
            avg_entry_efficiency=df['entry_efficiency'].mean(),
            missed_entries=self._count_missed_entries(df),
            false_signals=self._count_false_signals(df),
            entry_time_distribution=self._analyze_entry_timing(df),
            avg_entry_slippage=df['entry_slippage'].mean() if 'entry_slippage' in df.columns else 0.0,
            successful_entry_patterns=entry_patterns['successful'],
            failed_entry_patterns=entry_patterns['failed'],
            avg_entry_range_position=self._calculate_avg_range_position(df),
            entry_volume_profile=self._analyze_entry_volume(df)
        )
        
        return metrics

    def calculate_range_metrics(self, trades: List[Dict], ranges: List[Dict]) -> RangeMetrics:
        """Calculate range-specific metrics."""
        if not trades or not ranges:
            return RangeMetrics()
        
        trades_df = pd.DataFrame(trades)
        ranges_df = pd.DataFrame(ranges)
        
        metrics = RangeMetrics(
            avg_range_size=ranges_df['range_size'].mean(),
            range_success_rate=self._calculate_range_success_rate(trades_df, ranges_df),
            range_failure_rate=self._calculate_range_failure_rate(trades_df, ranges_df),
            range_breakout_distribution=self._analyze_range_breakouts(ranges_df),
            range_volume_profile=self._analyze_range_volume(ranges_df),
            range_duration_stats=self._analyze_range_duration(ranges_df),
            range_quality_score=self._calculate_range_quality(ranges_df)
        )
        
        return metrics

    def _calculate_entry_efficiency(self, entry_price: float, exit_price: float, position_type: str) -> float:
        """Calculate entry efficiency score."""
        if position_type == "LONG":
            return (exit_price - entry_price) / entry_price
        return (entry_price - exit_price) / entry_price

    def _analyze_entry_patterns(self, trades_df: pd.DataFrame) -> Dict[str, Dict[str, int]]:
        """Analyze successful and failed entry patterns."""
        successful = trades_df[trades_df['realized_pnl'] > 0]['entry_pattern'].value_counts().to_dict()
        failed = trades_df[trades_df['realized_pnl'] <= 0]['entry_pattern'].value_counts().to_dict()
        return {'successful': successful, 'failed': failed}

    def _analyze_entry_timing(self, trades_df: pd.DataFrame) -> Dict[str, float]:
        """Analyze entry timing distribution."""
        trades_df['entry_hour'] = pd.to_datetime(trades_df['entry_time']).dt.hour
        return trades_df.groupby('entry_hour')['realized_pnl'].mean().to_dict()

    def _calculate_avg_range_position(self, trades_df: pd.DataFrame) -> float:
        """Calculate average position of entry within the range."""
        if 'range_high' not in trades_df.columns or 'range_low' not in trades_df.columns:
            return 0.0
        
        trades_df['range_position'] = (trades_df['entry_price'] - trades_df['range_low']) / \
                                    (trades_df['range_high'] - trades_df['range_low'])
        return trades_df['range_position'].mean()

    def _analyze_entry_volume(self, trades_df: pd.DataFrame) -> Dict[str, float]:
        """Analyze volume profile at entry points."""
        if 'entry_volume' not in trades_df.columns:
            return {}
        
        volume_profile = {
            'avg_volume': trades_df['entry_volume'].mean(),
            'high_volume_success_rate': len(trades_df[
                (trades_df['entry_volume'] > trades_df['entry_volume'].mean()) & 
                (trades_df['realized_pnl'] > 0)
            ]) / len(trades_df[trades_df['entry_volume'] > trades_df['entry_volume'].mean()]),
            'low_volume_success_rate': len(trades_df[
                (trades_df['entry_volume'] <= trades_df['entry_volume'].mean()) & 
                (trades_df['realized_pnl'] > 0)
            ]) / len(trades_df[trades_df['entry_volume'] <= trades_df['entry_volume'].mean()])
        }
        
        return volume_profile

    def _calculate_range_success_rate(self, trades_df: pd.DataFrame, ranges_df: pd.DataFrame) -> float:
        """Calculate success rate of range breakouts."""
        successful_breakouts = len(trades_df[trades_df['realized_pnl'] > 0])
        total_ranges = len(ranges_df)
        return successful_breakouts / total_ranges if total_ranges > 0 else 0.0

    def _analyze_range_breakouts(self, ranges_df: pd.DataFrame) -> Dict[str, float]:
        """Analyze distribution of range breakouts."""
        if 'breakout_direction' not in ranges_df.columns:
            return {}
        
        breakout_dist = ranges_df['breakout_direction'].value_counts(normalize=True).to_dict()
        return breakout_dist

    def _calculate_range_quality(self, ranges_df: pd.DataFrame) -> float:
        """Calculate quality score for ranges."""
        if not all(col in ranges_df.columns for col in ['range_size', 'volume', 'duration']):
            return 0.0
        
        # Normalize factors
        size_score = ranges_df['range_size'].rank(pct=True)
        volume_score = ranges_df['volume'].rank(pct=True)
        duration_score = ranges_df['duration'].rank(pct=True)
        
        # Calculate weighted average
        quality_score = (size_score * 0.4 + volume_score * 0.4 + duration_score * 0.2).mean()
        return quality_score

    def _calculate_sharpe_ratio(self, trades_df: pd.DataFrame) -> float:
        """Calculate Sharpe ratio."""
        if trades_df.empty:
            return 0.0
        
        returns = trades_df['realized_pnl'].pct_change()
        excess_returns = returns - (self.risk_free_rate / 252)  # Daily risk-free rate
        return np.sqrt(252) * (excess_returns.mean() / excess_returns.std()) if excess_returns.std() != 0 else 0.0

    def _calculate_sortino_ratio(self, trades_df: pd.DataFrame) -> float:
        """Calculate Sortino ratio."""
        if trades_df.empty:
            return 0.0
        
        returns = trades_df['realized_pnl'].pct_change()
        excess_returns = returns - (self.risk_free_rate / 252)
        downside_returns = excess_returns[excess_returns < 0]
        downside_std = downside_returns.std()
        
        return np.sqrt(252) * (excess_returns.mean() / downside_std) if downside_std != 0 else 0.0

    def _calculate_max_drawdown(self, trades_df: pd.DataFrame) -> float:
        """Calculate maximum drawdown."""
        if trades_df.empty:
            return 0.0
        
        cumulative_returns = trades_df['realized_pnl'].cumsum()
        rolling_max = cumulative_returns.expanding().max()
        drawdowns = cumulative_returns - rolling_max
        return abs(drawdowns.min())

    def generate_performance_report(self, trades: List[Dict], ranges: List[Dict]) -> Dict:
        """Generate comprehensive performance report."""
        base_metrics = self.calculate_base_metrics(trades)
        entry_metrics = self.calculate_entry_metrics(trades)
        range_metrics = self.calculate_range_metrics(trades, ranges)
        
        report = {
            "summary": {
                "total_trades": base_metrics.total_trades,
                "win_rate": base_metrics.win_rate,
                "profit_factor": base_metrics.profit_factor,
                "total_return": base_metrics.total_return,
                "max_drawdown": base_metrics.max_drawdown
            },
            "detailed_metrics": {
                "base": dataclasses.asdict(base_metrics),
                "entry": dataclasses.asdict(entry_metrics),
                "range": dataclasses.asdict(range_metrics)
            },
            "analysis": {
                "entry_patterns": entry_metrics.successful_entry_patterns,
                "range_quality": range_metrics.range_quality_score,
                "timing_efficiency": entry_metrics.avg_entry_efficiency
            },
            "recommendations": self._generate_recommendations(base_metrics, entry_metrics, range_metrics)
        }
        
        return report

    def _generate_recommendations(self, 
                                base_metrics: BaseMetrics,
                                entry_metrics: EntryMetrics,
                                range_metrics: RangeMetrics) -> List[str]:
        """Generate performance improvement recommendations."""
        recommendations = []
        
        # Win rate analysis
        if base_metrics.win_rate < 0.5:
            recommendations.append("Consider reviewing entry criteria to improve win rate")
        
        # Risk management
        if base_metrics.max_drawdown > base_metrics.total_return * 0.3:
            recommendations.append("Review position sizing and risk management rules")
        
        # Entry efficiency
        if entry_metrics.avg_entry_efficiency < 0.3:
            recommendations.append("Optimize entry timing to improve entry efficiency")
        
        # Range quality
        if range_metrics.range_quality_score < 0.5:
            recommendations.append("Focus on higher quality range formations")
        
        return recommendations
