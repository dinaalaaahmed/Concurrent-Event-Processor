# Concurrent Event Processor using GoLang


## Prerequisites
- Docker

## Description
- Following [this](https://www.youtube.com/watch?v=s3XItrqfccw&t=7158s) tutorial to build GO Lang Production REST API

## How to RUN

- To run Project

```docker compose up```


- To run go run -race

### 1: Make sure db is running By running following command

``` docker compose up events-processing-postgres```

### 2: In another Terminal run go run -race with the next command
```
  docker run --rm -it `
    --network nahdi-assesment_default `
    -p 3000:3000 `
    -v ${PWD}:/app `
    -w /app `
    -e CGO_ENABLED=1 `
    -e DB_DSN="postgres://postgres:mysecretpassword@events-processing-postgres:5432/events-processing?sslmode=disable" `
    golang:1.25 go run -race ./cmd/...
```


### 3: Test concurrent requests with the next curl

```
for i in {1..100}; do
  curl -X POST http://localhost:3000/events \
    -H "Content-Type: application/json" \
    -d "{
        \"user_id\": \"user-$i\",
        \"event_type\": \"click\",
        \"value\": 1,
        \"timestamp\": \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\"
    }" &
done
wait
```

## Architecture Overview
The service follows Clean Architecture principles, ensuring a strict separation of concerns between transport (HTTP), business logic (Service), and data persistence (Repository).

### Core Components:
- Transport Layer (Handler): POST /events: Validates JSON input and dispatches jobs to the worker pool.

- GET /stats?user_id={user_id}: Queries the service for aggregated data.

- Worker Pool (Concurrency Manager):
  - A fixed set of goroutines that pull from a buffered Job channel.
  - Implements Backpressure by rejecting new events when the buffer is full (returning HTTP 429).

- Service Layer:
  - Contains the business logic for calculating statistics.
  - Uses a sync.RWMutex to protect an in-memory cache for high-speed reads, ensuring thread-safe reads, and writes to the in-memory cache. While this is optimal for a single-node deployment, a horizontally scaled architecture would require a distributed in-memory cache like Redis to maintain data consistency across multiple instances. Additionally, a distributed lock (e.g., via Redis) would replace the local Mutex to prevent data divergence.

- Data Layer (SQLC/Postgres):
  - Handles persistent storage of events.
  - Uses pgxpool for thread-safe connection management.
  - Created Composite Index on (user_id, event_type) to increase performance for statics query.


### Design Decisions and Trade-offs
- Concurrency: Worker Pool vs. Goroutine-per-Request
  - Decision: I implemented a fixed-size worker pool with a buffered channel.
  - Trade-off: While spawning a goroutine per request is easy, it risks crashing the service under a spike (OOM or DB connection exhaustion). The worker pool provides predictable resource limits. I implemented Backpressure using a select statement in the Enqueue method; if the buffer is full, the system returns a 429 Too Many Requests, preserving the health of the existing workers.

- Synchronization: in memory map protected with Mutex vs. Quering db for every GET /stats request
  - Decision: I used a sync.RWMutex to protect an in-memory map[string]map[string]int64.
  - Trade-off: I chose a Read-Write Mutex because the GET /stats endpoint is expected to be read-heavy. which is significantly faster than calling the db for every read.

- Reliability: Graceful Shutdown with WaitGroups
  - Decision: I integrated sync.WaitGroup into the worker lifecycle.
  - Trade-off: To ensure Data Integrity, the application catches OS interrupt signals (SIGINT/SIGTERM). It closes the job channel (draining the buffer) and uses wg.Wait() to ensure workers finish their current database transactions before the process exits. This prevents "half-written" events during deployments.

- Scalability: Local Memory vs. Distributed Cache
  - Decision: For this assignment, I utilized local memory for speed and simplicity.
  - Trade-off: I recognize that a sync.RWMutex only scales vertically. In a horizontally scaled environment (multiple pods), this would lead to data divergence.
  - Future Roadmap: To scale this service, I would replace the local map with a Distributed Cache (Redis) and use Distributed Locking or Atomic Increments (INCRBY) to ensure consistency across all nodes.

- Database Connection Management: Worker Pool vs. Pool Size  
  - Decision: I configured the pgxpool size to closely align with the worker pool count, maintaining a 1:1.3 ratio (e.g., 10 connections for 13 workers).
  - Trade-off: This setup ensures that most active jobs have immediate access to a database connection, significantly reducing "Acquire" latency. By allowing 30% more workers than connections, the system remains efficient. while the additional workers can simultaneously handle CPU-bound tasks.