# Backend System Design

Notas tomada del curso [BACKEND SYSTEM DESIGN](https://frontendmasters.com/courses/backend-system-design).

## What is a SD?

I could say a collection of component that work together and has inputs and outpus and it has boundaries.

## What is a distributed System?

- Miltiple copmputer working together as a single system.
- Components may be physically separeted.
- Sesigned to handle failed and scale across multiple locations.

| Components     | Role                            |
| -------------- | ------------------------------- |
| Client         | Send request and display data.  |
| Database       | Store data.                     |
| Serve          | Process request business logic. |
| Load balancers | Distributed traffic             |
| Cache          | Temporary storage data.         |

## How to build anything?

### Core elementes of design system

- Translating business requirements.
- Desinging API and architecture.
- Understandings technologies and trade off

### Strategy

- Scope the problem.
- Design a high level architecture.
- Address key challenges and trade off.

### Functional requirements

Ask questions like: "What are the requirements?".

- Functional: describe "WHAT" the system should do.
- Non-functional: describe "HOW" the system should perfom.

## CAP Theorem

Any distributed system can only guarantee 2 out of 3 at the same time:
C = Consistency
A = Availability
P = Partition tolerance

C+P = Always show the lastest data but ureliable perfomence.
A+P = Always responde but data might be out of date

## System Quality

| System Quality |                                                                        |
| -------------- | ---------------------------------------------------------------------- |
| Reliability    | .                                                                      |
| Observability  | The ability to know what is happening to your system.                  |
| Security       | The ability to safeguard the system and its data.                      |
| Scalability    | The ability to handle increases and decreases in system usage.         |
| Adaptability   | The ability to handle changing requirements or user behaviors.         |
| Perfomance     | Latency: How quikly does the system respond                            |
| Perfomance     | Throughput: How much data can move through the system at a given time. |

## High-Level design

1. Modeling
   - Requirements -> Entity modeling -> API Design -> Endpoint (optional)

   - Entity modeling:
     Define the main functional elements.

   - API Design:
     Define the actions and operations of the system

2. Architectural Design

### Protocols

#### HTTP (Hypertext Transport Protocol)

Key Characteristics:

- Simple
- Human readable
- Supported by all browsers
- Stateless

Commom use cases:

- Web browsers

#### WebSockets

Key characteristics:

- bi-directional communication.
- persistent connection
- low-latency
- stateful

Commom use cases:

- chat app
- live dashboards
- collaborative editing

#### Server - Sent Events

Key characteristics:

- one way communication (server to client)
- human readable

Commom use cases

- News feeds
- Status updates
- Stock tickers

#### gRPC (Remote procedure call)

Key characteristics

- binary protocol (http/2)
- strongly-typed contracts (proto buffs)
- requires code generation

Common use cases

- microservice communication
- perfomance critical systems
- loT devices

#### REST (Representational state transfer)

Key characteristics

- multiple endpoints
- human readable supported by all browsers
- stateless

Common use cases

- single-sources of data.
- CRUD apps.
- easily cached data.

#### GraphQL

Key characteristics

- single endpoint
- precise data retrieval
- self documenting api
- strongly typed

Common use cases

- complex or multiple sources of data
- app supporting multiple client types
- decoupling frontend from backend development

#### Protocol cheat sheet

![image](./protocol-cheat-sheet.png)

## Scaling

Vertical scaling: more power (CPU, RAM, GPU, etc) to a machine.
Pros:

- simple to implement
- no code changes required
- easier maintenance

Cons:

- physical limits of hardware
- expensive to scale up/down
- decreased resilency

Horizontal scaling: more machines
Pros:

- easily scales up/down with traffic
- high availability
- increased fault tolerance

Cons

- requires orchestration
- can require code changes

## Load Balancers

- Distributes traffic evenly across services
- Can act as a gateway for routing
- Handles health checks and failovers

| Algorithm         |                                                                  |
| ----------------- | ---------------------------------------------------------------- |
| Round Robin       | Requests distributed sequentially to each server in rotation.    |
| Least Connections | Sends requests to server with fewest active connections.         |
| Least Latency     | Selects server with lowest response time.                        |
| IP Hash           | Uses client IP address to determine server (consistent hashing). |

## Data Storage

"At the end of the day, mostr of what you do is reading from and writing to a database"

### Dimensions

structed vs unstructured

persistent vs ephemeral

read-optimized vs write-optimized

consistency vs availability

### Relational

- "SQL"
- Structued data with relationships
- Enforced schema
- ACID transactions

- Atomicity: Each transaction is all or nothing. Either every operation succeeds or none do.
- Consistent: Transactions always follows the rules set for the database.
- Isolated: Concurrent transactions do not interfere with each other. Each runs as if it were alone.
- Durable: Once a transaction is saved, its changes are permanent.

### Non-Relational

- "NoSQL"
- Semi or unstructured data
- Flexible
- Horizontal scale

- Document: stores data as flexible, structured documents (like JSON).
- Key-value: stores data as simple pairs of a unique key and its value.
- Column: organizes data into columns into columns instead of rows for fast retrieval of similar data.
- Graph: stores data as nodes and relationships, making it easy to represent and query connections.

### How do we scale our data storage?

- Sharding
- Partitioning

## Caching

- Reduces latency
- Imporves user experience
- Decreases system load
- Lowers costs

### Type of caching

- Cache Aside (lazy loading)
  - cache miss
  - read from database
  - update cache
- Read through
  - read from cache
  - on miss, read from db
  - write to cache
- Write behind
  - write to cache
  - immediately return
  - asynchronously, write to db

### Caching trade off

perfomance vs freshness

### Caching invalidations

- time based expiration (ttl)
- event-based
- version tagging
- refresh ahead

### Cache evicton

- FIFO: first in first out
- LIFO: last in first out
- LRU: least recently used
- MRU: most recently used
- LFU: least frequently used
- RR: random replacement

## Estimations

- Help ground vague requirements in reality
- Helps you thing about the specifics of the system components
- Shows the interviewer yur thought process
- Dont have to be precise

## Strategy

- Clarify
  a. What are you estimating?
  i. users, request, storage
  b. Ask or make reasonable assumptions
  i. how many users etc
  c. Validate your assumptions
  i. write it down
- Do math
- Sanity check your results

### Example

how big will our bookmarks table get in a month if we have 100000 users and a bookmark is 500 bytes?

- Assumptions
  3 bookmarks per day
  30 days in a month

- Calculations
  Tasks per months: 3 * 30*1000 = 90000 bookmarks
  90k \* 500 bytes = 45mm bytes
  convert to mb:45mm / 1mm = 45mb per month

## Security

- SSL/.TLS termination is the process of decrypting encrypted traffic
- Typically happens at the edge of your infrastructure
- Converts HTTPS connections to http internally

### strategy

- termination at load balancer:
  most common approach
  application servers receive http

- Termination at application layer
  load balancer passes through encrypted traffic
  each application instance handles decryption

- Re-encryption
  Terminate at load balancer
  re-encript between load balancer and applications

### Authentication vs Authorization

- Authentication: verifies identity
  - common methods:
  * username/password
  * MFA
  * OAuth/Social Login
  * Biometrics
  * Certificate-based
  * Single sign on (SSO)
  - Stateful:
  * Session-based
  * Server stores session
  * Fast invalidation
  - Stateless
  * Token-based
  * Client stores token
  * More complex invalidation

Summary: Stateless architecture make scaling easier

- Authorization: Determines permissions
  - Auth layers
  * API gateway level
  * Service level
  * Database level
  * Object/data level
