# ACCWeb → SimSports Backend
# Complete Incident Pipeline Implementation Specification

## Document Purpose

This document is the detailed implementation specification for extending the existing SimSports ACCWeb telemetry collector and SimSports backend.

The goal is to establish a clean, scalable evidence pipeline in which:

- **ACCWeb performs real-time ACC data collection and preprocessing.**
- **ACCWeb detects and aggregates meaningful race incidents locally.**
- **The backend does NOT receive every raw/high-frequency telemetry sample.**
- **The backend receives structured race results and normalized incidents.**
- **MongoDB uses `incidents` as the primary incident collection, NOT `collisions`.**
- **Collision is only one incident type among many.**
- **Future SA, ELO, and automated stewarding systems can consume the stored evidence.**

This specification is based on the existing ACCWeb collector architecture and the existing SimSports backend implementation described in the project walkthrough.

---

# 1. Current System

The current platform already contains an important foundation.

## Current architecture

```text
┌─────────────────────────────────────────────┐
│           Assetto Corsa Competizione        │
│                accServer.exe                │
└──────────────────────┬──────────────────────┘
                       │
             ┌─────────┴─────────┐
             │                   │
        Live Logs          Session Results
             │                   │
             ▼                   ▼
┌────────────────────────────────────────────┐
│                  ACCWeb                    │
│                                            │
│  Collector                                 │
│  Result Watcher                            │
│  Collision/Cut Detector                    │
│  Qualifying Cache                          │
│  Race Result Processor                     │
└──────────────────────┬─────────────────────┘
                       │ HTTP
                       ▼
┌────────────────────────────────────────────┐
│             SimSports Backend              │
│                                            │
│  Events                                    │
│  Race Results                              │
│  Collisions / Incidents                    │
│  Users                                     │
└──────────────────────┬─────────────────────┘
                       │
                       ▼
┌────────────────────────────────────────────┐
│                 MongoDB                    │
└────────────────────────────────────────────┘
```

The existing implementation already handles:

- Event ID configuration
- Collector enable/disable
- Collector status
- Server ID synchronization
- ACC server Start/Stop
- UTF-16LE result decoding
- Qualifying start-position caching
- Race result extraction
- Finish position
- Best lap
- Total time
- Completed laps
- Driver status
- Track cuts
- Invalid laps
- In-game penalties
- Live collision detection
- Steam ID resolution
- Car-number resolution
- Rejoin idempotency

The objective is to extend this foundation rather than replace it.

---

# 2. Main Architectural Decision

## ACCWeb becomes the edge-processing layer

The system should be divided into two major responsibilities.

### ACCWeb

ACCWeb is responsible for:

- Reading ACC data
- Parsing ACC logs
- Watching ACC result files
- Resolving driver identities
- Detecting events
- Detecting incidents
- Aggregating statistics
- Creating normalized incident objects
- Queuing incidents
- Batching incidents
- Sending only meaningful information to the backend
- Keeping raw/high-frequency evidence locally when required

### Backend

The backend is responsible for:

- Receiving validated data
- Persisting incidents
- Persisting race results
- Maintaining event/session relationships
- Maintaining driver history
- SA calculations
- ELO calculations
- Future stewarding decisions
- Driver profiles
- Leaderboards
- APIs for the web GUI

---

# 3. Target Architecture

```text
                         ACC SERVER
                       accServer.exe
                            │
             ┌──────────────┴──────────────┐
             │                             │
        Live Logs                     Result Files
             │                             │
             └──────────────┬──────────────┘
                            ▼
                   ┌──────────────────┐
                   │      ACCWeb      │
                   │                  │
                   │ Collector        │
                   │                  │
                   │ Result Watcher   │
                   │ Log Watcher      │
                   │ Driver Tracker   │
                   │ Event Detector   │
                   │ Incident Engine  │
                   │ Aggregator       │
                   │ Batcher          │
                   └────────┬─────────┘
                            │
                 Processed / useful data
                            │
                            ▼
                   ┌──────────────────┐
                   │  SimSports API   │
                   │                  │
                   │ Race Results     │
                   │ Incidents        │
                   │ Events           │
                   │ Drivers          │
                   │ SA / ELO         │
                   └────────┬─────────┘
                            │
                            ▼
                         MongoDB
```

---

# 4. Why Raw Telemetry Must Not Be Sent Continuously

Do NOT implement this:

```text
ACC telemetry sample
       ↓
HTTP POST
       ↓
Backend
       ↓
MongoDB
```

for every telemetry sample.

If multiple cars are generating frequent telemetry samples, this creates:

- Excessive HTTP requests
- Backend CPU overhead
- MongoDB write pressure
- Large network traffic
- Large database growth
- Increased latency
- More failure/retry opportunities
- Difficult scaling when multiple ACC servers are active

The backend does not need every raw sample for ordinary race-result processing.

---

# 5. Correct Data Strategy

Use this model:

```text
ACC
 ↓
ACCWeb
 ↓
RAW DATA
 ↓
PROCESS
 ↓
DETECT
 ↓
AGGREGATE
 ↓
MEANINGFUL EVENTS
 ↓
BACKEND
```

ACCWeb should transform raw ACC information into useful domain information before sending it.

Example:

Instead of sending:

```text
Frame 1
Frame 2
Frame 3
Frame 4
Frame 5
...
Frame 10000
```

ACCWeb should derive:

```json
{
  "driverId": "steam_123",
  "cutsCount": 3,
  "invalidLapsCount": 2
}
```

and send the aggregate.

---

# 6. Data Categories

The system should distinguish between four levels of data.

## Level 1 — Raw telemetry

High-frequency information.

Examples:

- Speed
- Position
- Velocity
- Steering
- Brake
- Throttle
- RPM
- Gear
- G-force
- Heading
- Wheel slip
- Damage
- Suspension data

This data should NOT normally be sent continuously to the backend.

If retained, store it locally or in dedicated telemetry storage.

---

## Level 2 — Derived events

Events extracted from raw data.

Examples:

- Track limit transition
- Invalid lap
- Wheel spin
- Spin
- Loss of control
- Damage increase
- Collision candidate
- Rejoin
- Pit event

These are much smaller and useful to the backend.

---

## Level 3 — Incidents

An incident is a normalized event that has enough context to be useful for SA/stewarding.

Examples:

```text
COLLISION
WHEEL_SPIN
SPIN
LOSS_OF_CONTROL
TRACK_LIMIT
CUT
UNSAFE_REJOIN
DAMAGE
PENALTY
WRONG_WAY
```

---

## Level 4 — Ratings

The backend can eventually derive:

```text
SA
ELO
Licence
Stewarding outcome
Driver reputation
```

The ratings should be based on stored evidence rather than raw telemetry.

---

# 7. Generic Incident Model

The most important architectural change is:

## Do NOT use Collision as the primary data model.

The backend currently has a collision concept.

Replace/refactor it into:

```text
Incident
```

MongoDB collection:

```text
incidents
```

Not:

```text
collisions
```

The reason is that collisions are only one type of race incident.

The future system needs to represent:

```text
COLLISION
WHEEL_SPIN
SPIN
LOSS_OF_CONTROL
TRACK_LIMIT
CUT
UNSAFE_REJOIN
DAMAGE
PENALTY
WRONG_WAY
```

This creates one consistent evidence layer for SA and future stewarding.

---

# 8. Incident Lifecycle

Every incident should follow:

```text
ACC data
    ↓
Detection
    ↓
Incident object created
    ↓
Incident queued
    ↓
Batcher
    ↓
Backend API
    ↓
Validation
    ↓
MongoDB
    ↓
SA / Stewarding processing
```

---

# 9. Incident Identity

Every incident must have a unique identifier.

Example:

```text
inc_01JXXXXXXXX
```

The identifier must remain stable if the collector retries the request.

This is required for idempotency.

If ACCWeb sends the same incident twice:

```text
inc_001
inc_001
```

the backend must store only one record.

---

# 10. Required Incident Fields

Every incident should contain, at minimum:

```text
incidentId
eventId
serverId
sessionId
timestamp
type
participants
```

Participants should contain:

```text
steamId
carNumber
```

Optional contextual fields:

```text
lap
sector
location
severity
confidence
classification
evidence
telemetryWindow
processed
createdAt
```

---

# 11. Recommended Incident Schema

```json
{
  "incidentId": "inc_01JXXXXXXXX",

  "eventId": "event_123",
  "serverId": "server_01",
  "sessionId": "session_001",

  "timestamp": 1842.71,
  "timestampUtc": "2026-08-17T18:31:22.710Z",

  "lap": 17,
  "sector": 2,

  "type": "COLLISION",

  "participants": [
    {
      "steamId": "76561198789435655",
      "carNumber": 333
    },
    {
      "steamId": "76561198789435656",
      "carNumber": 21
    }
  ],

  "location": {
    "x": 123.42,
    "y": -2.18,
    "z": 482.31
  },

  "severity": 0.72,

  "confidence": 0.93,

  "classification": {
    "type": "REAR_END",

    "probabilities": {
      "driverA": 0.93,
      "driverB": 0.02,
      "racingIncident": 0.05
    }
  },

  "evidence": {
    "relativeSpeed": 31.7,
    "closingSpeed": 31.7,
    "distanceBeforeImpact": 4.2,

    "driverA": {
      "speedBefore": 218.4,
      "speedAtIncident": 176.2,
      "brake": 0.12,
      "throttle": 0.84,
      "steering": -0.03
    },

    "driverB": {
      "speedBefore": 172.1,
      "speedAtIncident": 191.4,
      "brake": 0.00,
      "throttle": 0.71,
      "steering": 0.01
    }
  },

  "telemetryWindow": {
    "available": true,
    "start": 1839.71,
    "impact": 1842.71,
    "end": 1844.71
  },

  "processed": false,

  "createdAt": "2026-08-17T18:31:23.000Z"
}
```

---

# 12. Probability Design

Do NOT use only:

```json
{
  "probability": 0.93
}
```

This is ambiguous.

A probability must have a defined meaning.

For a two-driver incident:

```json
{
  "classification": {
    "type": "REAR_END",

    "probabilities": {
      "driverA": 0.93,
      "driverB": 0.02,
      "racingIncident": 0.05
    }
  }
}
```

The probabilities should normally sum to approximately:

```text
1.0
```

Example:

```text
0.93 + 0.02 + 0.05 = 1.00
```

This allows the future stewarding system to understand:

- Who is likely responsible
- Whether the other driver is likely responsible
- Whether it is likely a racing incident

---

# 13. Single-Driver Incidents

Not every incident involves multiple drivers.

For:

```text
WHEEL_SPIN
SPIN
LOSS_OF_CONTROL
TRACK_LIMIT
CUT
```

use one participant.

Example:

```json
{
  "type": "WHEEL_SPIN",

  "participants": [
    {
      "steamId": "76561198789435655",
      "carNumber": 333
    }
  ],

  "classification": {
    "type": "WHEEL_SPIN",
    "probabilities": {
      "driver": 1.0
    }
  }
}
```

If no classification is available yet, do not fabricate a probability.

Use:

```json
{
  "classification": null
}
```

or an explicit processing state.

---

# 14. Collision Incident

Collision is one type of Incident.

When ACCWeb detects a collision:

1. Identify the involved cars.
2. Resolve ACC car IDs.
3. Resolve Steam IDs.
4. Resolve car numbers.
5. Capture timestamp.
6. Capture lap.
7. Capture sector if available.
8. Capture track position if available.
9. Capture available evidence.
10. Create an Incident.
11. Add it to the local queue.
12. Send asynchronously.

Example:

```json
{
  "incidentId": "inc_001",
  "eventId": "event_123",
  "sessionId": "race_001",

  "timestamp": 1842.71,

  "lap": 17,

  "type": "COLLISION",

  "participants": [
    {
      "steamId": "76561198789435655",
      "carNumber": 333
    },
    {
      "steamId": "76561198789435656",
      "carNumber": 21
    }
  ]
}
```

---

# 15. Collision Evidence

The collision detector should preserve the evidence that caused the collision to be detected.

Useful evidence includes:

```text
relative speed
closing speed
distance before impact
speed before impact
speed at impact
brake
throttle
steering
heading
g-force
damage delta
track position
car orientation
```

For each driver where available.

Example:

```json
{
  "evidence": {
    "relativeSpeed": 31.7,
    "closingSpeed": 31.7,
    "distanceBeforeImpact": 4.2,

    "driverA": {
      "speedBefore": 218.4,
      "speedAtIncident": 176.2,
      "brake": 0.12,
      "throttle": 0.84,
      "steering": -0.03
    },

    "driverB": {
      "speedBefore": 172.1,
      "speedAtIncident": 191.4,
      "brake": 0.00,
      "throttle": 0.71,
      "steering": 0.01
    }
  }
}
```

Do not fabricate fields that the current ACCWeb collector cannot actually obtain.

If a value is unavailable:

```json
null
```

or omit the optional field.

---

# 16. Collision Telemetry Window

A collision cannot always be understood from the exact impact frame.

The system should support an evidence window:

```text
T - 3.0 seconds
T - 2.0 seconds
T - 1.0 second
T - 0.5 seconds
T = impact
T + 0.5 seconds
T + 1.0 second
T + 2.0 seconds
```

Example:

```json
{
  "telemetryWindow": {
    "available": true,
    "start": 1839.71,
    "impact": 1842.71,
    "end": 1844.71
  }
}
```

Do not place thousands of telemetry samples inside the MongoDB Incident document.

The Incident should reference the evidence window.

---

# 17. Wheel Spin Incident

The system should support:

```text
WHEEL_SPIN
```

when reliable telemetry is available.

Possible evidence:

```text
speed
throttle
wheel speed
wheel slip
gear
RPM
TC state
```

Example:

```json
{
  "type": "WHEEL_SPIN",

  "participants": [
    {
      "steamId": "76561198789435655",
      "carNumber": 333
    }
  ],

  "timestamp": 1842.71,

  "lap": 17,

  "evidence": {
    "speed": 84.2,
    "throttle": 0.98,
    "wheelSlip": 0.61
  },

  "severity": 0.55
}
```

If the current ACCWeb implementation does not have enough telemetry to reliably detect wheel spin, implement the generic Incident infrastructure but do not fake detection.

---

# 18. Spin / Loss of Control

Support:

```text
SPIN
LOSS_OF_CONTROL
```

Potential evidence:

```text
speed
heading
yaw rate
steering
throttle
brake
lateral acceleration
longitudinal acceleration
track position
```

Example:

```json
{
  "type": "SPIN",

  "participants": [
    {
      "steamId": "76561198789435655",
      "carNumber": 333
    }
  ],

  "timestamp": 2201.42,

  "lap": 18,

  "severity": 0.68,

  "evidence": {
    "speed": 92.1,
    "headingChange": 142.4,
    "yawRate": 3.8
  }
}
```

---

# 19. Track Limit Incident

Do not create one incident for every invalid telemetry frame.

Detect the transition:

```text
VALID
   ↓
INVALID
```

Create one meaningful event.

Example:

```json
{
  "type": "TRACK_LIMIT",

  "participants": [
    {
      "steamId": "76561198789435655",
      "carNumber": 333
    }
  ],

  "timestamp": 1842.71,

  "lap": 17,

  "severity": 0.25
}
```

Repeated invalid frames must be deduplicated.

---

# 20. Cuts

ACCWeb already tracks `cutsCount`.

Continue maintaining the aggregate.

The backend race result can contain:

```json
{
  "cutsCount": 1,
  "invalidLapsCount": 1
}
```

If individual track-limit incidents are also enabled, those should go into the generic Incident system.

This means the system can maintain both:

```text
Race aggregate
    ↓
cutsCount = 3
```

and:

```text
Incident history
    ↓
TRACK_LIMIT at lap 5
TRACK_LIMIT at lap 11
TRACK_LIMIT at lap 18
```

---

# 21. Penalty Incident

Official ACC penalties should be represented as an Incident where appropriate.

Example:

```json
{
  "type": "PENALTY",

  "participants": [
    {
      "steamId": "76561198789435655",
      "carNumber": 333
    }
  ],

  "timestamp": 3012.2,

  "evidence": {
    "reason": "Cutting",
    "penalty": "DriveThrough",
    "penaltyValue": null,
    "violationInLap": 23,
    "clearedInLap": 24
  }
}
```

The existing race result should also preserve the official penalty information.

---

# 22. Unsafe Rejoin

Support:

```text
UNSAFE_REJOIN
```

This is important for future SA/stewarding.

Potential evidence:

```text
driver off track
driver returns to track
track position
other cars nearby
relative speed
distance
direction
time to rejoin
contact after rejoin
```

The initial implementation can create the data model and detection interface without claiming perfect classification.

---

# 23. Driver Identity

Use Steam ID as the persistent driver identity.

Do not use:

```text
displayName
```

as the primary identity.

The participant structure should be:

```json
{
  "steamId": "76561198789435655",
  "carNumber": 333
}
```

The display name can be stored separately if useful, but Steam ID should remain the stable identifier.

---

# 24. Rejoin Handling

Existing rejoin behavior must remain intact.

A driver may:

```text
Join
 ↓
Race
 ↓
Disconnect
 ↓
Reconnect
```

The system must not create an entirely new logical driver record for the same:

```text
steamId + carNumber
```

The existing object should be updated in place.

Starting position must remain immutable.

---

# 25. Incident Queue

ACCWeb should maintain an in-memory queue.

Conceptually:

```text
ACC LOG
   ↓
Detector
   ↓
Incident
   ↓
Local Queue
   ↓
Batcher
   ↓
HTTP Client
   ↓
Backend
```

The detector must not wait synchronously for the backend.

Bad:

```text
detect incident
     ↓
HTTP POST
     ↓
wait
     ↓
continue
```

Better:

```text
detect incident
     ↓
queue
     ↓
continue race processing

background sender
     ↓
batch
     ↓
POST
```

---

# 26. Incident Batching

Preferred API:

```http
POST /events/:eventId/incidents/batch
```

Example:

```json
{
  "eventId": "event_123",
  "serverId": "server_01",
  "sessionId": "session_001",

  "incidents": [
    {
      "incidentId": "inc_001",
      "type": "COLLISION",
      "timestamp": 1842.71,

      "participants": [
        {
          "steamId": "76561198789435655",
          "carNumber": 333
        },
        {
          "steamId": "76561198789435656",
          "carNumber": 21
        }
      ]
    },

    {
      "incidentId": "inc_002",
      "type": "TRACK_LIMIT",
      "timestamp": 1911.21,

      "participants": [
        {
          "steamId": "76561198789435655",
          "carNumber": 333
        }
      ]
    }
  ]
}
```

---

# 27. Batching Rules

The batcher should support:

- Maximum batch size
- Time-based flush
- Immediate flush for critical events
- Retry
- Backoff
- Idempotency
- Graceful shutdown

For example:

```text
Queue:
0 incidents → do nothing

Queue reaches N incidents:
    ↓
flush

OR

Timer expires:
    ↓
flush

OR

critical incident:
    ↓
optional immediate flush
```

The exact batch size and timer should be configurable rather than hard-coded if possible.

---

# 28. Retry Strategy

If the backend is temporarily unavailable:

```text
ACCWeb
   ↓
POST
   ↓
FAIL
   ↓
queue remains
   ↓
retry
```

Do not silently discard incidents.

Use:

```text
retry count
last attempt
next retry time
```

where appropriate.

If the server stops while incidents are still queued, persist the queue locally if practical so incidents are not lost.

---

# 29. Backend API

Implement:

```http
POST /events/:eventId/incidents/batch
```

Optional read APIs:

```http
GET /events/:eventId/incidents
GET /events/:eventId/incidents/:incidentId
```

Filtering should eventually support:

```text
steamId
carNumber
type
sessionId
lap
timestamp
```

Example:

```http
GET /events/event_123/incidents?steamId=76561198789435655
```

---

# 30. Backend Validation

Never trust collector input blindly.

Validate:

## Event

- Event exists
- Event is valid
- Server relationship is valid

## Session

- Session ID is valid
- Session belongs to the event/server where required

## Participant

- Steam ID exists or is valid
- Car number is valid

## Type

Incident type must be supported.

## Timestamp

Must be a valid numeric/time representation.

## Probability

Probability values must be:

```text
0 <= probability <= 1
```

For classification distributions, validate that values are consistent with the intended probability model.

---

# 31. MongoDB Incident Model

Create:

```text
src/models/Incident.js
```

Collection:

```text
incidents
```

Suggested fields:

```text
incidentId
eventId
serverId
sessionId

timestamp
timestampUtc

lap
sector

type

participants[]

location

severity
confidence

classification

evidence

telemetryWindow

processed

createdAt
updatedAt
```

Use indexes for common queries.

At minimum consider indexes around:

```text
eventId
sessionId
participants.steamId
type
timestamp
incidentId
```

The exact indexing strategy should follow the current MongoDB/Mongoose conventions in the repository.

---

# 32. Collision.js Migration

The existing:

```text
src/models/Collision.js
```

should be refactored/replaced.

Do not leave two competing concepts:

```text
Collision
Incident
```

The correct conceptual hierarchy is:

```text
Incident
   └── type = COLLISION
```

The API and model should use Incident terminology.

If existing collision data is already in MongoDB, preserve it through a migration or compatibility layer rather than deleting it.

The final system should use:

```text
incidents
```

as the canonical collection.

---

# 33. Race Results

Keep the existing race-result architecture.

At race completion:

```text
Qualifying Cache
       +
Race Results
       +
Cuts
       +
Invalid Laps
       +
Penalties
       ↓
ONE consolidated payload
       ↓
Backend
```

Example:

```json
{
  "eventId": "event_123",
  "serverId": "server_01",
  "sessionId": "race_001",

  "drivers": [
    {
      "steamId": "76561198789435655",
      "carNumber": 333,

      "startPosition": 1,
      "finishPosition": 1,

      "status": "FINISHED",

      "bestLapTime": 118387,
      "totalTime": 230526,
      "lapCount": 2,

      "cutsCount": 1,
      "invalidLapsCount": 1,

      "penalties": []
    }
  ]
}
```

Do not turn every lap into an HTTP request.

---

# 34. SA Integration

Do not attempt to build the complete SA algorithm in this implementation milestone.

This task establishes the evidence pipeline.

Later:

```text
Incident
   ↓
SA Engine
   ↓
SA change
```

Potential SA inputs:

```text
track limits
cuts
wheel spins
spins
damage
collision involvement
responsibility probability
unsafe rejoins
clean racing
incident frequency
incident severity
```

The important requirement now is that the Incident model contains enough evidence to support future calculations.

---

# 35. ELO Integration

ELO should primarily consume race results.

Flow:

```text
RaceResult
   ↓
ELO Engine
   ↓
Expected score
   ↓
Actual score
   ↓
ELO delta
   ↓
ELO history
```

The Incident system should not directly modify ELO unless a future design explicitly requires it.

Keep:

```text
SA = safety / incident evidence

ELO = competitive performance
```

as separate systems.

---

# 36. Rating History

Do not only store:

```text
SA = 87.2
ELO = 1841
```

Store history.

Example:

```text
Race #1083

SA:
87.1 → -2.4 → 84.7

ELO:
1841 → +22 → 1863
```

This allows:

- Graphs
- Historical analysis
- Debugging
- Recalculation
- Rating audits

---

# 37. Raw Telemetry Storage

Raw high-frequency telemetry should not be stored in the Incident MongoDB document.

Instead:

```text
data/
└── events/
    └── event_123/
        └── race_001/
            ├── session.json
            ├── results.json
            ├── incidents.json
            └── telemetry.parquet
```

The exact storage format can be implemented later.

The key architecture is:

```text
Raw telemetry
     ↓
local/dedicated storage

Incident metadata
     ↓
MongoDB
```

---

# 38. Telemetry Window Reference

An Incident can reference:

```json
{
  "telemetryWindow": {
    "available": true,

    "start": 1839.71,

    "impact": 1842.71,

    "end": 1844.71
  }
}
```

This tells future analysis where to retrieve the raw evidence.

---

# 39. ACCWeb Internal Structure

Keep the existing files where possible.

Recommended logical structure:

```text
internal/pkg/instance/

collector.go
    ↓
collector coordinator

collector_results.go
    ↓
result watcher
result parsing
qualifying cache
race results

collector_collisions.go
    ↓
collision detection
cut detection

incident.go
    ↓
generic Incident structure

incident_detector.go
    ↓
incident detection interfaces

incident_queue.go
    ↓
in-memory queue

incident_batcher.go
    ↓
batch creation

backend_client.go
    ↓
HTTP API communication
```

The exact file split should follow the existing repository style rather than creating unnecessary abstractions.

---

# 40. Detector Architecture

Use the generic concept:

```text
Detector
   ↓
Incident
```

Potential detectors:

```text
CollisionDetector
WheelSpinDetector
SpinDetector
TrackLimitDetector
DamageDetector
PenaltyDetector
UnsafeRejoinDetector
WrongWayDetector
```

Each detector should produce a normalized Incident.

Example:

```text
CollisionDetector
       ↓
Incident{
    Type: COLLISION
}
```

and:

```text
WheelSpinDetector
       ↓
Incident{
    Type: WHEEL_SPIN
}
```

This keeps the backend independent of how the event was detected.

---

# 41. Do Not Fabricate Data

This is a strict requirement.

If the current ACCWeb collector does not have access to:

```text
wheel slip
G-force
damage delta
heading
yaw rate
```

do not invent them.

Use:

```text
null
```

or omit optional fields.

The system should clearly distinguish:

```text
detected
```

from:

```text
estimated
```

and:

```text
not available
```

where necessary.

---

# 42. Existing Functionality Must Remain Working

Do not break:

- ACC server Start
- ACC server Stop
- Event ID configuration
- Runtime configuration locking
- Collector enable/disable
- Collector status
- Server ID synchronization
- Qualifying grid caching
- Race result processing
- Rejoin idempotency
- UTF-16LE result decoding
- UTF-8 BOM handling
- `sessionResult.leaderBoardLines` extraction
- Track-cut counting
- Invalid-lap counting
- Penalty extraction
- Collision detection

The current real race classification behavior must continue working.

---

# 43. Testing Strategy

## ACCWeb tests

Test:

### Collision conversion

Input:

```text
collision detected
car A
car B
timestamp
```

Expected:

```text
Incident
type = COLLISION
participants = A/B
```

### Wheel spin conversion

Expected:

```text
Incident
type = WHEEL_SPIN
```

only when required telemetry is available.

### Track limit

Repeated invalid frames should produce one meaningful event rather than many duplicates.

### Steam ID resolution

Verify:

```text
ACC car ID
    ↓
Steam ID
```

### Car number resolution

Verify:

```text
ACC car ID
    ↓
car number
```

### Batch

Given:

```text
25 incidents
```

verify they are sent according to batching rules.

### Retry

Simulate backend failure.

Verify incidents remain queued and are retried.

### Duplicate

Send the same incident twice.

Verify the backend stores only one.

---

# 44. Backend Tests

Test:

```text
create incident
batch incidents
duplicate incident
invalid incident type
invalid probability
invalid participant
invalid event
invalid server
incident retrieval
driver filtering
type filtering
session filtering
```

---

# 45. Integration Test

Simulate:

```text
ACC session starts
    ↓
driver joins
    ↓
race begins
    ↓
track-limit incident
    ↓
collision
    ↓
race finishes
    ↓
result file appears
    ↓
ACCWeb processes results
    ↓
incidents sent
    ↓
race results sent
    ↓
MongoDB updated
```

Verify:

```text
RaceResult exists
Incident records exist
No duplicate incidents
Driver identity is correct
Start position is preserved
Finish position is correct
Cuts are correct
Invalid laps are correct
```

---

# 46. Example End-to-End Race

Assume:

```text
Event:
event_123

Server:
server_01

Driver:
MK 01

Steam ID:
76561198789435655

Car:
333
```

Race begins.

ACCWeb tracks:

```text
Steam ID
Car number
Lap
Position
Session state
```

Driver cuts once.

ACCWeb detects:

```text
TRACK_LIMIT
```

and increments:

```text
cutsCount
```

Later driver spins.

ACCWeb creates:

```text
SPIN incident
```

Later driver contacts another car.

ACCWeb creates:

```text
COLLISION incident
```

The incidents remain in the local queue.

At a flush point:

```text
3 incidents
    ↓
one batch
    ↓
backend
```

Race finishes.

ACCWeb reads:

```text
results.json
```

and creates one consolidated race result.

Backend now contains:

```text
RaceResult
    ↓
startPosition
finishPosition
bestLapTime
totalTime
lapCount
cutsCount
invalidLapsCount
penalties
```

and:

```text
Incidents
    ↓
TRACK_LIMIT
SPIN
COLLISION
```

The SA engine can later consume these incidents.

The ELO engine consumes the race result.

---

# 47. Dashboard Data

The web GUI should be able to request:

```text
Driver
 ↓
Race history
 ↓
Incidents
 ↓
SA
 ↓
ELO
```

Example:

```text
Driver: MK 01

SA: 87.4
ELO: 1863

Recent incidents:

Lap 5
TRACK_LIMIT

Lap 12
WHEEL_SPIN

Lap 17
COLLISION
Classification: REAR_END
Confidence: 93%
```

This creates an explainable rating system rather than just displaying a mysterious number.

---

# 48. API Separation

Recommended:

## Collector lifecycle

```http
POST /events/:eventId/collector/start
POST /events/:eventId/collector/heartbeat
POST /events/:eventId/collector/stop
```

## Incidents

```http
POST /events/:eventId/incidents/batch
GET  /events/:eventId/incidents
GET  /events/:eventId/incidents/:incidentId
```

## Results

Use the existing race-result endpoint pattern.

Do not break the current endpoint unnecessarily.

---

# 49. Security

The collector API should eventually authenticate the ACCWeb instance.

At minimum, validate:

```text
serverId
eventId
collector identity
```

The backend should not accept arbitrary incident submissions from unauthenticated clients.

Do not expose internal collector endpoints publicly without authentication.

---

# 50. Failure Handling

If backend goes down:

```text
ACC
 ↓
ACCWeb
 ↓
incident queue
 ↓
backend unavailable
```

Race processing must continue.

The ACC server must not depend on backend availability.

When backend returns:

```text
queue
 ↓
retry
 ↓
backend
```

This is an important requirement for a race platform.

---

# 51. Graceful Shutdown

When ACCWeb stops:

1. Stop accepting new detector events.
2. Flush incident queue.
3. Send remaining batches.
4. Persist unsent data if necessary.
5. Stop collector.
6. Notify backend collector status.

Do not lose the final incidents simply because the ACC server stopped.

---

# 52. Performance Requirements

The design should ensure:

```text
ACC processing
    ↓
does not wait on HTTP
```

Backend communication should run asynchronously.

MongoDB should receive:

```text
race result payload
incident payloads
```

rather than:

```text
every telemetry sample
```

This keeps the system scalable when multiple ACC servers are running.

---

# 53. Multiple Server Support

The architecture must work with:

```text
Server 1 → ACCWeb 1
Server 2 → ACCWeb 2
Server 3 → ACCWeb 3
...
```

Each collector sends:

```text
eventId
serverId
sessionId
```

This ensures incidents can be associated with the correct race.

Example:

```text
server_01
    ↓
event_100
    ↓
race_001

server_02
    ↓
event_101
    ↓
race_002
```

The backend must never mix these incidents.

---

# 54. Event/Server Relationship

The existing automatic synchronization should remain:

```text
ACCWeb Start
     ↓
server starts
     ↓
collector starts
     ↓
event updated
```

Example Event state:

```text
serverId = instance_01
collectorEnabled = true
collectorStatus = running
```

When stopped:

```text
collectorEnabled = false
collectorStatus = stopped
```

This remains independent from the Incident system.

---

# 55. Data Ownership

Use this rule:

```text
ACCWeb owns collection and detection.

Backend owns persistence and platform-level calculations.

MongoDB owns durable storage.

Dashboard owns presentation.
```

This prevents responsibilities from becoming mixed.

---

# 56. What NOT to Implement in This Milestone

Do NOT turn this implementation into a giant SA/stewarding project.

Do not implement:

- Final SA formula
- Final ELO formula
- AI collision responsibility
- Machine learning model
- Automatic penalties
- Complex steward UI

unless required by existing code.

The current milestone is:

```text
ACC data
 ↓
Incident pipeline
 ↓
Backend persistence
```

The stored evidence will enable the later systems.

---

# 57. Future SA Architecture

Later:

```text
Incident
    │
    ├── type
    ├── severity
    ├── participants
    ├── evidence
    └── responsibility
            ↓
         SA Engine
            ↓
        SA Change
```

Example:

```text
Collision
    ↓
Driver A probability = 0.93
    ↓
severity = 0.72
    ↓
SA penalty
```

The exact formula should be calibrated using real race data.

---

# 58. Future Stewarding Architecture

Later:

```text
Incident
    ↓
Telemetry evidence
    ↓
Classification
    ↓
Responsibility probability
    ↓
Decision
```

Example:

```json
{
  "classification": {
    "type": "REAR_END",

    "probabilities": {
      "driverA": 0.93,
      "driverB": 0.02,
      "racingIncident": 0.05
    }
  }
}
```

The system can eventually use thresholds such as:

```text
high confidence
    ↓
automatic decision

medium confidence
    ↓
review required

low confidence
    ↓
racing incident/manual review
```

These thresholds should be treated as future calibration parameters, not hard-coded truth.

---

# 59. Final File Structure

A reasonable final architecture is:

```text
ACCWeb
│
├── internal/pkg/instance/
│   ├── collector.go
│   ├── collector_results.go
│   ├── collector_collisions.go
│   ├── incident.go
│   ├── incident_detector.go
│   ├── incident_queue.go
│   ├── incident_batcher.go
│   └── backend_client.go
│
└── existing files

SimSports Backend
│
├── src/models/
│   ├── Event.js
│   ├── RaceResult.js
│   └── Incident.js
│
├── src/controllers/
│   ├── raceResult.controller.js
│   └── incident.controller.js
│
├── src/routes/
│   └── incident routes
│
└── existing files
```

Do not blindly create every file if the repository already has an equivalent abstraction. Reuse the existing project structure.

---

# 60. Implementation Order

Implement in this order.

## Phase 1 — Inspect

Inspect:

```text
collector.go
collector_results.go
collector_collisions.go
instance.go
handler_instances.go

RaceResult.js
raceResult.controller.js
Collision.js
Event.js
routes
```

Understand the existing architecture before changing it.

---

## Phase 2 — Generic Incident Model

Create:

```text
Incident
```

with:

```text
incidentId
eventId
serverId
sessionId
timestamp
type
participants
```

Add optional evidence fields.

---

## Phase 3 — Backend API

Implement:

```http
POST /events/:eventId/incidents/batch
```

Add:

- validation
- idempotency
- MongoDB persistence
- `incidents` collection

---

## Phase 4 — ACCWeb Incident Conversion

Convert existing collision detection:

```text
Collision
    ↓
Incident(type=COLLISION)
```

Do not change the underlying detection unless necessary.

---

## Phase 5 — Queue

Add:

```text
IncidentQueue
```

Detectors add to the queue.

No synchronous HTTP calls.

---

## Phase 6 — Batcher

Implement:

```text
queue
 ↓
batch
 ↓
backend
```

with retry.

---

## Phase 7 — Existing Race Results

Verify that the existing race result pipeline still works.

Do not replace it unnecessarily.

---

## Phase 8 — Additional Incident Types

Add supported detectors only where reliable data exists:

```text
TRACK_LIMIT
WHEEL_SPIN
SPIN
LOSS_OF_CONTROL
DAMAGE
PENALTY
UNSAFE_REJOIN
```

Do not fabricate unavailable telemetry.

---

## Phase 9 — Tests

Run:

```text
unit tests
integration tests
collector tests
backend tests
real ACC race test
```

---

# 61. Acceptance Criteria

The implementation is complete when all of the following are true.

### Collector

- ACCWeb starts normally.
- ACC server lifecycle remains functional.
- Event ID remains functional.
- Collector status synchronization remains functional.
- Existing result watcher remains functional.
- Existing qualifying cache remains functional.
- Existing race result processing remains functional.
- Existing collision detection remains functional.

### Incident system

- Generic Incident object exists.
- Collision is represented as `type = COLLISION`.
- `incidents` is the canonical MongoDB collection.
- `collisions` is no longer the canonical collection.
- Steam IDs are stored.
- Car numbers are stored.
- Timestamp is stored.
- Incident type is stored.
- Event/server/session relationship is stored.
- Probability/classification is supported.
- Evidence is supported.
- Telemetry-window reference is supported.
- Single-driver incidents are supported.
- Multi-driver incidents are supported.

### Networking

- Raw telemetry is not continuously POSTed.
- Incidents are queued.
- Incidents are batched.
- Backend requests run asynchronously.
- Failed requests are retried.
- Duplicate incidents are prevented.

### Race results

- Race results remain consolidated.
- Start position remains correct.
- Finish position remains correct.
- Best lap remains correct.
- Total time remains correct.
- Lap count remains correct.
- Cuts remain correct.
- Invalid laps remain correct.
- Penalties remain correct.
- Rejoin behavior remains correct.

### Database

- `incidents` collection exists.
- Unique incident identity is enforced.
- Useful indexes exist.
- Existing race-result data remains intact.

---

# 62. Final Architecture

The finished system should look like:

```text
                         ACC SERVER
                             │
                             ▼
                      ┌─────────────┐
                      │   ACCWeb    │
                      │             │
                      │ Collector   │
                      │ Parser      │
                      │ Detectors   │
                      │ Aggregator  │
                      │ Queue       │
                      │ Batcher     │
                      └──────┬──────┘
                             │
                ┌────────────┴────────────┐
                │                         │
         Local Evidence              Backend API
                │                         │
         Raw telemetry              Race Results
         session data               Incidents
                │                         │
                │                         ▼
                │                    MongoDB
                │                         │
                │                  ┌──────┴──────┐
                │                  │             │
                │                SA Engine     ELO Engine
                │                  │             │
                │                  └──────┬──────┘
                │                         │
                │                    Driver Rating
                │                         │
                └─────────────────────────┘
```

---

# 63. Core Principle

The most important rule for this implementation is:

> **ACCWeb processes the high-frequency ACC data locally. The backend receives durable, meaningful, normalized race evidence—not a telemetry firehose.**

The data hierarchy is:

```text
RAW ACC DATA
      ↓
NORMALIZED DATA
      ↓
EVENTS
      ↓
INCIDENTS
      ↓
RACE RESULTS
      ↓
SA / ELO / STEWARDING
```

And the most important data model decision is:

```text
Incident
│
├── COLLISION
├── WHEEL_SPIN
├── SPIN
├── LOSS_OF_CONTROL
├── TRACK_LIMIT
├── CUT
├── UNSAFE_REJOIN
├── DAMAGE
├── PENALTY
└── WRONG_WAY
```

not:

```text
Collision
Cuts
Spins
WheelSpins
...
```

as unrelated database concepts.

This gives the platform one consistent evidence layer that can later support safety rating, driver history, automated stewarding, and analytics without redesigning the collector/backend architecture.
