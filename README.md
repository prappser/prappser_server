# Prappser Server

A Go-based server for the Prappser platform, providing event-driven application state synchronization across devices.

## Table of Contents
- [Authentication Flows](#authentication-flows)
- [Event Architecture](#event-architecture)
- [API Endpoints](#api-endpoints)

## Authentication Flows
![Authentication Flows](/docs/auth_flows.svg)

## Event Architecture

Prappser uses a **client-produced, server-validated** event architecture for application state synchronization.

### Architecture Overview

**Flow:**
```
Client → Produces Event → Server Validates & Orders → Server ACKs → Client Processes Locally → Other Clients Poll & Process
```

**Key Principles:**
- Clients produce all events (no server-side event generation)
- Server validates structure, data, and authorization before accepting
- Server assigns sequence numbers for strict total ordering
- Clients wait for server ACK before processing locally (no optimistic updates)
- All clients receive events via polling and process them identically

### Event Submission Flow

1. **Client Creates Event**: User performs an action (leave app, delete app, etc.)
2. **Client Submits**: `POST /events` with event JSON
3. **Server Validates**: Checks event structure and required fields
4. **Server Authorizes**: Verifies user has permission for this event type
5. **Server Orders**: Assigns sequence number and server timestamp
6. **Server Persists**: Saves event to database
7. **Server ACKs**: Returns accepted event with sequence number
8. **Client Processes**: Updates local database based on event
9. **Other Clients Poll**: `GET /events?since={lastEventId}` retrieves new events
10. **Other Clients Process**: Update their local databases identically

### Event Types

| Event Type | Description | Authorization |
|------------|-------------|---------------|
| `application_deleted` | Application removed | Owner only |
| `member_removed` | Member leaves or is removed | Self or owner |
| `member_added` | New member joins via invite | Any member (via invite system) |
| `member_role_changed` | Member role updated | Owner only |
| `application_data_changed` | Application data modified | Any member |
| `invite_revoked` | Invitation cancelled | Owner only |

### Authorization Rules

Event authorization is enforced in `internal/event/event_authorizer.go`:

- **application_deleted**: Only the application owner can delete the entire application
- **member_removed**: Members can remove themselves; owners can remove any member
- **member_added**: Any member can add others (via invitation system)
- **member_role_changed**: Only owners can change member roles
- **application_data_changed**: Any member can update application data
- **invite_revoked**: Only owners can revoke invitations

### Event Ordering

Events are ordered using two fields for guaranteed consistency:

1. **Primary**: `sequence_number` - Per-application counter (1, 2, 3, ...)
2. **Secondary**: `created_at` - Server-assigned Unix timestamp

Query ordering: `ORDER BY sequence_number ASC, created_at ASC`

This ensures:
- No timestamp ambiguity for sub-second events
- Client-independent ordering (no clock skew issues)
- Total order guarantee within each application
- Conflict detection capability

### Event Validation

Server validates events in `internal/event/event_validator.go`:

**Required Fields:**
- `id`: Unique event identifier (UUID)
- `type`: Event type (must be valid enum)
- `creatorPublicKey`: Public key of event creator
- `data`: Event-specific data object
- `version`: Event schema version

**Type-Specific Validation:**
Each event type has specific data requirements validated by the server before acceptance.

### Rejection Handling

If the server rejects an event:

**Response Format:**
```json
{
  "accepted": false,
  "error": "authorization failed: only owner can delete application",
  "reason": "unauthorized"
}
```

**HTTP Status Codes:**
- `200`: Event accepted
- `400`: Validation failed (invalid structure/data)
- `403`: Unauthorized (permission denied)
- `500`: Internal server error

**Client Behavior:**
The client automatically shows an error dialog via AsyncValue error handling when events are rejected.

## API Endpoints

### Event Endpoints

#### POST /events
Submit an event for validation and processing.

**Authentication**: Required (JWT token)

**Request Body:**
```json
{
  "event": {
    "id": "uuid-v7",
    "type": "member_removed",
    "creatorPublicKey": "user-public-key",
    "data": {
      "applicationId": "app-id",
      "memberPublicKey": "user-public-key",
      "reason": "left",
      "version": 1
    },
    "createdAt": 1234567890
  }
}
```

**Success Response (200):**
```json
{
  "accepted": true,
  "event": { /* event with server-assigned sequence and timestamp */ },
  "sequence": 42,
  "timestamp": 1234567890
}
```

**Rejection Response (400/403):**
```json
{
  "accepted": false,
  "error": "validation failed: applicationId is required",
  "reason": "validation_failed"
}
```

#### GET /events
Poll for new events since last received event.

**Authentication**: Required (JWT token)

**Query Parameters:**
- `since` (optional): Last event ID received by client
- `limit` (optional): Maximum events to return (default: 100, max: 500)

**Response:**
```json
{
  "events": [/* array of events */],
  "hasMore": true,
  "fullResyncRequired": false
}
```

### Legacy Endpoints

The following endpoints are kept for backward compatibility but no longer produce events:

- `DELETE /applications/{id}` - Delete application (operation only, client submits event)
- `DELETE /applications/{id}/members/me` - Leave application (operation only, client submits event)

