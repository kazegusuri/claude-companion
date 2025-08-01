#!/usr/bin/env python3
import json

# Read the current configuration
with open('narrator/narrator-rules.json', 'r', encoding='utf-8') as f:
    data = json.load(f)

# Update Read tool - add filename capture
data['rules']['Read']['captures'] = [
    {
        'inputKey': 'file_path',
        'parseFileType': True
    },
    {
        'inputKey': 'filename'  # This will be computed from file_path
    }
]

# Update Write tool - add filename capture
data['rules']['Write']['captures'] = [
    {
        'inputKey': 'file_path',
        'parseFileType': True
    },
    {
        'inputKey': 'filename'
    }
]

# Update Edit tool - add filename capture
data['rules']['Edit']['captures'] = [
    {
        'inputKey': 'file_path',
        'parseFileType': True
    },
    {
        'inputKey': 'filename'
    }
]

# Update NotebookRead tool - add filename capture
data['rules']['NotebookRead']['captures'] = [
    {
        'inputKey': 'notebook_path',
        'parseFileType': True
    },
    {
        'inputKey': 'filename'
    }
]

# Update NotebookEdit tool - add filename capture
data['rules']['NotebookEdit']['captures'] = [
    {
        'inputKey': 'notebook_path',
        'parseFileType': True
    },
    {
        'inputKey': 'filename'
    }
]

# Write the updated configuration
with open('narrator/narrator-rules.json', 'w', encoding='utf-8') as f:
    json.dump(data, f, ensure_ascii=False, indent=2)

print("Updated narrator-rules.json with filename captures")