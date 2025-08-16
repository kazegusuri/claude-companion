# Instructions for Claude

## Project Overview
This is a Go daemon that parses Claude's JSONL log files in real-time.

## Important Guidelines

### Go Development

#### Code Formatting
After editing any Go files, you MUST run:
```bash
make fmt
```

This ensures all Go code follows standard formatting conventions.

#### Building
Use the Makefile for building:
```bash
make build
```

#### Testing
Before committing any changes, run:
```bash
make test
```

#### Running the Application
```bash
make run PROJECT=project_name SESSION=session_name
```

#### Code Style
- Follow standard Go conventions
- Use meaningful variable names
- Keep functions focused and small
- Add appropriate error handling

### Web Development with Bun

Default to using Bun instead of Node.js.

- Use `bun <file>` instead of `node <file>` or `ts-node <file>`
- Use `bun test` instead of `jest` or `vitest`
- Use `bun build <file.html|file.ts|file.css>` instead of `webpack` or `esbuild`
- Use `bun install` instead of `npm install` or `yarn install` or `pnpm install`
- Use `bun run <script>` instead of `npm run <script>` or `yarn run <script>` or `pnpm run <script>`
- Bun automatically loads .env, so don't use dotenv.

#### Testing
Use `bun test` to run tests.

```ts#index.test.ts
import { test, expect } from "bun:test";

test("hello world", () => {
  expect(1).toBe(1);
});
```

## Git Workflow
1. Make changes
2. Run `make fmt` to format Go code (if applicable)
3. Run `make test` or `bun test` to ensure tests pass
4. Commit with meaningful commit messages