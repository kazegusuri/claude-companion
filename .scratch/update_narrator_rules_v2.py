#!/usr/bin/env python3
import json

# Read the current configuration
with open('narrator/narrator-rules.json', 'r', encoding='utf-8') as f:
    data = json.load(f)

# Remove extensionTemplates
if 'extensionTemplates' in data:
    del data['extensionTemplates']

# Update Read tool
data['rules']['Read']['default'] = '{filetype}「{filename}」を読み込みます'
data['rules']['Read']['captures'] = [
    {
        'inputKey': 'file_path',
        'parseFileType': True
    }
]

# Update Write tool
data['rules']['Write']['default'] = '{filetype}「{filename}」を作成します'
if 'captures' not in data['rules']['Write']:
    data['rules']['Write']['captures'] = []
data['rules']['Write']['captures'].append({
    'inputKey': 'file_path',
    'parseFileType': True
})

# Update Edit tool
data['rules']['Edit']['default'] = '{filetype}「{filename}」を編集します'
if 'captures' not in data['rules']['Edit']:
    data['rules']['Edit']['captures'] = []
data['rules']['Edit']['captures'].append({
    'inputKey': 'file_path',
    'parseFileType': True
})

# Update NotebookRead tool  
data['rules']['NotebookRead']['default'] = '{filetype}「{filename}」を読み込みます'
if 'captures' not in data['rules']['NotebookRead']:
    data['rules']['NotebookRead']['captures'] = []
data['rules']['NotebookRead']['captures'].append({
    'inputKey': 'notebook_path',
    'parseFileType': True
})

# Update NotebookEdit tool
data['rules']['NotebookEdit']['default'] = '{filetype}「{filename}」を編集します'
if 'captures' not in data['rules']['NotebookEdit']:
    data['rules']['NotebookEdit']['captures'] = []
data['rules']['NotebookEdit']['captures'].append({
    'inputKey': 'notebook_path', 
    'parseFileType': True
})

# Update mcp__serena__read_file tool
data['rules']['mcp__serena__read_file']['default'] = '{filetype}「{file_path}」を読み込みます'
if 'captures' not in data['rules']['mcp__serena__read_file']:
    data['rules']['mcp__serena__read_file']['captures'] = []
data['rules']['mcp__serena__read_file']['captures'].append({
    'inputKey': 'file_path',
    'parseFileType': True
})

# Write the updated configuration
with open('narrator/narrator-rules.json', 'w', encoding='utf-8') as f:
    json.dump(data, f, ensure_ascii=False, indent=2)

print("Updated narrator-rules.json with parseFileType captures")