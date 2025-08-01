#!/bin/bash

# Script to analyze gh commands from JSONL files

LOGS_DIR="/home/kazegusuri/.claude/projects/-home-kazegusuri-go-src-github-com-newmohq-newmo-app"

echo "Analyzing GitHub CLI (gh) commands from JSONL logs..."
echo "=================================================="
echo

# Extract all gh commands from JSON logs
echo "Extracting gh commands..."
grep -h '"command":"gh\s' "$LOGS_DIR"/*.jsonl 2>/dev/null | \
    grep -oE '"command":"gh [^"]+' | \
    sed 's/"command":"//g' | \
    sed 's/"$//' > /tmp/gh_commands.txt

# Count total commands
total_commands=$(wc -l < /tmp/gh_commands.txt)
echo "Total gh commands found: $total_commands"
echo

# Extract and categorize by subcommand
echo "Commands by subcommand:"
echo "-----------------------"
cat /tmp/gh_commands.txt | \
    awk '{print $2}' | \
    sort | \
    uniq -c | \
    sort -nr

echo
echo "Detailed command patterns:"
echo "------------------------"

# Group by subcommand and show variations
for subcmd in $(cat /tmp/gh_commands.txt | awk '{print $2}' | sort | uniq); do
    echo
    echo "=== gh $subcmd ==="
    grep "^gh $subcmd" /tmp/gh_commands.txt | \
        sed "s/^gh $subcmd //" | \
        sort | \
        uniq -c | \
        sort -nr | \
        head -20
done

# Clean up
rm -f /tmp/gh_commands.txt