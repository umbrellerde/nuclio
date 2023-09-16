#!/bin/bash

mkdir -p "usage"
# Output CSV file name
time=$(date +%s%3N)
output_file="usage/cpu_ram_utilization_$time.csv"

# Check if the output file already exists; if not, create it with headers
if [ ! -e "$output_file" ]; then
    echo "unix_time,cpu_percent,ram_percent" > "$output_file"
fi

# Main loop to measure CPU and RAM utilization
while true; do
    # Get current Unix timestamp in milliseconds
    unix_timestamp_ms=$(date +%s%3N)

    # Measure CPU utilization using mpstat
    cpu_utilization=$(mpstat 2 1 | awk 'END{print 100-$NF""}')

    # Measure RAM utilization using free
    ram_utilization=$(free | awk 'NR==2{printf "%.2f", $3*100/$2}')

    # Append data to the CSV file
    echo "$unix_timestamp_ms,$cpu_utilization,$ram_utilization" >> "$output_file"
done