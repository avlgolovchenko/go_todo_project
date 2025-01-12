package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go_todo_project/internal/rules"
)

type TaskRequest struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

type TaskResponse struct {
	ID    int64  `json:"id,omitempty"`    // ID созданной задачи
	Error string `json:"error,omitempty"` // Сообщение об ошибке, если есть
}

func HandleNextDate(w http.ResponseWriter, r *http.Request) {
	nowStr := r.FormValue("now")
	dateStr := r.FormValue("date")
	repeat := r.FormValue("repeat")

	now, err := time.Parse("20060102", nowStr)
	if err != nil {
		http.Error(w, "некорректная дата now", http.StatusBadRequest)
		return
	}

	nextDate, err := rules.NextDate(now, dateStr, repeat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, nextDate)
}

func HandleTaskList(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	search := r.URL.Query().Get("search")
	limit := 50

	var query string
	var args []interface{}

	if search != "" {
		if parsedDate, err := time.Parse("02.01.2006", search); err == nil {
			query = `SELECT * FROM scheduler WHERE date = ? ORDER BY date LIMIT ?`
			args = append(args, parsedDate.Format("20060102"), limit)
		} else {
			query = `SELECT * FROM scheduler WHERE title LIKE ? OR comment LIKE ? ORDER BY date LIMIT ?`
			searchPattern := "%" + search + "%"
			args = append(args, searchPattern, searchPattern, limit)
		}
	} else {
		query = `SELECT * FROM scheduler ORDER BY date LIMIT ?`
		args = append(args, limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, `{"error":"Ошибка чтения базы данных"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tasks := []map[string]string{}
	for rows.Next() {
		var id int
		var date, title, comment, repeat string
		if err := rows.Scan(&id, &date, &title, &comment, &repeat); err != nil {
			http.Error(w, `{"error":"Ошибка чтения строки базы данных"}`, http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, map[string]string{
			"id":      strconv.Itoa(id),
			"date":    date,
			"title":   title,
			"comment": comment,
			"repeat":  repeat,
		})
	}

	if tasks == nil {
		tasks = []map[string]string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"tasks": tasks})
}

func HandleAddTask(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Ошибка декодирования JSON"}`, http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		http.Error(w, `{"error":"Не указан заголовок задачи"}`, http.StatusBadRequest)
		return
	}

	now := time.Now()
	taskDate := now.Format("20060102")

	if req.Date != "" {
		parsedDate, err := time.Parse("20060102", req.Date)
		if err != nil {
			http.Error(w, `{"error":"Некорректный формат даты"}`, http.StatusBadRequest)
			return
		}

		normalizedNow := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		if parsedDate.Before(normalizedNow) && req.Repeat != "" {
			nextDate, err := rules.NextDate(normalizedNow, parsedDate.Format("20060102"), req.Repeat)
			if err != nil {
				http.Error(w, `{"error":"Некорректное правило повторения"}`, http.StatusBadRequest)
				return
			}
			taskDate = nextDate
		} else if parsedDate.After(normalizedNow) {
			taskDate = parsedDate.Format("20060102")
		} else {
			taskDate = normalizedNow.Format("20060102")
		}
	}

	if req.Repeat != "" {
		_, err := rules.NextDate(now, taskDate, req.Repeat)
		if err != nil {
			http.Error(w, `{"error":"Некорректное правило повторения"}`, http.StatusBadRequest)
			return
		}
	}

	query := `INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)`
	res, err := db.Exec(query, taskDate, req.Title, req.Comment, req.Repeat)
	if err != nil {
		http.Error(w, `{"error":"Ошибка добавления задачи в базу данных"}`, http.StatusInternalServerError)
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		http.Error(w, `{"error":"Ошибка получения ID задачи"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TaskResponse{ID: id})
}

func HandleGetTask(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, `{"error":"Не указан идентификатор задачи"}`, http.StatusBadRequest)
		return
	}

	var task TaskRequest
	query := `SELECT id, date, title, comment, repeat FROM scheduler WHERE id = ?`
	err := db.QueryRow(query, id).Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"Задача не найдена"}`, http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, `{"error":"Ошибка получения задачи"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func HandleUpdateTask(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req TaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Ошибка декодирования JSON"}`, http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, `{"error":"Не указан идентификатор задачи"}`, http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(req.ID)
	if err != nil || id <= 0 {
		http.Error(w, `{"error":"Некорректный идентификатор задачи"}`, http.StatusBadRequest)
		return
	}

	var taskDate string
	now := time.Now()

	if req.Date != "" {
		parsedDate, err := time.Parse("20060102", req.Date)
		if err != nil {
			http.Error(w, `{"error":"Некорректный формат даты"}`, http.StatusBadRequest)
			return
		}

		if parsedDate.Before(now) && req.Repeat != "" {
			nextDate, err := rules.NextDate(now, parsedDate.Format("20060102"), req.Repeat)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
				return
			}
			taskDate = nextDate
		} else {
			taskDate = parsedDate.Format("20060102")
		}
	} else {
		taskDate = now.Format("20060102")
	}

	if req.Repeat != "" {
		_, err := rules.NextDate(now, taskDate, req.Repeat)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
	}

	if req.Title == "" {
		http.Error(w, `{"error":"Заголовок задачи не может быть пустым"}`, http.StatusBadRequest)
		return
	}

	query := `UPDATE scheduler SET date = ?, title = ?, comment = ?, repeat = ? WHERE id = ?`
	result, err := db.Exec(query, taskDate, req.Title, req.Comment, req.Repeat, id)
	if err != nil {
		http.Error(w, `{"error":"Ошибка обновления задачи в базе данных"}`, http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		http.Error(w, `{"error":"Задача не найдена"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{}`))
}

func HandleDoneTask(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, `{"error":"Не указан идентификатор задачи"}`, http.StatusBadRequest)
		return
	}

	var repeatRule string
	query := `SELECT repeat FROM scheduler WHERE id = ?`
	err := db.QueryRow(query, id).Scan(&repeatRule)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"Задача не найдена"}`, http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, `{"error":"Ошибка получения задачи"}`, http.StatusInternalServerError)
		return
	}

	if repeatRule == "" {
		query = `DELETE FROM scheduler WHERE id = ?`
		_, err = db.Exec(query, id)
		if err != nil {
			http.Error(w, `{"error":"Ошибка удаления задачи"}`, http.StatusInternalServerError)
			return
		}
	} else {
		var currentDate string
		query = `SELECT date FROM scheduler WHERE id = ?`
		err = db.QueryRow(query, id).Scan(&currentDate)
		if err != nil {
			http.Error(w, `{"error":"Ошибка получения текущей даты задачи"}`, http.StatusInternalServerError)
			return
		}

		nextDate, err := rules.NextDate(time.Now(), currentDate, repeatRule)
		if err != nil {
			http.Error(w, `{"error":"Ошибка расчета следующей даты"}`, http.StatusBadRequest)
			return
		}

		query = `UPDATE scheduler SET date = ? WHERE id = ?`
		_, err = db.Exec(query, nextDate, id)
		if err != nil {
			http.Error(w, `{"error":"Ошибка обновления задачи"}`, http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{}`))
}

func HandleDeleteTask(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, `{"error":"Не указан идентификатор задачи"}`, http.StatusBadRequest)
		return
	}

	query := `DELETE FROM scheduler WHERE id = ?`
	result, err := db.Exec(query, id)
	if err != nil {
		http.Error(w, `{"error":"Ошибка удаления задачи"}`, http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		http.Error(w, `{"error":"Задача не найдена"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{}`))
}
