# Stock Group Management Script

This directory contains scripts and tools for managing stock groups in the Setbull Trader application.

## Overview

The main script `add_stocks_to_group.py` processes stocks from a text file, looks them up in the NSE Upstox JSON file, creates them via the API, and adds them to a specified stock group.

## Files

- `add_stocks_to_group.py` - Main processing script
- `1_Buying_28008.txt` - Input file containing stock symbols (one per line)
- `requirements.txt` - Python dependencies
- `README.md` - This documentation file
- `stock_processing.log` - Generated log file (created during execution)

## Prerequisites

1. **Python 3.6+** installed on your system
2. **Setbull Trader API** running on `http://localhost:8083`
3. **NSE Upstox JSON file** available at `../../nse_upstox.json`

## Installation

1. Install Python dependencies:
   ```bash
   pip install -r requirements.txt
   ```

2. Ensure the Setbull Trader API is running:
   ```bash
   # Start your API server
   # The script expects it to be running on http://localhost:8083
   ```

## Configuration

The script uses the following configuration (defined in `add_stocks_to_group.py`):

```python
API_BASE_URL = "http://localhost:8083/api/v1"
STOCK_GROUP_ID = "d63c7112-2e10-443a-bb99-221fb9238e9e"
NSE_UPSTOX_JSON_PATH = "../../nse_upstox.json"
STOCKS_FILE_PATH = "1_Buying_28008.txt"
```

### Configuration Parameters

- **API_BASE_URL**: Base URL for the Setbull Trader API
- **STOCK_GROUP_ENTRY_TYPE**: Entry type for the stock group (e.g., "1ST_ENTRY")
- **NSE_UPSTOX_JSON_PATH**: Path to the NSE Upstox JSON file
- **STOCKS_FILE_PATH**: Path to the input file containing stock symbols

## Usage

### Basic Usage

Run the script from the command line:

```bash
python add_stocks_to_group.py
```

### Input File Format

The input file (`1_Buying_28008.txt`) should contain stock symbols, one per line:

```
MUFIN
KAJARIACER
RELIGARE
KUANTUM
SPMLINFRA
...
```

### API Endpoints Used

The script interacts with the following API endpoints:

1. **Create Stock**:
   ```bash
   POST /api/v1/stocks
   Content-Type: application/json
   
   {
     "symbol": "MUFIN",
     "name": "MUFIN GREEN FINANCE LTD",
     "isSelected": false
   }
   ```

2. **Get Stock by Symbol**:
   ```bash
   GET /api/v1/stocks?symbol=MUFIN
   ```

3. **Create Group with Stocks**:
   ```bash
   POST /api/v1/groups
   Content-Type: application/json
   
   {
     "entryType": "1ST_ENTRY",
     "stockIds": ["stock_id_1", "stock_id_2", ...]
   }
   ```

## How It Works

1. **Data Loading**: 
   - Loads the NSE Upstox JSON file containing all available stocks
   - Reads stock symbols from the input text file

2. **Stock Lookup**:
   - For each stock symbol, searches the NSE data for matching `trading_symbol`
   - Extracts the full stock name and other details

3. **Stock Creation**:
   - Attempts to create each stock via the API
   - If the stock already exists (409 status), fetches the existing stock ID
   - Handles errors gracefully and logs failures

4. **Group Creation**:
   - Collects all successfully created/found stock IDs
   - Creates a new group with all stocks in a single API call

5. **Logging and Reporting**:
   - Provides detailed logging throughout the process
   - Generates a summary report at the end
   - Creates a log file for debugging

## Output

### Console Output

The script provides real-time feedback:

```
2025-01-XX XX:XX:XX,XXX - INFO - Stock Group Management Script Started
2025-01-XX XX:XX:XX,XXX - INFO - Loaded 5000+ stocks from NSE Upstox JSON
2025-01-XX XX:XX:XX,XXX - INFO - Loaded 142 stocks from 1_Buying_28008.txt
2025-01-XX XX:XX:XX,XXX - INFO - Processing stock 1/142: MUFIN
2025-01-XX XX:XX:XX,XXX - INFO - Created stock: MUFIN (MUFIN GREEN FINANCE LTD) with ID: abc123
...
```

### Log File

A detailed log file (`stock_processing.log`) is created with all operations and any errors.

### Summary Report

At the end, a summary is displayed:

```
==================================================
PROCESSING SUMMARY
==================================================
Total stocks processed: 142
Successfully created: 140
Failed to create: 1
Skipped (not found in NSE data): 1
Failed stocks: INVALID_STOCK
Skipped stocks: UNKNOWN_STOCK
==================================================
```

## Error Handling

The script handles various error scenarios:

- **File not found**: Validates input files before processing
- **API errors**: Logs HTTP errors and continues with other stocks
- **Network timeouts**: Uses 30-second timeouts for API calls
- **Duplicate stocks**: Handles existing stocks gracefully
- **Missing NSE data**: Skips stocks not found in the JSON file

## Performance Considerations

- **Rate limiting**: 0.5-second delay between API calls to avoid overwhelming the server
- **Batch processing**: Adds all stocks to the group in a single API call
- **Memory efficient**: Processes stocks one at a time rather than loading all into memory

## Troubleshooting

### Common Issues

1. **API Connection Failed**:
   - Ensure the Setbull Trader API is running on `http://localhost:8083`
   - Check firewall settings and network connectivity

2. **File Not Found**:
   - Verify the NSE Upstox JSON file exists at the specified path
   - Check the input file path and permissions

3. **Permission Errors**:
   - Ensure write permissions for log file creation
   - Check file read permissions for input files

4. **Stock Not Found in NSE Data**:
   - Verify the stock symbol matches exactly with the `trading_symbol` field
   - Check if the stock is listed in the NSE Upstox JSON file

### Debug Mode

For detailed debugging, you can modify the logging level in the script:

```python
logging.basicConfig(level=logging.DEBUG, ...)
```

## API Response Examples

### Successful Stock Creation
```json
{
  "id": "abc123-def456-ghi789",
  "symbol": "MUFIN",
  "name": "MUFIN GREEN FINANCE LTD",
  "isSelected": false,
  "createdAt": "2025-01-XXTXX:XX:XXZ",
  "updatedAt": "2025-01-XXTXX:XX:XXZ"
}
```

### Stock Already Exists (409)
```json
{
  "error": "Stock already exists",
  "message": "A stock with symbol MUFIN already exists"
}
```

### Successful Group Creation
```json
{
  "success": true,
  "data": {
    "id": "new-group-uuid",
    "entryType": "1ST_ENTRY",
    "stockIds": ["abc123", "def456", "ghi789"],
    "createdAt": "2025-01-XXTXX:XX:XXZ"
  }
}
```

## Security Considerations

- The script uses HTTP (not HTTPS) for local development
- API endpoints should be secured in production environments
- Consider implementing authentication if required
- Validate all input data before processing

## Future Enhancements

Potential improvements for the script:

1. **Authentication**: Add API key or token authentication
2. **Batch Processing**: Process stocks in batches for better performance
3. **Retry Logic**: Implement exponential backoff for failed API calls
4. **Dry Run Mode**: Add option to preview changes without making them
5. **Configuration File**: Move configuration to a separate JSON/YAML file
6. **Progress Bar**: Add visual progress indicator for large datasets

## Support

For issues or questions:

1. Check the log file (`stock_processing.log`) for detailed error information
2. Verify all prerequisites are met
3. Test with a small subset of stocks first
4. Ensure the API server is running and accessible

## License

This script is part of the Setbull Trader project and follows the same licensing terms. 