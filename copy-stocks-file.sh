#!/bin/bash
# Script to copy the nse_stocks.txt file to the static folder

# Path to the source file
SOURCE_FILE="nse_stocks.txt"

# Path to the destination in the static folder
DEST_FOLDER="frontend/static"

# Check if source file exists
if [ ! -f "$SOURCE_FILE" ]; then
  echo "Error: Source file $SOURCE_FILE not found"
  exit 1
fi

# Check if destination folder exists
if [ ! -d "$DEST_FOLDER" ]; then
  echo "Creating destination folder $DEST_FOLDER"
  mkdir -p "$DEST_FOLDER"
fi

# Copy the file
cp "$SOURCE_FILE" "$DEST_FOLDER"

# Check if the copy was successful
if [ $? -eq 0 ]; then
  echo "Successfully copied $SOURCE_FILE to $DEST_FOLDER"
else
  echo "Error: Failed to copy $SOURCE_FILE to $DEST_FOLDER"
  exit 1
fi

echo "Stock file is now accessible at /nse_stocks.txt in your web application"