# REST API Specification

Base URL: same origin. The Go server serves both the frontend and API, and all responses are JSON except `204 No Content`.

## Common patterns

- **List endpoints** return `[]` (empty array, never `null`)
- **Get endpoints** return the object or `404`
- **Create endpoints** return the created object with `201`
- **Update endpoints** return the updated object with `200`; send only the fields you want to change
- **Delete endpoints** return `204` with no body, or `404`

## Error format

```json
{ "error": "human-readable message" }
```

## Projects

### `GET /projects`
Response: `[{ id, name, color, icon, position, archived, task_count, created_at, updated_at }]`

### `POST /projects`
Body: `{ "name": "Work", "color": "#3b82f6", "icon": "briefcase" }`
Response: 201 + created project

### `GET /projects/{id}`
Response: project object or 404

### `PATCH /projects/{id}`
Body: any subset of `{ name, color, icon, archived }`
Response: updated project or 404

### `DELETE /projects/{id}`
Response: 204 or 403 (Inbox) or 404

## Labels

### `GET /labels`
Response: `[{ id, name, color, created_at }]`

### `POST /labels`
Body: `{ "name": "urgent", "color": "#ef4444" }`
Response: 201 + created label

### `PATCH /labels/{id}`
Body: any subset of `{ name, color }`
Response: updated label or 404

### `DELETE /labels/{id}`
Response: 204 or 404

## Tasks

### `GET /tasks`
Query params:
- `project_id` — filter to a single project
- `due=today` — due today
- `due=upcoming` — due in next 7 days
- `completed=true|false` — filter by completion status

Response: `[{ id, title, notes, project_id, parent_id, priority, due_date, completed, completed_at, position, labels: [...], created_at, updated_at }]`

### `GET /tasks/search?q=query`
Response: same as `GET /tasks` but filtered by ILIKE on title and notes

### `POST /tasks`
Body:
```json
{
  "title": "Buy milk",
  "notes": "2% organic",
  "project_id": "...",
  "parent_id": "...",
  "priority": 2,
  "due_date": "2026-06-15",
  "label_ids": ["...", "..."]
}
```
Response: 201 + created task (with labels populated)

### `GET /tasks/{id}`
Response: task with `labels` and `subtasks` arrays populated, or 404

### `PATCH /tasks/{id}`
Body: any subset of `{ title, notes, project_id, priority, due_date, completed, label_ids }`
- `due_date: ""` clears the due date
- `project_id: ""` moves to Inbox
- `label_ids` replaces all labels when present

Response: updated task or 404

### `DELETE /tasks/{id}`
Response: 204 or 404

## Task JSON shape

```json
{
  "id": "uuid",
  "title": "string",
  "notes": "string",
  "project_id": "uuid|null",
  "parent_id": "uuid|null",
  "priority": 1,
  "due_date": "2026-06-15|null",
  "completed": false,
  "completed_at": "2026-06-01T12:00:00Z|null",
  "position": 1,
  "labels": [
    { "id": "uuid", "name": "urgent", "color": "#ef4444", "created_at": "..." }
  ],
  "subtasks": [],
  "created_at": "2026-06-01T10:00:00Z",
  "updated_at": "2026-06-01T10:00:00Z"
}
```
