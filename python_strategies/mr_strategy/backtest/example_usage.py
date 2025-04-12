# Create simulation configuration
market_impact_config = MarketImpactConfig(
    slippage_percentage=0.01,
    volume_impact_factor=0.1,
    spread_percentage=0.02,
    tick_size=0.05,
    min_volume=100,
    max_position_volume=0.1
)

sim_config = SimulationConfig(
    market_impact=market_impact_config,
    time_in_force=30,
    enable_partial_fills=True,
    enable_market_impact=True,
    enable_volume_validation=True,
    replay_speed=1
)

# Create simulator
simulator = BacktestSimulator(
    config=sim_config,
    position_manager=position_manager,
    trade_manager=trade_manager,
    risk_calculator=risk_calculator
)

# Process candles
for candle in candles:
    executions = simulator.process_candle(strategy, candle)
    
    # Check execution results
    for execution in executions:
        if execution["status"] == "filled":
            print(f"Trade executed at {execution['executed_price']} with slippage {execution['slippage']}")
        else:
            print(f"Trade rejected: {execution['reason']}")

# Get simulation metrics
metrics = simulator.get_simulation_metrics()
print(f"Simulation metrics: {metrics}")

# Initialize analyzer
analyzer = PerformanceAnalyzer(risk_free_rate=0.05)

# Calculate metrics
base_metrics = analyzer.calculate_base_metrics(trades)
entry_metrics = analyzer.calculate_entry_metrics(trades)
range_metrics = analyzer.calculate_range_metrics(trades, ranges)

# Generate comprehensive report
report = analyzer.generate_performance_report(trades, ranges)

# Print summary
print(f"Strategy Performance Summary:")
print(f"Total Trades: {report['summary']['total_trades']}")
print(f"Win Rate: {report['summary']['win_rate']:.2%}")
print(f"Profit Factor: {report['summary']['profit_factor']:.2f}")
print(f"\nRecommendations:")
for rec in report['recommendations']:
    print(f"- {rec}")

# Create backtest run configuration
run_config = BacktestRunConfig(
    mode=BacktestMode.BATCH,
    start_date=datetime(2024, 1, 1, tzinfo=pytz.UTC),
    end_date=datetime(2024, 3, 1, tzinfo=pytz.UTC),
    instruments=["NSE_EQ|INE123456789"],
    strategies=[
        {
            "type": "MorningRange",
            "params": {
                "range_type": "5MR",
                "entry_type": "1ST_ENTRY",
                "sl_percentage": 0.5,
                "target_r": 2.0
            }
        }
    ],
    initial_capital=100000.0,
    batch_size=50,
    parallel_runs=4,
    output_dir="backtest_results"
)

# Create and run backtest runner
runner = BacktestRunner(run_config)
results = await runner.run_backtests()

# Access results and reports
print(f"Backtest Summary:")
print(f"Total Trades: {results['summary']['total_trades']}")
print(f"Win Rate: {results['summary']['win_rate']:.2%}")
print(f"Total Return: {results['summary']['total_return']:.2f}")

# Check recommendations
print("\nRecommendations:")
for rec in runner.reports[run_config.mode.value]["recommendations"]:
    print(f"- {rec}")
