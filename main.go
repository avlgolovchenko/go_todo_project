package main

import (
	"log"
	"net/http"
	"os"

	"go_todo_project/internal/database"
	"go_todo_project/internal/handlers"
)

func main() {
	port := os.Getenv("TODO_PORT")
	if port == "" {
		port = "7540"
	}

	dbPath := os.Getenv("TODO_DBFILE")
	if dbPath == "" {
		dbPath = "./scheduler.db"
	}

	db, err := database.ConnectDB(dbPath)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	err = database.RunMigrations(db)
	if err != nil {
		log.Fatalf("Ошибка выполнения миграций: %v", err)
	}

	http.Handle("/", http.FileServer(http.Dir("./web")))

	http.HandleFunc("/api/signin", handlers.HandleSignIn)
	http.HandleFunc("/api/nextdate", handlers.HandleNextDate)

	http.HandleFunc("/api/task", handlers.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.HandleGetTask(w, r, db)
		case http.MethodPut:
			handlers.HandleUpdateTask(w, r, db)
		case http.MethodPost:
			handlers.HandleAddTask(w, r, db)
		case http.MethodDelete:
			handlers.HandleDeleteTask(w, r, db)
		default:
			http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
		}
	}))

	http.HandleFunc("/api/tasks", handlers.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleTaskList(w, r, db)
	}))

	http.HandleFunc("/api/task/done", handlers.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handlers.HandleDoneTask(w, r, db)
		} else {
			http.Error(w, `{"error":"Метод не поддерживается"}`, http.StatusMethodNotAllowed)
		}
	}))

	log.Printf("Сервер запущен на порту %s", port)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
