# Remote Clients Milestone

Future milestone for remote client support. Not in v1 scope.

## Features Required

### Agent Interaction Data
- `interaction_id` - Unique ID for pending interaction
- `interaction_type` - question | plan | permission
- `interaction_data` - JSON blob with interaction details
- `last_message_preview` - Last agent message (for context)

### Real-time Streaming
- WebSocket endpoint for agent status updates
- Push notifications for status changes
- Live log streaming

### Remote Actions
- Answer agent questions remotely
- Approve/reject permissions remotely
- Approve/reject sessions remotely
- Provide plan feedback remotely

### Authentication
- API tokens for daemon access
- Per-project access control
- Session-scoped tokens for agents

### Notifications
- Webhook support for status changes
- Slack/Discord integration
- Email notifications for review requests

### Multi-machine
- Machine identity tracking
- Session handoff between machines
- Conflict resolution for concurrent edits

## Data Model Additions

```json
"session": {
  "current": {
    "status": "waiting_permission",
    "interaction": {
      "id": "int-uuid",
      "type": "permission",
      "data": {
        "tool": "Bash",
        "command": "rm -rf node_modules",
        "reason": "Clean install"
      },
      "asked_at": "2026-01-18T11:30:00Z"
    },
    "last_message": "I need to clean node_modules before reinstalling...",
    "updated_at": "2026-01-18T11:30:00Z"
  }
}
```

## API Additions

| Method | Endpoint | Description |
|--------|----------|-------------|
| WS | `/ws/sessions/{id}` | Real-time status stream |
| POST | `/sessions/{id}/respond` | Answer interaction |
| POST | `/webhooks` | Register webhook |
| GET | `/auth/token` | Generate API token |
