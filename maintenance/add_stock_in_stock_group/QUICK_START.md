# Quick Start Guide

## 🚀 Get Started in 3 Steps

### 1. Install Dependencies
```bash
pip install -r requirements.txt
```

### 2. Test Your Setup
```bash
python test_setup.py
```

### 3. Run the Script
```bash
python add_stocks_to_group.py
```

## 📁 What's Included

- **`add_stocks_to_group.py`** - Main script that processes your stocks
- **`test_setup.py`** - Validates your environment before running
- **`config.json`** - Configuration file (modify settings here)
- **`run.sh`** - Interactive menu script
- **`README.md`** - Complete documentation
- **`requirements.txt`** - Python dependencies

## ⚙️ Configuration

Edit `config.json` to customize:

```json
{
  "api": {
    "base_url": "http://localhost:8083/api/v1"
  },
  "stock_group": {
    "entry_type": "1ST_ENTRY"
  }
}
```

## 📊 What the Script Does

1. **Reads** stock symbols from `1_Buying_28008.txt`
2. **Looks up** each stock in `nse_upstox.json`
3. **Creates** stocks via API (or gets existing IDs)
4. **Creates** a new group with all stocks
5. **Logs** everything to `stock_processing.log`

## 🔧 Troubleshooting

### API Not Running?
```bash
# Start your Setbull Trader API server first
# Then run: python test_setup.py
```

### File Not Found?
```bash
# Check if these files exist:
ls -la ../../nse_upstox.json
ls -la 1_Buying_28008.txt
```

### Permission Issues?
```bash
# Make sure you have read/write permissions
chmod +x run.sh
```

## 📈 Expected Output

```
2025-01-XX XX:XX:XX,XXX - INFO - Stock Group Management Script Started
2025-01-XX XX:XX:XX,XXX - INFO - Loaded 5000+ stocks from NSE Upstox JSON
2025-01-XX XX:XX:XX,XXX - INFO - Loaded 142 stocks from 1_Buying_28008.txt
2025-01-XX XX:XX:XX,XXX - INFO - Processing stock 1/142: MUFIN
2025-01-XX XX:XX:XX,XXX - INFO - Created stock: MUFIN (MUFIN GREEN FINANCE LTD) with ID: abc123
...
==================================================
PROCESSING SUMMARY
==================================================
Total stocks processed: 142
Successfully created: 140
Failed to create: 1
Skipped (not found in NSE data): 1
==================================================
```

## 🎯 Interactive Mode

Use the interactive menu:
```bash
./run.sh
```

This gives you options to:
- Install dependencies
- Run setup test
- Run main script
- Run complete workflow

## 📞 Need Help?

1. Check the log file: `stock_processing.log`
2. Run the setup test: `python test_setup.py`
3. Read the full documentation: `README.md` 