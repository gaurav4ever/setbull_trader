"""
Risk Calculator for Morning Range Strategy.

This module handles risk calculations, R-multiple analysis, and risk metrics
for the Morning Range trading strategy.
"""

from dataclasses import dataclass
from enum import Enum
from typing import Dict, Optional, Union, List, Tuple
import logging
from datetime import datetime, time
import numpy as np
import pandas as pd

logger = logging.getLogger(__name__)

class RiskLevel(Enum):
    """Risk exposure levels."""
    LOW = "low"
    MODERATE = "moderate"
    HIGH = "high"
    CRITICAL = "critical"

@dataclass
class RiskConfig:
    """Configuration for risk calculations."""
    max_risk_per_trade: float  # Maximum risk per trade (%)
    max_daily_risk: float  # Maximum daily risk exposure (%)
    max_correlated_risk: float  # Maximum risk for correlated positions (%)
    position_size_limit: float  # Maximum position size (%)
    max_drawdown_limit: float  # Maximum drawdown limit (%)
    risk_free_rate: float = 0.05  # Annual risk-free rate
    correlation_threshold: float = 0.7  # Correlation threshold for risk grouping

@dataclass
class RiskMetrics:
    """Risk metrics for a trade or portfolio."""
    r_multiple: float = 0.0
    risk_reward_ratio: float = 0.0
    sharpe_ratio: float = 0.0
    sortino_ratio: float = 0.0
    max_drawdown: float = 0.0
    risk_adjusted_return: float = 0.0
    win_rate: float = 0.0
    profit_factor: float = 0.0
    expected_value: float = 0.0

class RiskCalculator:
    """Calculator for risk metrics and analysis."""
    
    def __init__(self, risk_config: RiskConfig):
        """
        Initialize the Risk Calculator.
        
        Args:
            risk_config: Risk calculation configuration
        """
        self.config = risk_config
        self.trade_history: List[Dict] = []
        self.daily_risk_tracker: Dict[str, float] = {}  # Date -> risk exposure
        self.correlation_matrix: Dict[str, Dict[str, float]] = {}  # Instrument correlations
        
        logger.info(f"Initialized RiskCalculator with config: {risk_config}")

    def calculate_r_multiple(self, 
                           entry_price: float,
                           exit_price: float,
                           stop_loss: float,
                           position_type: str) -> float:
        """
        Calculate R-multiple for a trade.
        
        Args:
            entry_price: Entry price
            exit_price: Exit price
            stop_loss: Stop loss price
            position_type: Type of position (LONG/SHORT)
            
        Returns:
            float: R-multiple value
        """
        if position_type == "LONG":
            risk = entry_price - stop_loss
            reward = exit_price - entry_price
        else:  # SHORT
            risk = stop_loss - entry_price
            reward = entry_price - exit_price
        
        if risk <= 0:
            logger.warning("Invalid risk amount (zero or negative)")
            return 0.0
        
        r_multiple = reward / risk
        logger.debug(f"Calculated R-multiple: {r_multiple}")
        return round(r_multiple, 2)

    def calculate_risk_reward_ratio(self,
                                  entry_price: float,
                                  target_price: float,
                                  stop_loss: float,
                                  position_type: str) -> float:
        """Calculate risk-to-reward ratio for a trade setup."""
        if position_type == "LONG":
            risk = entry_price - stop_loss
            reward = target_price - entry_price
        else:  # SHORT
            risk = stop_loss - entry_price
            reward = entry_price - target_price
        
        if risk <= 0:
            logger.warning("Invalid risk amount (zero or negative)")
            return 0.0
        
        rr_ratio = reward / risk
        logger.debug(f"Calculated risk-reward ratio: {rr_ratio}")
        return round(rr_ratio, 2)

    def calculate_position_risk(self,
                              position_size: float,
                              entry_price: float,
                              stop_loss: float,
                              account_size: float) -> Dict[str, float]:
        """Calculate risk metrics for a position."""
        risk_per_share = abs(entry_price - stop_loss)
        total_risk_amount = risk_per_share * position_size
        risk_percentage = (total_risk_amount / account_size) * 100
        
        risk_metrics = {
            "risk_per_share": round(risk_per_share, 2),
            "total_risk_amount": round(total_risk_amount, 2),
            "risk_percentage": round(risk_percentage, 2),
            "position_value": round(position_size * entry_price, 2),
            "position_percentage": round((position_size * entry_price / account_size) * 100, 2)
        }
        
        logger.debug(f"Position risk metrics: {risk_metrics}")
        return risk_metrics

    def validate_position_risk(self,
                             position_risk: Dict[str, float],
                             instrument_key: str,
                             trade_date: datetime) -> Tuple[bool, str]:
        """Validate if position risk is within acceptable limits."""
        # Check individual position risk
        if position_risk["risk_percentage"] > self.config.max_risk_per_trade:
            return False, f"Position risk {position_risk['risk_percentage']}% exceeds maximum {self.config.max_risk_per_trade}%"
        
        # Check position size
        if position_risk["position_percentage"] > self.config.position_size_limit:
            return False, f"Position size {position_risk['position_percentage']}% exceeds maximum {self.config.position_size_limit}%"
        
        # Check daily risk exposure
        date_key = trade_date.strftime("%Y-%m-%d")
        current_daily_risk = self.daily_risk_tracker.get(date_key, 0.0)
        total_daily_risk = current_daily_risk + position_risk["risk_percentage"]
        
        if total_daily_risk > self.config.max_daily_risk:
            return False, f"Daily risk exposure {total_daily_risk}% would exceed maximum {self.config.max_daily_risk}%"
        
        # Check correlated risk
        if not self._validate_correlated_risk(instrument_key, position_risk["risk_percentage"]):
            return False, "Correlated position risk would exceed maximum"
        
        return True, "Position risk within acceptable limits"

    def _validate_correlated_risk(self, instrument_key: str, additional_risk: float) -> bool:
        """Validate risk considering correlated positions."""
        if instrument_key not in self.correlation_matrix:
            return True
        
        correlated_instruments = [
            inst for inst, corr in self.correlation_matrix[instrument_key].items()
            if corr >= self.config.correlation_threshold
        ]
        
        total_correlated_risk = additional_risk + sum(
            self._get_active_position_risk(inst)
            for inst in correlated_instruments
        )
        
        return total_correlated_risk <= self.config.max_correlated_risk

    def _get_active_position_risk(self, instrument_key: str) -> float:
        """Get current risk exposure for an instrument."""
        # This would need to be implemented based on your position tracking system
        return 0.0  # Placeholder

    def calculate_portfolio_metrics(self, 
                                  trades: List[Dict],
                                  period_days: int = 252) -> RiskMetrics:
        """Calculate comprehensive portfolio risk metrics."""
        if not trades:
            return RiskMetrics()
        
        # Convert trades to DataFrame for analysis
        df = pd.DataFrame(trades)
        df['return'] = df['realized_pnl'] / df['risk_amount']
        
        # Calculate basic metrics
        total_trades = len(trades)
        winning_trades = len(df[df['realized_pnl'] > 0])
        total_profit = df[df['realized_pnl'] > 0]['realized_pnl'].sum()
        total_loss = abs(df[df['realized_pnl'] <= 0]['realized_pnl'].sum())
        
        # Calculate advanced metrics
        returns = df['return'].values
        excess_returns = returns - (self.config.risk_free_rate / period_days)
        
        metrics = RiskMetrics(
            r_multiple=np.mean(df['r_multiple']),
            risk_reward_ratio=np.mean(df['risk_reward_ratio']),
            sharpe_ratio=self._calculate_sharpe_ratio(excess_returns),
            sortino_ratio=self._calculate_sortino_ratio(excess_returns),
            max_drawdown=self._calculate_max_drawdown(df['realized_pnl'].cumsum()),
            risk_adjusted_return=self._calculate_risk_adjusted_return(returns),
            win_rate=winning_trades / total_trades * 100,
            profit_factor=total_profit / total_loss if total_loss > 0 else float('inf'),
            expected_value=np.mean(returns)
        )
        
        logger.info(f"Portfolio metrics: {metrics}")
        return metrics

    def _calculate_sharpe_ratio(self, excess_returns: np.ndarray) -> float:
        """Calculate Sharpe ratio."""
        if len(excess_returns) < 2:
            return 0.0
        return np.mean(excess_returns) / np.std(excess_returns, ddof=1) * np.sqrt(252)

    def _calculate_sortino_ratio(self, excess_returns: np.ndarray) -> float:
        """Calculate Sortino ratio."""
        if len(excess_returns) < 2:
            return 0.0
        negative_returns = excess_returns[excess_returns < 0]
        if len(negative_returns) == 0:
            return float('inf')
        downside_std = np.std(negative_returns, ddof=1)
        return np.mean(excess_returns) / downside_std * np.sqrt(252)

    def _calculate_max_drawdown(self, cumulative_returns: np.ndarray) -> float:
        """Calculate maximum drawdown."""
        rolling_max = np.maximum.accumulate(cumulative_returns)
        drawdowns = cumulative_returns - rolling_max
        return abs(min(drawdowns, default=0.0))

    def _calculate_risk_adjusted_return(self, returns: np.ndarray) -> float:
        """Calculate risk-adjusted return."""
        if len(returns) < 2:
            return 0.0
        return np.mean(returns) / np.std(returns, ddof=1)

    def get_risk_exposure_level(self, metrics: RiskMetrics) -> RiskLevel:
        """Determine current risk exposure level."""
        if metrics.max_drawdown >= self.config.max_drawdown_limit:
            return RiskLevel.CRITICAL
        
        if metrics.max_drawdown >= self.config.max_drawdown_limit * 0.8:
            return RiskLevel.HIGH
        
        if metrics.max_drawdown >= self.config.max_drawdown_limit * 0.5:
            return RiskLevel.MODERATE
        
        return RiskLevel.LOW

    def generate_risk_report(self, trades: List[Dict]) -> Dict:
        """Generate comprehensive risk report."""
        metrics = self.calculate_portfolio_metrics(trades)
        risk_level = self.get_risk_exposure_level(metrics)
        
        report = {
            "risk_level": risk_level.value,
            "metrics": {
                "r_multiple": round(metrics.r_multiple, 2),
                "risk_reward_ratio": round(metrics.risk_reward_ratio, 2),
                "sharpe_ratio": round(metrics.sharpe_ratio, 2),
                "sortino_ratio": round(metrics.sortino_ratio, 2),
                "max_drawdown": round(metrics.max_drawdown, 2),
                "risk_adjusted_return": round(metrics.risk_adjusted_return, 2),
                "win_rate": round(metrics.win_rate, 2),
                "profit_factor": round(metrics.profit_factor, 2),
                "expected_value": round(metrics.expected_value, 2)
            },
            "risk_exposure": {
                "daily_risk": self.daily_risk_tracker,
                "position_correlations": self.correlation_matrix
            },
            "recommendations": self._generate_risk_recommendations(metrics, risk_level)
        }
        
        logger.info(f"Generated risk report: {report}")
        return report

    def _generate_risk_recommendations(self, 
                                     metrics: RiskMetrics,
                                     risk_level: RiskLevel) -> List[str]:
        """Generate risk management recommendations."""
        recommendations = []
        
        if risk_level == RiskLevel.CRITICAL:
            recommendations.append("Immediately reduce position sizes and risk exposure")
            recommendations.append("Consider closing underperforming positions")
        
        if metrics.win_rate < 50:
            recommendations.append("Review entry criteria and trade validation")
        
        if metrics.profit_factor < 1.5:
            recommendations.append("Optimize risk-reward ratios and exit strategies")
        
        if metrics.max_drawdown > self.config.max_drawdown_limit * 0.7:
            recommendations.append("Implement stricter drawdown controls")
        
        return recommendations 