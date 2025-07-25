#!/usr/bin/env python3

import json
import time
import requests
from typing import List, Dict

def load_backtest_request(file_path: str) -> Dict:
    """Load the backtest request from JSON file."""
    with open(file_path, 'r') as f:
        return json.load(f)

def split_instruments_into_groups(instruments: List[Dict], group_size: int = 4) -> List[List[Dict]]:
    """Split instruments into groups of specified size."""
    groups = []
    for i in range(0, len(instruments), group_size):
        groups.append(instruments[i:i + group_size])
    return groups

def create_request_payload(runner_config: Dict, instrument_group: List[Dict]) -> Dict:
    """Create the request payload for a group of instruments."""
    return {
        "runner_config": runner_config,
        "instrument_configs": instrument_group
    }

def call_backtest_api(payload: Dict, group_number: int, total_groups: int) -> bool:
    """Call the backtest API with the given payload."""
    url = "http://localhost:3000/backtest/run/single"
    headers = {"Content-Type": "application/json"}
    
    try:
        print(f"Making API call for group {group_number}/{total_groups}...")
        print(f"Symbols in this group: {[inst['name'] for inst in payload['instrument_configs']]}")
        
        response = requests.post(url, headers=headers, json=payload, timeout=30)
        
        if response.status_code == 200:
            print(f"✅ Group {group_number} successful - Status: {response.status_code}")
            try:
                result = response.json()
                print(f"   Response: {result}")
            except:
                print(f"   Response: {response.text[:200]}...")
        else:
            print(f"❌ Group {group_number} failed - Status: {response.status_code}")
            print(f"   Error: {response.text}")
            return False
            
    except requests.exceptions.RequestException as e:
        print(f"❌ Group {group_number} failed - Network error: {e}")
        return False
    
    return True

def main():
    # Load the backtest request
    backtest_data = load_backtest_request('maintenance/backtest_request.json')
    
    # Extract runner config and instruments
    runner_config = backtest_data['runner_config']
    all_instruments = backtest_data['instrument_configs']
    
    print(f"Total instruments: {len(all_instruments)}")
    
    # Split instruments into groups of 4
    instrument_groups = split_instruments_into_groups(all_instruments, 4)
    
    print(f"Created {len(instrument_groups)} groups of 4 instruments each")
    print("=" * 80)
    
    # Process each group
    successful_calls = 0
    failed_calls = 0
    
    for i, group in enumerate(instrument_groups, 1):
        # Create payload for this group
        payload = create_request_payload(runner_config, group)
        
        # Make API call
        success = call_backtest_api(payload, i, len(instrument_groups))
        
        if success:
            successful_calls += 1
        else:
            failed_calls += 1
        
        # Wait 5 seconds before next call (except for the last one)
        if i < len(instrument_groups):
            print(f"⏳ Waiting 5 seconds before next group...")
            time.sleep(5)
        
        print("-" * 80)
    
    # Summary
    print("=" * 80)
    print("SUMMARY:")
    print(f"Total groups processed: {len(instrument_groups)}")
    print(f"Successful calls: {successful_calls}")
    print(f"Failed calls: {failed_calls}")
    print(f"Success rate: {(successful_calls/len(instrument_groups)*100):.1f}%")

if __name__ == "__main__":
    main() 