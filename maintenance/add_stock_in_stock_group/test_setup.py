#!/usr/bin/env python3
"""
Setup Test Script

This script validates the environment and configuration before running the main
stock processing script.

Usage:
    python test_setup.py
"""

import json
import requests
import sys
from pathlib import Path

# Configuration (same as main script)
API_BASE_URL = "http://localhost:8083/api/v1"
STOCK_GROUP_ID = "d63c7112-2e10-443a-bb99-221fb9238e9e"
NSE_UPSTOX_JSON_PATH = "../../nse_upstox.json"
STOCKS_FILE_PATH = "1_Buying_28008.txt"


def test_file_paths():
    """Test if required files exist"""
    print("Testing file paths...")
    
    # Test NSE Upstox JSON file
    if Path(NSE_UPSTOX_JSON_PATH).exists():
        print(f"✓ NSE Upstox JSON file found: {NSE_UPSTOX_JSON_PATH}")
        
        # Test if it's valid JSON
        try:
            with open(NSE_UPSTOX_JSON_PATH, 'r') as f:
                data = json.load(f)
            print(f"✓ NSE Upstox JSON is valid (contains {len(data)} stocks)")
        except json.JSONDecodeError as e:
            print(f"✗ NSE Upstox JSON is invalid: {e}")
            return False
    else:
        print(f"✗ NSE Upstox JSON file not found: {NSE_UPSTOX_JSON_PATH}")
        return False
    
    # Test stocks file
    if Path(STOCKS_FILE_PATH).exists():
        print(f"✓ Stocks file found: {STOCKS_FILE_PATH}")
        
        # Count lines
        with open(STOCKS_FILE_PATH, 'r') as f:
            lines = [line.strip() for line in f if line.strip()]
        print(f"✓ Stocks file contains {len(lines)} stock symbols")
    else:
        print(f"✗ Stocks file not found: {STOCKS_FILE_PATH}")
        return False
    
    return True


def test_api_connectivity():
    """Test API connectivity and endpoints"""
    print("\nTesting API connectivity...")
    
    # Test basic connectivity
    try:
        response = requests.get(f"{API_BASE_URL}/health", timeout=10)
        if response.status_code == 200:
            print("✓ API is accessible")
        else:
            print(f"⚠ API responded with status {response.status_code}")
    except requests.exceptions.RequestException as e:
        print(f"✗ Cannot connect to API: {e}")
        return False
    
    # Test stocks endpoint
    try:
        response = requests.get(f"{API_BASE_URL}/stocks", timeout=10)
        if response.status_code == 200:
            print("✓ Stocks endpoint is accessible")
        else:
            print(f"⚠ Stocks endpoint responded with status {response.status_code}")
    except requests.exceptions.RequestException as e:
        print(f"✗ Cannot access stocks endpoint: {e}")
        return False
    
    # Test groups endpoint
    try:
        response = requests.get(f"{API_BASE_URL}/groups", timeout=10)
        if response.status_code == 200:
            print("✓ Groups endpoint is accessible")
        else:
            print(f"⚠ Groups endpoint responded with status {response.status_code}")
    except requests.exceptions.RequestException as e:
        print(f"✗ Cannot access groups endpoint: {e}")
        return False
    
    return True


def test_sample_stock_lookup():
    """Test stock lookup functionality"""
    print("\nTesting stock lookup...")
    
    # Load NSE data
    try:
        with open(NSE_UPSTOX_JSON_PATH, 'r') as f:
            nse_data = json.load(f)
    except Exception as e:
        print(f"✗ Failed to load NSE data: {e}")
        return False
    
    # Load sample stock
    try:
        with open(STOCKS_FILE_PATH, 'r') as f:
            sample_stock = f.readline().strip()
    except Exception as e:
        print(f"✗ Failed to read sample stock: {e}")
        return False
    
    if not sample_stock:
        print("✗ No sample stock found in file")
        return False
    
    # Find stock in NSE data
    found_stock = None
    for stock in nse_data:
        if stock.get('trading_symbol') == sample_stock:
            found_stock = stock
            break
    
    if found_stock:
        print(f"✓ Sample stock '{sample_stock}' found in NSE data")
        print(f"  Name: {found_stock.get('name')}")
        print(f"  ISIN: {found_stock.get('isin')}")
        print(f"  Instrument Key: {found_stock.get('instrument_key')}")
    else:
        print(f"✗ Sample stock '{sample_stock}' not found in NSE data")
        return False
    
    return True


def test_api_endpoints():
    """Test specific API endpoints"""
    print("\nTesting API endpoints...")
    
    # Test stock creation endpoint (with a test stock)
    test_payload = {
        "symbol": "TEST_STOCK",
        "name": "TEST STOCK FOR VALIDATION",
        "isSelected": False
    }
    
    try:
        response = requests.post(f"{API_BASE_URL}/stocks", json=test_payload, timeout=10)
        if response.status_code in [201, 409]:  # Created or already exists
            print("✓ Stock creation endpoint is working")
            
            # Clean up test stock if it was created
            if response.status_code == 201:
                stock_data = response.json()
                stock_id = stock_data.get('id')
                if stock_id:
                    # Try to delete the test stock (if delete endpoint exists)
                    try:
                        requests.delete(f"{API_BASE_URL}/stocks/{stock_id}", timeout=10)
                    except:
                        pass  # Ignore cleanup errors
        else:
            print(f"⚠ Stock creation endpoint responded with status {response.status_code}")
    except requests.exceptions.RequestException as e:
        print(f"✗ Cannot test stock creation endpoint: {e}")
        return False
    
    # Test group creation endpoint
    test_group_payload = {
        "entryType": "1ST_ENTRY",
        "stockIds": []
    }
    
    try:
        response = requests.post(f"{API_BASE_URL}/groups", 
                              json=test_group_payload, timeout=10)
        if response.status_code in [201, 400]:  # Success or validation error
            print("✓ Group creation endpoint is working")
        else:
            print(f"⚠ Group creation endpoint responded with status {response.status_code}")
    except requests.exceptions.RequestException as e:
        print(f"✗ Cannot test group creation endpoint: {e}")
        return False
    
    return True


def main():
    """Main test function"""
    print("=" * 60)
    print("SETUP VALIDATION TEST")
    print("=" * 60)
    
    all_tests_passed = True
    
    # Run all tests
    if not test_file_paths():
        all_tests_passed = False
    
    if not test_api_connectivity():
        all_tests_passed = False
    
    if not test_sample_stock_lookup():
        all_tests_passed = False
    
    if not test_api_endpoints():
        all_tests_passed = False
    
    # Summary
    print("\n" + "=" * 60)
    print("TEST SUMMARY")
    print("=" * 60)
    
    if all_tests_passed:
        print("✓ All tests passed! You can now run the main script.")
        print("\nNext steps:")
        print("1. Ensure your API server is running")
        print("2. Run: python add_stocks_to_group.py")
        return True
    else:
        print("✗ Some tests failed. Please fix the issues before running the main script.")
        print("\nCommon fixes:")
        print("1. Start the Setbull Trader API server")
        print("2. Check file paths and permissions")
        print("3. Verify network connectivity")
        print("4. Ensure all dependencies are installed")
        return False


if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1) 