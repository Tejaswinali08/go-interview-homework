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

How it works

The API reads data from Postgres and returns it through GraphQL.

- `users` and `user(id)` return user data
- `tasks` and `task(id)` return task data
- `tasks` supports optional filtering by `status` and `userId`
- `createTask` inserts a new task into the database
- `updateTaskStatus` updates the status of an existing task

Important fixes made during implementation

1. Seed SQL fix  
The seed script had a SQL issue that had to be corrected before seeding the database successfully.

2. Database connection fix  
The API initially used the wrong Postgres credentials. I updated it to use the homework values:

- host: `localhost`
- port: `5432`
- user: `admin`
- password: `todo`
- database: `homework`

3. Missing `tags` column  
The GraphQL model expected `tags`, but the database table did not include it. I added the missing `tags` column to the `tasks` table so the schema and API matched.

4. Nullable field handling  
Some seeded rows had `NULL` values for `description`, which caused scanning errors in Go. I handled this using `COALESCE` in the SQL queries.

5. Status enum mapping  
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

6. Nested user resolver fix  
The nested `user` field on `Task` needed to handle both `Task` and `*Task` so that it worked for both:
- `tasks`
- `task(id)`

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

Assumptions

- Docker/Postgres is started through `docker compose`
- Postgres is available on `localhost:5432`
- The seeded schema is the source of truth for users and tasks
- The homework expects the GraphQL API to be implemented in Go even though the starter repo did not include a ready API server entrypoint

Limitations

- The implementation is currently in a single file: `cmd/api/main.go`
- For a production setup, I would split this into separate packages for:
  - schema
  - resolvers
  - database access
  - models
- Error handling and validation could be expanded further
- Database migrations for schema changes like `tags` could be formalized instead of applied manually

Summary

The GraphQL service is working locally, connected to the seeded Postgres database, and the required queries and mutations are implemented and verified end to end.
