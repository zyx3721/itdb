# ITDB Refactor Guide (PHP -> Go + Vue3)

## Target Architecture
- Frontend: `frontend/` (Vue3, Router, Pinia)
- Backend: `backend/` (Go, Chi, SQLite)
- Database: reuse `backend/data/itdb.db`

## Compatibility Principles
- Keep original SQLite data model and business relationships.
- Preserve original write-path behavior (association updates, orphan file cleanup, user/rack/location delete rules, history logging).
- Use Go implementation only; no PHP runtime dependency for new stack.

## Start Backend
```bash
cd backend
go run main.go
```

## Start Frontend
```bash
cd frontend
npm install
npm run dev
```

Default frontend API target:
- `/api` (dev server proxies to `http://127.0.0.1:8080`)

## Data Backup Policy
When changing schema in future:
1. Backup first.
2. Apply migration.
3. Validate row counts and key relations.

Backup script:
```powershell
cd backend\scripts
.\backup_db.ps1
```

## Current Scope
- Core modules migrated to API routes:
  - auth
  - items / software / invoices / contracts / files
  - agents / users / locations / racks
  - dictionaries / settings / contract events / location areas
  - tags management
  - item actions (maintenance log)
  - reports
  - csv import (preview + execute)
- Frontend provides a modernized admin workspace with resource pages and settings/dictionary views.
