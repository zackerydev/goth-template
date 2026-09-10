package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (lab *Lab) mutate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid or oversized form", http.StatusBadRequest)
		return
	}
	notice := ""
	switch r.URL.Path {
	case "/lab/task":
		lab.saveTask(w, r)
		return
	case "/lab/bulk":
		selected := r.PostForm["task"]
		if len(selected) == 0 {
			notice = "Select at least one task first."
		}
		for i := range lab.tasks {
			for _, id := range selected {
				if strconv.Itoa(lab.tasks[i].ID) == id {
					lab.tasks[i].Status = "Done"
					notice = "Selected tasks completed."
				}
			}
		}
	case "/lab/build":
		lab.started = time.Now()
		notice = "Preview build started. The server owns the clock."
	case "/lab/reset":
		lab.tasks = seedTasks()
		lab.events = nil
		lab.started = time.Time{}
		notice = "Sandbox reset. Make something happen."
	case "/lab/remove":
		id, _ := strconv.Atoi(r.URL.Query().Get("id"))
		for i, task := range lab.tasks {
			if task.ID == id {
				lab.tasks = append(lab.tasks[:i], lab.tasks[i+1:]...)
				notice = "Task deleted."
				break
			}
		}
		if notice == "" {
			http.NotFound(w, r)
			return
		}
	default:
		http.NotFound(w, r)
		return
	}
	lab.finish(w, r, notice)
}

func (lab *Lab) saveTask(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))
	task := Task{ID: id, Title: strings.TrimSpace(r.PostForm.Get("title")), Category: "Studio", Status: r.PostForm.Get("status"), Priority: r.PostForm.Get("priority"), Notes: strings.TrimSpace(r.PostForm.Get("notes"))}
	index := -1
	for i, existing := range lab.tasks {
		if existing.ID == id {
			index = i
			task.Category = existing.Category
		}
	}
	if id != -1 && index == -1 {
		http.NotFound(w, r)
		return
	}
	message := validateTask(task)
	if message != "" {
		data := lab.page(r)
		data.Selected, data.Error = &task, message
		if r.Header.Get("HX-Request") == "true" {
			data.Fragment = "editor"
		}
		lab.render(w, data, http.StatusUnprocessableEntity)
		return
	}
	if id == -1 {
		task.ID = 1
		for _, existing := range lab.tasks {
			task.ID = max(task.ID, existing.ID+1)
		}
		lab.tasks = append(lab.tasks, task)
	} else {
		lab.tasks[index] = task
	}
	lab.finish(w, r, "Saved: "+task.Title)
}

func validateTask(task Task) string {
	if len(task.Title) < 3 || len(task.Title) > 100 {
		return "Give this task a title between 3 and 100 characters."
	}
	if task.Status != "Backlog" && task.Status != "In progress" && task.Status != "Done" {
		return "Choose a valid workflow status."
	}
	if task.Priority != "Low" && task.Priority != "Medium" && task.Priority != "High" {
		return "Choose a valid priority."
	}
	if len(task.Notes) > 2000 {
		return "Keep notes under 2,000 characters."
	}
	return ""
}

func (lab *Lab) finish(w http.ResponseWriter, r *http.Request, notice string) {
	if notice != "" {
		lab.events = append([]string{time.Now().Format("15:04:05") + " · " + notice}, lab.events...)
		lab.events = lab.events[:min(len(lab.events), 100)]
	}
	query := r.URL.Query()
	query.Del("id")
	r.URL.RawQuery = query.Encode()
	location := "/lab?" + r.URL.RawQuery
	if r.Header.Get("HX-Request") != "true" {
		http.Redirect(w, r, location, http.StatusSeeOther) //nolint:gosec // Fixed /lab path; user input is confined to a URL-encoded query.
		return
	}
	data := lab.page(r)
	data.Fragment, data.Notice = "workspace", notice
	w.Header().Set("HX-Replace-Url", location)
	lab.render(w, data, http.StatusOK)
}
