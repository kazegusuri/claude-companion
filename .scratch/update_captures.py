#!/usr/bin/env python3
import json

# Read the JSON file
with open('narrator/narrator-rules.json', 'r', encoding='utf-8') as f:
    data = json.load(f)

# Update all captures to remove placeholder field
for tool_name, rules in data['rules'].items():
    if 'captures' in rules:
        new_captures = []
        for capture in rules['captures']:
            # Only keep inputKey field
            new_captures.append({
                'inputKey': capture['inputKey']
            })
        rules['captures'] = new_captures

# Write the updated JSON back
with open('narrator/narrator-rules.json', 'w', encoding='utf-8') as f:
    json.dump(data, f, ensure_ascii=False, indent=2)

print("Updated narrator-rules.json to remove placeholder fields from captures")