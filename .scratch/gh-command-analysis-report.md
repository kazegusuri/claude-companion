# GitHub CLI (gh) Command Analysis Report

## Overview
Analysis of GitHub CLI (gh) commands found in JSONL logs from the directory:
`~/.claude/projects/-home-kazegusuri-go-src-github-com-newmohq-newmo-app/`

## Summary Statistics
- **Total gh commands found**: 194
- **Sessions analyzed**: 64 JSONL files

## Command Distribution by Subcommand

| Subcommand | Count | Percentage |
|------------|-------|------------|
| gh pr      | 138   | 71.1%      |
| gh run     | 51    | 26.3%      |
| gh api     | 3     | 1.5%       |
| gh --version | 2   | 1.0%       |

## Detailed Command Categories

### 1. Pull Request Operations (gh pr)

#### Most Common PR Commands:
- **gh pr create --draft --title**: 46 occurrences
  - Purpose: Creating draft pull requests with specified titles
  - Often includes complex body content using heredocs
  
- **gh pr view --web**: 44 occurrences
  - Purpose: Opening pull requests in the web browser
  
- **gh pr edit --body**: 7 occurrences
  - Purpose: Updating PR descriptions/body content
  
- **gh pr checks --watch**: 6 occurrences
  - Purpose: Monitoring CI/CD check status in real-time

#### Other PR Operations:
- **gh pr status**: Checking overall PR status
- **gh pr view [number]**: Viewing specific PR details
- **gh pr diff**: Examining PR changes
- **gh pr review --approve**: Approving PRs programmatically
- **gh pr list**: Listing PRs with filtering capabilities

### 2. GitHub Actions/Workflow Operations (gh run)

#### Common Patterns:
- **gh run view [run-id]**: Basic run information
- **gh run view [run-id] --log**: Full execution logs
- **gh run view [run-id] --job [job-id]**: Specific job details
- **gh run view [run-id] --log-failed**: Only failed step logs
- **gh run view [run-id] --web**: Open in browser
- **gh run list --branch [branch-name]**: List runs for specific branch

#### Specific Use Cases:
- Debugging CI failures (using --log-failed)
- Monitoring job progress during execution
- Analyzing test failures and build errors
- Checking workflow status for specific branches

### 3. GitHub API Direct Access (gh api)

#### API Endpoints Used:
- **repos/newmohq/newmo-app/branches**: Listing all branches
- **repos/newmohq/newmo-app/commits/[branch]**: Getting commit information
- **repos/newmohq/newmo-app/actions/runs/[id]/jobs**: Getting job details
- **DELETE repos/newmohq/newmo-app/git/refs/heads/[branch]**: Deleting branches

#### Notable Script Usage:
Found in `delete-stale-branches.sh`:
```bash
# List all branches
gh api repos/newmohq/newmo-app/branches --paginate -q '.[].name'

# Get commit info for a branch
gh api repos/newmohq/newmo-app/commits/$branch

# Delete a branch
gh api -X DELETE repos/newmohq/newmo-app/git/refs/heads/$branch
```

## Common Usage Patterns

### 1. Pull Request Workflow
1. Create draft PR with detailed description
2. Monitor PR checks
3. Update PR body/title as needed
4. View PR in browser for review
5. Approve PR when ready

### 2. CI/CD Debugging Workflow
1. Check run status for a branch
2. View specific job details
3. Examine failed logs
4. Analyze test failures with grep patterns

### 3. Branch Management
- Automated branch cleanup using gh api
- Checking branch age via commit timestamps
- Conditional deletion based on branch patterns

## Language Usage
Most commands include Japanese descriptions, indicating usage by Japanese-speaking developers:
- "GitHub Action実行のログを確認"
- "test jobの詳細を確認"
- "ブラウザでAction実行ページを開く"

## Integration Patterns
- Heavy use of JSON output with jq for processing
- Integration with shell scripts for automation
- Combination with grep for log analysis
- Use of --watch for real-time monitoring

## Recommendations
1. The high usage of draft PRs suggests a review-heavy workflow
2. Frequent use of --web flag indicates preference for browser-based review
3. API usage for branch management shows advanced automation practices
4. Log analysis patterns suggest active debugging and monitoring culture