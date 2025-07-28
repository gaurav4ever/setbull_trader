#!/bin/bash

# Stock Group Management Script Runner
# This script helps you run the stock processing tools

set -e  # Exit on any error

echo "=========================================="
echo "Stock Group Management Script Runner"
echo "=========================================="

# Check if Python is available
if ! command -v python3 &> /dev/null; then
    echo "Error: Python 3 is not installed or not in PATH"
    exit 1
fi

# Check if we're in the right directory
if [ ! -f "add_stocks_to_group.py" ]; then
    echo "Error: Please run this script from the add_stock_in_stock_group directory"
    exit 1
fi

# Function to install dependencies
install_dependencies() {
    echo "Installing Python dependencies..."
    pip3 install -r requirements.txt
    echo "Dependencies installed successfully!"
}

# Function to run setup test
run_setup_test() {
    echo "Running setup validation test..."
    python3 test_setup.py
}

# Function to run main script
run_main_script() {
    echo "Running main stock processing script..."
    python3 add_stocks_to_group.py
}

# Main menu
while true; do
    echo ""
    echo "Choose an option:"
    echo "1) Install dependencies"
    echo "2) Run setup test"
    echo "3) Run main script"
    echo "4) Run setup test + main script"
    echo "5) Exit"
    echo ""
    read -p "Enter your choice (1-5): " choice

    case $choice in
        1)
            install_dependencies
            ;;
        2)
            run_setup_test
            ;;
        3)
            run_main_script
            ;;
        4)
            echo "Running complete workflow..."
            run_setup_test
            if [ $? -eq 0 ]; then
                echo ""
                read -p "Setup test passed! Press Enter to continue with main script..."
                run_main_script
            else
                echo "Setup test failed! Please fix the issues before running the main script."
            fi
            ;;
        5)
            echo "Goodbye!"
            exit 0
            ;;
        *)
            echo "Invalid choice. Please enter a number between 1 and 5."
            ;;
    esac
done 