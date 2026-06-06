# API Reference

This app exposes same-origin JSON endpoints from the same Go server that serves
the embedded frontend. API routes are root-level paths, not `/api/*`. This
reference is kept simple because the project is mainly for learning Go.

## Base URL

Local development defaults to:

```text
http://localhost:8080
```

When running through Docker Compose, the host port is controlled by
`HOST_PORT`, and the in-container port is controlled by `PORT`.

## Conventions

- Request bodies and responses use JSON.
- Successful JSON responses include `Content-Type: application/json`.
- Timestamps are encoded as RFC 3339 strings.
- IDs are UUID strings.
- `PATCH` endpoints accept partial JSON bodies.
- `DELETE` endpoints return `204 No Content` on success.
- Error responses usually use:

```json
{
  "error": "message"
}
```

The `/readyz` endpoint returns a plain HTTP error body when the database is not
reachable.

## Health

### `GET /healthz`

Checks that the HTTP process is alive. This endpoint does not depend on the
database.

Response:

```json
{
  "status": "ok"
}
```

### `GET /readyz`

Checks whether the app can reach the configured database.

Response:

```json
{
  "status": "ready"
}
```

Failure:

- `503 Service Unavailable` when the database ping fails.

## Projects

Projects group tasks. The database migration seeds a built-in Inbox project.

### Project Shape

```json
{
  "id": "00000000-0000-0000-0000-000000000001",
  "name": "Inbox",
  "color": "#6b7280",
  "icon": "inbox",
  "position": 0,
  "archived": false,
  "task_count": 0,
  "created_at": "2026-06-06T00:00:00Z",
  "updated_at": "2026-06-06T00:00:00Z"
}
```

### `GET /projects`

Lists all non-archived projects with active top-level task counts.

Response:

```json
[
  {
    "id": "00000000-0000-0000-0000-000000000001",
    "name": "Inbox",
    "color": "#6b7280",
    "icon": "inbox",
    "position": 0,
    "archived": false,
    "task_count": 0,
    "created_at": "2026-06-06T00:00:00Z",
    "updated_at": "2026-06-06T00:00:00Z"
  }
]
```

### `POST /projects`

Creates a project.

Request:

```json
{
  "name": "Work",
  "color": "#2563eb",
  "icon": "briefcase"
}
```

Validation:

- `name` is required and cannot be blank.

Response:

- `201 Created`
- Project object.

### `GET /projects/{id}`

Gets one project.

Response:

- `200 OK` with a project object.
- `404 Not Found` when the project does not exist.

### `PATCH /projects/{id}`

Partially updates a project.

Request:

```json
{
  "name": "Deep Work",
  "color": "#0f766e",
  "icon": "target",
  "archived": false
}
```

All fields are optional. Omitted fields are left unchanged.

Response:

- `200 OK` with the updated project object.
- `404 Not Found` when the project does not exist.

### `DELETE /projects/{id}`

Deletes a project.

Response:

- `204 No Content` on success.
- `403 Forbidden` when attempting to delete the built-in Inbox project.
- `404 Not Found` when the project does not exist.

## Labels

Labels are colored tags attached to tasks.

### Label Shape

```json
{
  "id": "4fbc1ec2-5bb1-48ff-b89d-38d01ff6708d",
  "name": "Important",
  "color": "#dc2626",
  "created_at": "2026-06-06T00:00:00Z"
}
```

### `GET /labels`

Lists labels.

Response:

```json
[
  {
    "id": "4fbc1ec2-5bb1-48ff-b89d-38d01ff6708d",
    "name": "Important",
    "color": "#dc2626",
    "created_at": "2026-06-06T00:00:00Z"
  }
]
```

### `POST /labels`

Creates a label.

Request:

```json
{
  "name": "Important",
  "color": "#dc2626"
}
```

Validation:

- `name` is required and cannot be blank.

Response:

- `201 Created`
- Label object.

### `PATCH /labels/{id}`

Partially updates a label.

Request:

```json
{
  "name": "Errands",
  "color": "#16a34a"
}
```

All fields are optional. Omitted fields are left unchanged.

Response:

- `200 OK` with the updated label object.
- `404 Not Found` when the label does not exist.

### `DELETE /labels/{id}`

Deletes a label.

Response:

- `204 No Content` on success.
- `404 Not Found` when the label does not exist.

## Tasks

Tasks can belong to projects, have labels, and contain one level of subtasks in
the detailed task response.

Priority values:

- `1`: urgent
- `2`: high
- `3`: medium
- `4`: none

### Task Shape

```json
{
  "id": "ad2741b7-3b35-4f2b-9a90-3f82d0e9b41e",
  "title": "Plan the week",
  "notes": "Review projects and pick next actions.",
  "project_id": "00000000-0000-0000-0000-000000000001",
  "parent_id": null,
  "priority": 2,
  "due_date": "2026-06-06",
  "completed": false,
  "completed_at": null,
  "position": 0,
  "labels": [],
  "subtasks": [],
  "created_at": "2026-06-06T00:00:00Z",
  "updated_at": "2026-06-06T00:00:00Z"
}
```

`subtasks` is omitted from some list responses when it is empty.

### `GET /tasks`

Lists tasks.

Query parameters:

| Name | Values | Description |
| --- | --- | --- |
| `project_id` | UUID | Restrict tasks to one project |
| `due` | `today`, `upcoming` | Filter by due date |
| `completed` | `true`, `false` | Filter by completion state |

Examples:

```text
GET /tasks
GET /tasks?project_id=00000000-0000-0000-0000-000000000001
GET /tasks?due=today
GET /tasks?completed=false
```

Response:

- `200 OK`
- Array of task objects.

### `GET /tasks/search?q={query}`

Searches task titles and notes.

Behavior:

- Blank or missing `q` returns an empty array.
- Non-blank `q` performs a case-insensitive database search.

Example:

```text
GET /tasks/search?q=week
```

Response:

- `200 OK`
- Array of matching task objects.

### `POST /tasks`

Creates a task.

Request:

```json
{
  "title": "Plan the week",
  "notes": "Review projects and pick next actions.",
  "project_id": "00000000-0000-0000-0000-000000000001",
  "parent_id": null,
  "priority": 2,
  "due_date": "2026-06-06",
  "label_ids": []
}
```

Validation:

- `title` is required and cannot be blank.
- `due_date`, when provided, should be formatted as `YYYY-MM-DD`.

Response:

- `201 Created`
- Task object.

### `GET /tasks/{id}`

Gets one task with labels and one level of subtasks.

Response:

- `200 OK` with a task object.
- `404 Not Found` when the task does not exist.

### `PATCH /tasks/{id}`

Partially updates a task.

Request:

```json
{
  "title": "Plan next week",
  "notes": "Add review notes.",
  "project_id": "00000000-0000-0000-0000-000000000001",
  "priority": 1,
  "due_date": "2026-06-07",
  "completed": true,
  "label_ids": ["4fbc1ec2-5bb1-48ff-b89d-38d01ff6708d"]
}
```

All fields are optional. Omitted fields are left unchanged.

Notes:

- `due_date` can be set to an empty string to clear the due date.
- `label_ids`, when present, replaces the task's existing labels.
- `completed` updates completion state and lets the store maintain
  `completed_at`.

Response:

- `200 OK` with the updated task object.
- `404 Not Found` when the task does not exist.

### `DELETE /tasks/{id}`

Deletes a task. Subtasks are deleted through database cascade behavior.

Response:

- `204 No Content` on success.
- `404 Not Found` when the task does not exist.

## CORS

CORS behavior is configured by `CORS_ALLOWED_ORIGINS`.

- Empty value: same-origin usage only; no `Access-Control-Allow-Origin` header.
- `*`: allow all origins.
- Comma-separated origins: allow only exact origin matches.

Example:

```text
CORS_ALLOWED_ORIGINS=http://localhost:5173,http://localhost:3000
```

## Curl Examples

Create a label:

```sh
curl -sS -X POST http://localhost:8080/labels \
  -H 'Content-Type: application/json' \
  -d '{"name":"Important","color":"#dc2626"}'
```

Create a task:

```sh
curl -sS -X POST http://localhost:8080/tasks \
  -H 'Content-Type: application/json' \
  -d '{
    "title": "Plan the week",
    "notes": "Review projects and pick next actions.",
    "project_id": "00000000-0000-0000-0000-000000000001",
    "priority": 2,
    "due_date": "2026-06-06",
    "label_ids": []
  }'
```

List active Inbox tasks:

```sh
curl -sS 'http://localhost:8080/tasks?project_id=00000000-0000-0000-0000-000000000001&completed=false'
```
