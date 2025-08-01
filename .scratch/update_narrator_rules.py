#!/usr/bin/env python3
"""
Script to reorganize narrator-rules.json by moving MCP tools to mcpRules section
"""

import json
import re

# Read the current narrator-rules.json
with open('narrator/narrator-rules.json', 'r', encoding='utf-8') as f:
    data = json.load(f)

# Remove all mcp__serena__ and mcp__ide__ entries from rules
rules_to_remove = []
for key in data['rules'].keys():
    if key.startswith('mcp__serena__') or key.startswith('mcp__ide__'):
        rules_to_remove.append(key)

for key in rules_to_remove:
    del data['rules'][key]

# The mcpRules section is already added, so we just need to save
with open('narrator/narrator-rules.json', 'w', encoding='utf-8') as f:
    json.dump(data, f, ensure_ascii=False, indent=2)

print(f"Removed {len(rules_to_remove)} MCP tool entries from rules section")