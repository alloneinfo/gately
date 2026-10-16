package docker

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/wangwanjie/gately/internal/httpx"
)

// Task 是后台长任务（拉取镜像、构建、compose up 等）。接口立即返回任务 ID，界面轮询进度，
// 这样移动端切到后台或网络抖动后重新打开页面仍能看到结果。
type Task struct {
	ID       string    `json:"id"`
	Kind     string    `json:"kind"`   // pull / push / build / compose / upgrade / volume 等
	Title    string    `json:"title"`  // 展示用标题
	Status   string    `json:"status"` // running / done / error
	Error    string    `json:"error,omitempty"`
	Lines    []string  `json:"lines"` // 最近的进度行（最多 maxTaskLines 行）
	Started  time.Time `json:"started"`
	Finished time.Time `json:"finished,omitempty"`
}

const (
	maxTaskLines = 300
	maxTasks     = 50
)

type taskStore struct {
	mu    sync.Mutex
	seq   int
	tasks map[string]*Task
}

var tasks = &taskStore{tasks: map[string]*Task{}}

// startTask 在后台执行 fn 并返回任务 ID。fn 通过 progress 追加进度行；返回错误时任务状态为 error。
func (m *Module) startTask(kind, title string, fn func(ctx context.Context, progress func(line string)) error) string {
	tasks.mu.Lock()
	tasks.seq++
	id := strconv.FormatInt(time.Now().Unix(), 36) + "-" + strconv.Itoa(tasks.seq)
	t := &Task{ID: id, Kind: kind, Title: title, Status: "running", Lines: []string{}, Started: time.Now()}
	tasks.tasks[id] = t
	tasks.prune()
	tasks.mu.Unlock()

	progress := func(line string) {
		tasks.mu.Lock()
		defer tasks.mu.Unlock()
		t.Lines = append(t.Lines, line)
		if len(t.Lines) > maxTaskLines {
			t.Lines = t.Lines[len(t.Lines)-maxTaskLines:]
		}
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
		defer cancel()
		err := fn(ctx, progress)
		tasks.mu.Lock()
		defer tasks.mu.Unlock()
		t.Finished = time.Now()
		if err != nil {
			t.Status, t.Error = "error", err.Error()
			return
		}
		t.Status = "done"
	}()
	return id
}

// prune 只保留最近 maxTasks 个任务，调用方需持有锁。
func (s *taskStore) prune() {
	if len(s.tasks) <= maxTasks {
		return
	}
	list := make([]*Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		list = append(list, t)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Started.After(list[j].Started) })
	for _, t := range list[maxTasks:] {
		if t.Status != "running" {
			delete(s.tasks, t.ID)
		}
	}
}

func (s *taskStore) snapshot(id string) (Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return Task{}, false
	}
	c := *t
	c.Lines = append([]string{}, t.Lines...)
	return c, true
}

// registerTaskRoutes 注册任务查询接口。
func (m *Module) registerTaskRoutes(add func(pattern string, h http.HandlerFunc)) {
	add("GET /tasks", func(w http.ResponseWriter, r *http.Request) {
		tasks.mu.Lock()
		out := make([]Task, 0, len(tasks.tasks))
		for _, t := range tasks.tasks {
			c := *t
			c.Lines = nil // 列表不带进度行，详情接口再取
			out = append(out, c)
		}
		tasks.mu.Unlock()
		sort.Slice(out, func(i, j int) bool { return out[i].Started.After(out[j].Started) })
		httpx.OK(w, out)
	})
	add("GET /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		t, ok := tasks.snapshot(r.PathValue("id"))
		if !ok {
			httpx.Error(w, http.StatusNotFound, "任务不存在")
			return
		}
		httpx.OK(w, t)
	})
	add("DELETE /tasks", func(w http.ResponseWriter, r *http.Request) {
		tasks.mu.Lock()
		for id, t := range tasks.tasks {
			if t.Status != "running" {
				delete(tasks.tasks, id)
			}
		}
		tasks.mu.Unlock()
		httpx.OK(w, map[string]bool{"ok": true})
	})
}
