package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type User struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type Task struct {
	ID          int      `json:"id"`
	UserID      int      `json:"userId"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	DueDate     string   `json:"dueDate"`
	Tags        []string `json:"tags"`
}

var db *sql.DB

func idFromArg(v interface{}) int {
	switch t := v.(type) {
	case string:
		n, _ := strconv.Atoi(t)
		return n
	case int:
		return t
	case int64:
		return int(t)
	default:
		return 0
	}
}
func toGraphQLStatus(s string) string {
	switch s {
	case "pending":
		return "PENDING"
	case "in_progress":
		return "IN_PROGRESS"
	case "done":
		return "DONE"
	default:
		return ""
	}
}
func getAllUsers() ([]User, error) {
	rows, err := db.Query(`SELECT id, email, name FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func getTaskByID(id int) (*Task, error) {
	t := &Task{}

	err := db.QueryRow(`
		SELECT id, user_id, title, COALESCE(description, ''), status,
		       COALESCE(due_date::text, '')
		FROM tasks
		WHERE id = $1
	`, id).Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.DueDate)
	if err != nil {
		return nil, err
	}

	t.Status = toGraphQLStatus(t.Status)

	tags, err := getTagsForTask(t.ID)
	if err != nil {
		return nil, err
	}
	t.Tags = tags

	return t, nil
}

func getTasks(status string, userID int, hasStatus bool, hasUserID bool) ([]Task, error) {
	query := `
		SELECT id, user_id, title, COALESCE(description, ''), status,
		       COALESCE(due_date::text, '')
		FROM tasks
	`

	var conditions []string
	var args []interface{}

	if hasStatus {
		args = append(args, status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}

	if hasUserID {
		args = append(args, userID)
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", len(args)))
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY id"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task

		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Status, &t.DueDate); err != nil {
			return nil, err
		}

		t.Status = toGraphQLStatus(t.Status)

		tags, err := getTagsForTask(t.ID)
		if err != nil {
			return nil, err
		}
		t.Tags = tags

		tasks = append(tasks, t)
	}

	return tasks, rows.Err()
}

func getTasksByUser(userID int, status string, hasStatus bool) ([]Task, error) {
	return getTasks(status, userID, hasStatus, true)
}

func getTagsForTask(taskID int) ([]string, error) {
	rows, err := db.Query(`SELECT tag FROM task_tags WHERE task_id = $1 ORDER BY tag`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return tags, rows.Err()
}

func insertTags(taskID int, tags []string) error {
	for _, tag := range tags {
		_, err := db.Exec(`INSERT INTO task_tags (task_id, tag) VALUES ($1, $2)`, taskID, tag)
		if err != nil {
			return err
		}
	}
	return nil
}
func getUserByID(id int) (*User, error) {
	u := &User{}
	err := db.QueryRow(`SELECT id, email, name FROM users WHERE id = $1`, id).Scan(&u.ID, &u.Email, &u.Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}
func toDBStatus(s string) string {
	switch s {
	case "PENDING":
		return "pending"
	case "IN_PROGRESS":
		return "in_progress"
	case "DONE":
		return "done"
	default:
		return s
	}
}
func main() {
	// If this does not match your seed file, replace this line with the exact DB string from cmd/seed/main.go
	connStr := "postgres://admin:todo@localhost:5432/homework?sslmode=disable"

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	taskStatusEnum := graphql.NewEnum(graphql.EnumConfig{
		Name: "TaskStatus",
		Values: graphql.EnumValueConfigMap{
			"PENDING":     &graphql.EnumValueConfig{Value: "PENDING"},
			"IN_PROGRESS": &graphql.EnumValueConfig{Value: "IN_PROGRESS"},
			"DONE":        &graphql.EnumValueConfig{Value: "DONE"},
		},
	})

	var userType *graphql.Object
	var taskType *graphql.Object

	userType = graphql.NewObject(graphql.ObjectConfig{
		Name: "User",
		Fields: graphql.FieldsThunk(func() graphql.Fields {
			return graphql.Fields{
				"id":    &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
				"email": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"name":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"tasks": &graphql.Field{
					Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(taskType))),
					Args: graphql.FieldConfigArgument{
						"status": &graphql.ArgumentConfig{Type: taskStatusEnum},
					},
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						var u User
						switch src := p.Source.(type) {
						case User:
							u = src
						case *User:
							u = *src
						default:
							return []Task{}, nil
						}

						status, hasStatus := p.Args["status"].(string)
						if hasStatus {
							status = toDBStatus(status)
						}

						return getTasksByUser(u.ID, status, hasStatus)
					},
				},
			}
		}),
	})

	taskType = graphql.NewObject(graphql.ObjectConfig{
		Name: "Task",
		Fields: graphql.FieldsThunk(func() graphql.Fields {
			return graphql.Fields{
				"id":          &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
				"title":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
				"description": &graphql.Field{Type: graphql.String},
				"status":      &graphql.Field{Type: graphql.NewNonNull(taskStatusEnum)},
				"dueDate":     &graphql.Field{Type: graphql.String},
				"tags":        &graphql.Field{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String)))},
				"user": &graphql.Field{
					Type: graphql.NewNonNull(userType),
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						switch t := p.Source.(type) {
						case Task:
							return getUserByID(t.UserID)
						case *Task:
							return getUserByID(t.UserID)
						default:
							return nil, nil
						}
					},
				},
			}
		}),
	})
	rootQuery := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"user": &graphql.Field{
				Type: userType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return getUserByID(idFromArg(p.Args["id"]))
				},
			},
			"users": &graphql.Field{
				Type: graphql.NewList(userType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return getAllUsers()
				},
			},
			"task": &graphql.Field{
				Type: taskType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return getTaskByID(idFromArg(p.Args["id"]))
				},
			},
			"tasks": &graphql.Field{
				Type: graphql.NewList(taskType),
				Args: graphql.FieldConfigArgument{
					"status": &graphql.ArgumentConfig{Type: taskStatusEnum},
					"userId": &graphql.ArgumentConfig{Type: graphql.ID},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					status, hasStatus := p.Args["status"].(string)

					dbStatusMap := map[string]string{
						"PENDING":     "pending",
						"IN_PROGRESS": "in_progress",
						"DONE":        "done",
					}

					if hasStatus {
						status = dbStatusMap[status]
					}

					userID := 0
					hasUserID := false
					if raw, ok := p.Args["userId"]; ok {
						userID = idFromArg(raw)
						hasUserID = true
					}

					return getTasks(status, userID, hasStatus, hasUserID)

				},
			},
		},
	})

	rootMutation := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"createTask": &graphql.Field{
				Type: taskType,
				Args: graphql.FieldConfigArgument{
					"userId":      &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"title":       &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"description": &graphql.ArgumentConfig{Type: graphql.String},
					"dueDate":     &graphql.ArgumentConfig{Type: graphql.String},
					"tags":        &graphql.ArgumentConfig{Type: graphql.NewList(graphql.String)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					userID := idFromArg(p.Args["userId"])
					title := p.Args["title"].(string)

					description, _ := p.Args["description"].(string)
					dueDate, _ := p.Args["dueDate"].(string)

					var tags []string
					if rawTags, ok := p.Args["tags"].([]interface{}); ok {
						for _, v := range rawTags {
							if s, ok := v.(string); ok {
								tags = append(tags, s)
							}
						}
					}

					var newID int
					err := db.QueryRow(`
						INSERT INTO tasks (user_id, title, description, status, due_date, tags)
						VALUES ($1, $2, $3, $4, $5, $6)
						RETURNING id
					`, userID, title, description, "pending", dueDate, pq.Array(tags)).Scan(&newID)
					if err != nil {
						return nil, err
					}

					return getTaskByID(newID)
				},
			},
			"updateTaskStatus": &graphql.Field{
				Type: taskType,
				Args: graphql.FieldConfigArgument{
					"id":     &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
					"status": &graphql.ArgumentConfig{Type: graphql.NewNonNull(taskStatusEnum)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					id := idFromArg(p.Args["id"])
					status := p.Args["status"].(string)
					dbStatusMap := map[string]string{
						"PENDING":     "pending",
						"IN_PROGRESS": "in_progress",
						"DONE":        "done",
					}
					dbStatus := dbStatusMap[status]
					_, err := db.Exec(`UPDATE tasks SET status = $1 WHERE id = $2`, dbStatus, id)

					if err != nil {
						return nil, err
					}

					return getTaskByID(id)
				},
			},
		},
	})

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query:    rootQuery,
		Mutation: rootMutation,
	})
	if err != nil {
		log.Fatal(err)
	}

	h := handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})

	http.Handle("/graphql", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		h.ServeHTTP(w, r)
	}))

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
