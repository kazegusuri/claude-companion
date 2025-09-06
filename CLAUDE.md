# Instructions for Claude

## Project Overview
This is a Go daemon that parses Claude's JSONL log files in real-time, with a web frontend for visualization and API management using TypeSpec.

## Important Guidelines

### API Development with TypeSpec

#### API Definition
When modifying API endpoints or data structures:
1. Edit the API definition in `api/main.tsp`
2. Generate TypeScript types and OpenAPI spec:
   ```bash
   cd web && bun run tsp:generate
   ```
3. Update the backend Go code to match the API specification
4. Update the frontend to use the generated types from `web/src/types/api.ts`

#### API Workflow
- Always define APIs in TypeSpec first before implementing
- Generated types are the source of truth - do not manually edit `web/src/types/api.ts`
- Keep the API definition synchronized between frontend and backend

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

#### Code Formatting
After editing any TypeScript, TSX, JavaScript, or CSS files, you MUST run:
```bash
cd web
bun run format
```

This ensures all web code follows Biome formatting conventions.

#### Linting and Type Checking
Before committing any changes to web files, you MUST run:
```bash
cd web
bun run typecheck  # Run TypeScript type checking
bun run check      # Run Biome linting
bun run format     # Apply Biome formatting

# Or run all checks at once:
bun run check:all  # Run both TypeScript and Biome checks
```

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
2. If API changes are made:
   - Update `api/main.tsp`
   - Run `cd web && bun run tsp:generate` to generate types
3. Run code formatting:
   - For Go files: `make fmt`
   - For Web files: `cd web && bun run format`
   - For API files: `cd api && npm run format`
4. Run checks before committing:
   - For Go files: `make test`
   - For Web files: `cd web && bun run check:all && bun run format`
5. Ensure all checks pass
6. Commit with meaningful commit messages

## Project Structure
```
.
├── api/                    # API specification (TypeSpec)
│   ├── main.tsp           # API definition
│   ├── tspconfig.yaml     # TypeSpec configuration
│   └── tsp-output/        # Generated OpenAPI/JSON Schema
├── web/                   # Frontend application
│   ├── src/
│   │   ├── types/api.ts   # Generated TypeScript types
│   │   └── ...
│   └── package.json
├── internal/              # Go backend code
│   ├── server/
│   │   ├── api/          # HTTP API handlers
│   │   ├── websocket/    # WebSocket server
│   │   └── db/           # Database layer
│   └── ...
├── main.go               # Go application entry point
├── Makefile              # Go build automation
└── package.json          # Root workspace configuration
```