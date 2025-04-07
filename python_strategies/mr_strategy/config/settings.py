# python_strategies/mr_strategy/config/settings.py
import os
from dotenv import load_dotenv

# Load environment variables from .env file if it exists
load_dotenv()

# API Configuration
API_BASE_URL = os.getenv("API_BASE_URL", "http://localhost:8080/api/v1")
API_TIMEOUT = int(os.getenv("API_TIMEOUT", "30"))

# Strategy Parameters
RANGE_TYPE = os.getenv("MR_RANGE_TYPE", "5MR")  # "5MR" or "15MR"
RESPECT_TREND = os.getenv("MR_RESPECT_TREND", "true").lower() == "true"
RISK_AMOUNT = float(os.getenv("MR_RISK_AMOUNT", "30.0"))
STOP_LOSS_PERCENT = float(os.getenv("MR_STOP_LOSS_PERCENT", "0.75"))

# Take Profit Configuration
TP1_RR = float(os.getenv("MR_TP1_RR", "3.0"))  # Risk:Reward for TP1
TP2_RR = float(os.getenv("MR_TP2_RR", "5.0"))  # Risk:Reward for TP2
TP3_RR = float(os.getenv("MR_TP3_RR", "7.0"))  # Risk:Reward for TP3
TP1_SIZE = float(os.getenv("MR_TP1_SIZE", "10.0"))  # % of position to exit at TP1
TP2_SIZE = float(os.getenv("MR_TP2_SIZE", "40.0"))  # % of position to exit at TP2

# Entry Parameters
TICK_SIZE = float(os.getenv("MR_TICK_SIZE", "0.01"))
TICK_BUFFER = int(os.getenv("MR_TICK_BUFFER", "5"))
COMMISSION_PER_SHARE = float(os.getenv("MR_COMMISSION_PER_SHARE", "0.0"))

# ATR Configuration
ATR_LENGTH = int(os.getenv("MR_ATR_LENGTH", "14"))
ATR_TO_MR_RATIO_THRESHOLD = float(os.getenv("MR_ATR_TO_MR_RATIO_THRESHOLD", "3.0"))

# Time Configuration
MARKET_OPEN_HOUR = int(os.getenv("MARKET_OPEN_HOUR", "9"))
MARKET_OPEN_MINUTE = int(os.getenv("MARKET_OPEN_MINUTE", "15"))
MARKET_CLOSE_HOUR = int(os.getenv("MARKET_CLOSE_HOUR", "15"))
MARKET_CLOSE_MINUTE = int(os.getenv("MARKET_CLOSE_MINUTE", "30"))

# Morning Range Time Configuration
MR5_START_HOUR = int(os.getenv("MR5_START_HOUR", "9"))
MR5_START_MINUTE = int(os.getenv("MR5_START_MINUTE", "15"))
MR5_END_HOUR = int(os.getenv("MR5_END_HOUR", "9"))
MR5_END_MINUTE = int(os.getenv("MR5_END_MINUTE", "20"))

MR15_START_HOUR = int(os.getenv("MR15_START_HOUR", "9"))
MR15_START_MINUTE = int(os.getenv("MR15_START_MINUTE", "15"))
MR15_END_HOUR = int(os.getenv("MR15_END_HOUR", "9"))
MR15_END_MINUTE = int(os.getenv("MR15_END_MINUTE", "30"))

# Debug and Logging
DEBUG = os.getenv("MR_DEBUG", "false").lower() == "true"
LOG_LEVEL = os.getenv("MR_LOG_LEVEL", "INFO")