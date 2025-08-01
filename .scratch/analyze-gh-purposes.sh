#!/bin/bash

# Script to analyze gh commands with their descriptions/purposes

LOGS_DIR="/home/kazegusuri/.claude/projects/-home-kazegusuri-go-src-github-com-newmohq-newmo-app"

echo "GitHub CLI Command Usage Analysis with Purposes"
echo "============================================="
echo

# Extract commands with descriptions
echo "Extracting commands with their descriptions..."
grep -h '"command":"gh\s' "$LOGS_DIR"/*.jsonl 2>/dev/null | \
    grep -E '"command":"gh [^"]+.*"description":"[^"]+"' | \
    sed 's/.*"command":"//g' | \
    sed 's/","description":"/\t|\t/g' | \
    sed 's/".*//g' > /tmp/gh_commands_with_desc.txt

# Show sample commands with descriptions for each subcommand
for subcmd in pr run api; do
    echo
    echo "=== gh $subcmd - Examples with Descriptions ==="
    echo "-----------------------------------------------"
    grep "^gh $subcmd" /tmp/gh_commands_with_desc.txt | \
        head -10 | \
        while IFS=$'\t|\t' read -r cmd desc; do
            echo "Command: $cmd"
            echo "Purpose: $desc"
            echo
        done
done

# Extract specific use cases
echo
echo "=== Common Use Cases ==="
echo "----------------------"
echo
echo "1. Pull Request Operations:"
grep "^gh pr" /tmp/gh_commands_with_desc.txt | \
    cut -d$'\t' -f2 | \
    sort | uniq -c | sort -nr | head -10

echo
echo "2. GitHub Actions/Workflow Operations:"
grep "^gh run" /tmp/gh_commands_with_desc.txt | \
    cut -d$'\t' -f2 | \
    sort | uniq -c | sort -nr | head -10

echo
echo "3. API Direct Access:"
grep "^gh api" /tmp/gh_commands_with_desc.txt | \
    cut -d$'\t' -f2 | \
    sort | uniq -c | sort -nr | head -10

# Clean up
rm -f /tmp/gh_commands_with_desc.txt