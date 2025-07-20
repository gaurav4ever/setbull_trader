#!/bin/bash

# Script to fetch historical data in batches of 4 days
# Date range: 2025-06-01 to 2025-07-18

# TODO: Extend this to multiple stocks

API_URL="http://localhost:8083/api/v1/historical-data/batch-store"
INSTRUMENT_KEYS='["NSE_EQ|INE348A01023"]'
INTERVAL="1minute"

# Function to format date as YYYY-MM-DD
format_date() {
    date -d "$1" +%Y-%m-%d 2>/dev/null || date -j -f "%Y-%m-%d" "$1" +%Y-%m-%d 2>/dev/null || echo "$1"
}

# Function to add days to a date
add_days() {
    local date_str=$1
    local days=$2
    date -d "$date_str + $days days" +%Y-%m-%d 2>/dev/null || date -j -v+${days}d -f "%Y-%m-%d" "$date_str" +%Y-%m-%d 2>/dev/null || echo "$date_str"
}

# Start date
start_date="2025-06-01"
end_date="2025-07-18"

echo "Starting batch historical data fetch..."
echo "Date range: $start_date to $end_date"
echo "Batch size: 4 days"
echo "Instrument: $INSTRUMENT_KEYS"
echo "Interval: $INTERVAL"
echo "----------------------------------------"

current_date="$start_date"
batch_count=1

while [ "$current_date" \< "$end_date" ]; do
    # Calculate end date for this batch (4 days later)
    batch_end_date=$(add_days "$current_date" 3)
    
    # Ensure we don't exceed the overall end date
    if [ "$batch_end_date" \> "$end_date" ]; then
        batch_end_date="$end_date"
    fi
    
    echo "Batch $batch_count: $current_date to $batch_end_date"
    
    # Prepare the JSON payload
    json_data="{
        \"instrumentKeys\": $INSTRUMENT_KEYS,
        \"fromDate\": \"$current_date\",
        \"toDate\": \"$batch_end_date\",
        \"interval\": \"$INTERVAL\"
    }"
    
    echo "Sending request..."
    echo "Payload: $json_data"
    
    # Make the API call
    response=$(curl --location "$API_URL" \
        --header 'Content-Type: application/json' \
        --data "$json_data" \
        --silent \
        --show-error)
    
    # Check if the request was successful
    if [ $? -eq 0 ]; then
        echo "✅ Success: $response"
    else
        echo "❌ Error: $response"
    fi
    
    echo "----------------------------------------"
    
    # Move to next batch (start date + 4 days)
    current_date=$(add_days "$current_date" 4)
    batch_count=$((batch_count + 1))
    
    # Add a small delay between requests to avoid overwhelming the server
    sleep 1
done

echo "Batch historical data fetch completed!" 