Reviewed by: Tejaswi Nali
Senior Full Stack Developer

Implemented

I built the GraphQL API in `cmd/api/main.go` and connected it to the Postgres database provided by the homework setup.

The API now supports:

- `User`
- `Task`
- `TaskStatus`

Queries implemented:

- `user(id: ID!)`
- `users`
- `task(id: ID!)`
- `tasks(status: TaskStatus, userId: ID)`

Mutations implemented:

- `createTask(userId, title, description, dueDate, tags)`
- `updateTaskStatus(id, status)`

What I changed

I created the API entrypoint in `cmd/api/main.go` and added:

- GraphQL schema setup
- GraphQL types for `User` and `Task`
- `TaskStatus` enum
- query resolvers
- mutation resolvers
- Postgres connection
- HTTP handler for `/graphql`

Library / structure choices

I used `github.com/graphql-go/graphql` together with `github.com/graphql-go/handler`.

I chose this library because it was the fastest way to build a small GraphQL API in Go without adding code generation or extra setup. For speed, I kept the implementation in `cmd/api/main.go`. With more time, I would split it into separate packages for schema, resolvers, models, and database access.

How it works

The API reads data from Postgres and returns it through GraphQL.

- `users` and `user(id)` return user data
- `tasks` and `task(id)` return task data
- `tasks` supports optional filtering by `status` and `userId`
- `createTask` inserts a new task into the database
- `updateTaskStatus` updates the status of an existing task

Important fixes made during implementation

1. Seed SQL fix  
The seed script had a SQL issue in `cmd/seed/main.go`. I found it by running `go run ./cmd/seed`, reading the Postgres error, and tracing it back to the insert/query in the seed file. I fixed the SQL so the seed completed successfully.

2. Database connection fix  
The API initially used the wrong Postgres credentials. I updated it to use the homework values:

- host: `localhost`
- port: `5432`
- user: `admin`
- password: `todo`
- database: `homework`

3. Nullable field handling  
Some seeded rows had `NULL` values for `description`, which caused scanning errors in Go. I handled this using `COALESCE` in the SQL queries.

4. Status enum mapping  
Postgres stores task status values in lowercase like:

- `pending`
- `in_progress`
- `done`

GraphQL expects:

- `PENDING`
- `IN_PROGRESS`
- `DONE`

I added mapping in both directions:
- GraphQL to Postgres for writes and filters
- Postgres to GraphQL for API responses

5. Nested user resolver fix  
The nested `user` field on `Task` needed to handle both `Task` and `*Task` so that it worked for both:
- `tasks`
- `task(id)`

6. Tags handling  
I aligned tag handling with the provided schema through `task_tags`.

New field

I picked `dueDate` as the field to expose end to end.

I chose it because it already existed in the database and was the simplest field to expose cleanly through the database, API, and UI.

Validation completed

I tested the following successfully in GraphiQL:

- `users`
- `user(id: "1")`
- `tasks`
- `task(id: "1")`
- `tasks(status: DONE)`
- `tasks(userId: "1")`
- `tasks(status: DONE, userId: "1")`
- `createTask(...)`
- `updateTaskStatus(...)`

I also verified the browser UI loads users and tasks and displays `dueDate`.

Tradeoffs / what I skipped

- The implementation is currently in a single file: `cmd/api/main.go`
- I did not add a full automated test suite
- I did not add structured logging
- I did not add a Dockerfile for the API

With more time, I would:
- split the API into packages
- add tests for resolvers and database queries
- improve validation and error handling
- clean up migrations and project structure further

How to run everything from a fresh clone

1. `git clone <repo-url>`
2. `cd go-interview-homework`
3. `docker compose up -d`
4. `go run ./cmd/seed`
5. `go run ./cmd/api`
6. In another terminal: `python3 -m http.server 8081 --directory web`
7. Open GraphQL at `http://localhost:8080/graphql`
8. Open the UI at `http://localhost:8081`

Summary

The GraphQL service is working locally, connected to the seeded Postgres database, and the required queries and mutations are implemented and verified end to end. I also exposed `dueDate` through the UI.
AI usage:
- I used AI tools for debugging help and implementation guidance, and I verified the final code and behavior myself.
