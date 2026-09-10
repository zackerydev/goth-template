package handler

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zackerydev/goth-template/templates"
)

// Lab is a shared, in-memory hypermedia learning sandbox.
type Lab struct {
	mu       sync.Mutex
	renderer *templates.Renderer
	tasks    []Task
	events   []string
	started  time.Time
}

// Task is a work item represented by server-rendered HTML.
type Task struct {
	ID                                       int
	Title, Category, Status, Priority, Notes string
}

// Lesson describes an interaction implemented in the sandbox.
type Lesson struct {
	ID, Name, Summary, Markup, Response, Try string
}

type labPage struct {
	Title, View, Query, Status, Fragment, Notice, Error string
	Tasks                                               []Task
	Columns                                             []taskColumn
	Selected                                            *Task
	Lessons                                             []Lesson
	Lesson                                              Lesson
	Events                                              []string
	Next                                                int
	Total, Done, Active, Progress                       int
	Running                                             bool
}

type taskColumn struct {
	Name  string
	Tasks []Task
}

// NewLab creates a sandbox. State is intentionally shared and resets on restart.
func NewLab(renderer *templates.Renderer) http.Handler {
	lab := &Lab{renderer: renderer, tasks: seedTasks(), events: []string{"Workspace initialized · nine ideas, one HTML document."}}
	return http.NewCrossOriginProtection().Handler(lab)
}

func (lab *Lab) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	lab.mu.Lock()
	defer lab.mu.Unlock()
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "HX-Request, HX-Request-Type, HX-History-Restore-Request")
	if r.Method == http.MethodPost {
		lab.mutate(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data := lab.page(r)
	switch r.URL.Path {
	case "/lab":
		if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-History-Restore-Request") != "true" && r.Header.Get("HX-Request-Type") != "full" {
			data.Fragment = "workspace"
		}
	case "/lab/pulse":
		data.Fragment = "pulse"
	case "/lab/events":
		data.Fragment = "events"
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		data.Events, data.Next = eventPage(lab.events, offset)
	default:
		http.NotFound(w, r)
		return
	}
	lab.render(w, data, http.StatusOK)
}

func (lab *Lab) page(r *http.Request) labPage {
	query := r.URL.Query()
	data := labPage{Title: "Hypermedia Lab — HTML has range.", View: query.Get("view"), Query: query.Get("q"), Status: query.Get("status"), Total: len(lab.tasks), Lessons: lessons()}
	if data.View != "list" && data.View != "patterns" && data.View != "activity" {
		data.View = "board"
	}
	data.Lesson = data.Lessons[0]
	for _, lesson := range data.Lessons {
		if lesson.ID == query.Get("lesson") {
			data.Lesson = lesson
		}
	}
	id, _ := strconv.Atoi(query.Get("id"))
	if id == -1 {
		data.Selected = &Task{ID: -1, Status: "Backlog", Priority: "Medium"}
	}
	for _, task := range lab.tasks {
		if task.Status == "Done" {
			data.Done++
		}
		if task.Status == "In progress" {
			data.Active++
		}
		if task.ID == id {
			selected := task
			data.Selected = &selected
		}
		if (data.Status == "" || data.Status == task.Status) && strings.Contains(strings.ToLower(task.Title+" "+task.Category), strings.ToLower(data.Query)) {
			data.Tasks = append(data.Tasks, task)
		}
	}
	for _, status := range []string{"Backlog", "In progress", "Done"} {
		column := taskColumn{Name: status}
		for _, task := range data.Tasks {
			if task.Status == status {
				column.Tasks = append(column.Tasks, task)
			}
		}
		data.Columns = append(data.Columns, column)
	}
	data.Events, data.Next = eventPage(lab.events, 0)
	if !lab.started.IsZero() {
		data.Progress = min(100, int(time.Since(lab.started).Seconds()*10))
		data.Running = data.Progress < 100
	}
	return data
}

func (lab *Lab) render(w http.ResponseWriter, data labPage, status int) {
	var content bytes.Buffer
	if err := lab.renderer.Render(&content, "lab", data); err != nil {
		http.Error(w, "Unable to render the lab", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if _, err := w.Write(content.Bytes()); err != nil {
		return
	}
}

func eventPage(events []string, offset int) ([]string, int) {
	offset = max(0, min(offset, len(events)))
	end := min(offset+5, len(events))
	next := 0
	if end < len(events) {
		next = end
	}
	return events[offset:end], next
}
