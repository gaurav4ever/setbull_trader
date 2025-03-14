#!/bin/bash

# Set the output directory
OUTPUT_DIR="consolidated_code"
mkdir -p "$OUTPUT_DIR"

# Function to add file content with header to a consolidated file
append_file() {
    local source_file=$1
    local dest_file=$2
    
    if [ -f "$source_file" ]; then
        echo -e "\n\n===== FILE: $source_file =====\n" >> "$dest_file"
        cat "$source_file" >> "$dest_file"
        echo "Added $source_file to $dest_file"
    fi
}

# Create the main consolidated files
touch "$OUTPUT_DIR/models.txt"
touch "$OUTPUT_DIR/services.txt"
touch "$OUTPUT_DIR/transport.txt"
touch "$OUTPUT_DIR/config.txt"
touch "$OUTPUT_DIR/utils.txt"
touch "$OUTPUT_DIR/main.txt"

echo "Created consolidated files in $OUTPUT_DIR"

# Main application files
append_file "main.go" "$OUTPUT_DIR/main.txt"

# Config files
append_file "internal/trading/config/config.go" "$OUTPUT_DIR/config.txt"

# Transport/HTTP handlers
append_file "cmd/trading/transport/http.go" "$OUTPUT_DIR/transport.txt"
append_file "cmd/trading/app/app.go" "$OUTPUT_DIR/transport.txt"

# Service files
append_file "internal/core/service/orders/service.go" "$OUTPUT_DIR/services.txt"

# Model files
append_file "internal/core/adapters/client/dhan/models.go" "$OUTPUT_DIR/models.txt"
append_file "internal/core/dto/request/orders.go" "$OUTPUT_DIR/models.txt"
append_file "internal/core/dto/response/orders.go" "$OUTPUT_DIR/models.txt"

# Client files
append_file "internal/core/adapters/client/dhan/dhan_client.go" "$OUTPUT_DIR/services.txt"
append_file "internal/core/adapters/client/dhan/dhant_client.go" "$OUTPUT_DIR/services.txt"

# Utility files
append_file "pkg/apperrors/errors.go" "$OUTPUT_DIR/utils.txt"
append_file "pkg/log/log.go" "$OUTPUT_DIR/utils.txt"

echo "Consolidation complete! Files are available in the $OUTPUT_DIR directory."

# List created files with their sizes
echo -e "\nConsolidated files:"
ls -lh "$OUTPUT_DIR"