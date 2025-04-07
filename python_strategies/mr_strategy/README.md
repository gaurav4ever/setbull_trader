# Morning Range Strategy

A Python implementation of the Morning Range trading strategy for Indian equities.

## Overview

This package implements a systematic trading strategy based on morning range breakouts. The strategy identifies trading opportunities by tracking price breakouts above or below the initial morning range (either first 5 minutes or 15 minutes of trading).

## Key Features

- Morning range calculation and validation
- Trend-following or counter-trend trading options
- Risk-based position sizing
- Multi-level take profit targets
- Backtesting framework
- Performance analytics
- Interactive dashboard

## Directory Structure

- `config/` - Configuration settings
- `data/` - Data fetching and processing
- `strategy/` - Core strategy components
- `utils/` - Utility functions
- `backtest/` - Backtesting framework
- `dashboard/` - Visualization dashboard

## Integration

This package integrates with the existing Go backend API to:
- Fetch stock data
- Get pre-filtered stock candidates
- Execute trading signals

## Requirements

- Python 3.8+
- See requirements.txt for package dependencies