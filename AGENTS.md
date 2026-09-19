# \aires-magi

This file provides context about the project for AI assistants.

## Project Overview

- **Ecosystem**: Typescript

## Tech Stack

- **Runtime**: bun
- **Package Manager**: bun

### Frontend

- Framework: next
- CSS: tailwind
- UI Library: shadcn-ui
- State: zustand

### Database

- Database: postgres

### Additional Features

- Testing: vitest

## Project Structure

```
aires-magi/
├── apps/
│   ├── web/         # Frontend application
├── packages/
│   └── db/          # Database schema
```

## Common Commands

- `bun install` - Install dependencies
- `bun dev` - Start development server
- `bun build` - Build for production
- `bun test` - Run tests
- `bun db:push` - Push database schema
- `bun db:studio` - Open database UI

## Better Fullstack project context

`bts.jsonc` is the authority for the current Stack Graph. Its `stackParts` array owns role selection and `ownerPartId` bindings. Top-level option fields are a compatibility projection and must not become a second mutation path.

### Stack Parts, ownership, and evidence

- `backend.api:go:oapi-codegen`. It belongs to `backend:go:gin`. Evidence is `listed` with `unverified` freshness. Verification maintainer: @Marve10s.
- `backend.caching:go:redis`. It belongs to `backend:go:gin`. Evidence is `listed` with `unverified` freshness. Verification maintainer: @Marve10s.
- `backend.codeQuality:go:golangci-lint`. It belongs to `backend:go:gin`. Evidence is `listed` with `unverified` freshness. Verification maintainer: @Marve10s.
- `backend.config:go:viper`. It belongs to `backend:go:gin`. Evidence is `listed` with `unverified` freshness. Verification maintainer: @Marve10s.
- `backend.jobQueue:go:kafka-go`. It belongs to `backend:go:gin`. Evidence is `listed` with `unverified` freshness. Verification maintainer: @Marve10s.
- `backend.logging:go:zerolog`. It belongs to `backend:go:gin`. Evidence is `listed` with `unverified` freshness. Verification maintainer: @Marve10s.
- `backend.observability:go:opentelemetry`. It belongs to `backend:go:gin`. Evidence is `listed` with `unverified` freshness. Verification maintainer: @Marve10s.
- `backend.orm:go:gorm`. It belongs to `backend:go:gin`. Evidence is `listed` with `unverified` freshness. Verification maintainer: @Marve10s.
- `backend.testing:go:testify`. It belongs to `backend:go:gin`. Evidence is `listed` with `unverified` freshness. Verification maintainer: @Marve10s.
- `backend.validation:go:validator`. It belongs to `backend:go:gin`. Evidence is `listed` with `unverified` freshness. Verification maintainer: @Marve10s.
- `backend:go:gin`. Its generated target is `apps/server`. Evidence is `listed` with `unverified` freshness. Verification maintainer: @Marve10s.
- `database:universal:postgres`. Its generated target is `packages/db`. Evidence is `listed` with `unverified` freshness. Verification maintainer: @Marve10s.
- `frontend.css:typescript:tailwind`. It belongs to `frontend:typescript:next`. Evidence is `listed` with `unverified` freshness. Verification maintainer: @Marve10s.
- `frontend.deploy:typescript:docker`. It belongs to `frontend:typescript:next`. Evidence is `listed` with `unverified` freshness. Verification maintainer: @Marve10s.
- `frontend.forms:typescript:react-hook-form`. It belongs to `frontend:typescript:next`. Evidence is `listed` with `unverified` freshness. Verification maintainer: @Marve10s.
- `frontend.stateManagement:typescript:zustand`. It belongs to `frontend:typescript:next`. Evidence is `listed` with `unverified` freshness. Verification maintainer: @Marve10s.
- `frontend.ui:typescript:shadcn-ui`. It belongs to `frontend:typescript:next`. Evidence is `listed` with `unverified` freshness. Verification maintainer: @Marve10s.
- `frontend:typescript:next`. Its generated target is `apps/web`. Evidence is `listed` with `unverified` freshness. Verification maintainer: @Marve10s.
- `workspaceRunner:universal:turborepo`. Evidence is `listed` with `unverified` freshness. Verification maintainer: @Marve10s.

### Installed-version authority

Use `bts.jsonc` for the generator and schema version. Use local package manifests and lockfiles for installed dependency versions. Do not assume that documentation for a newer Better Fullstack release matches this project.

### Compatibility and lifecycle safety

Run `create-better-fullstack context --json` for bounded roles, capabilities, evidence, compatibility issues, and safe next actions. Run `create-better-fullstack doctor --json` before repairing graph drift. Existing-project writes must start with a plan and use the exact review token. Use `create-better-fullstack recipes check --json` before editing recipe-owned paths or managed regions, and use recipe history plus project recovery commands to undo a reviewed operation.

User code outside an explicit Better Fullstack managed region is not generator-owned. Missing or changed managed-region hashes stop recipe planning for manual review.

<!-- <better-fullstack:recipes sha256=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855> -->

<!-- </better-fullstack:recipes> -->

## Maintenance

Keep AGENTS.md updated when:

- Adding/removing dependencies
- Changing project structure
- Adding new features or services
- Modifying build/dev workflows

AI assistants should suggest updates to this file when they notice relevant changes.
