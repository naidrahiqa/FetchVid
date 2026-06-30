package engine

import (
	"fmt"
	"sync"
	"time"
)

type Job struct {
	Index   int
	URL     string
	Title   string
	Source  string
	Success bool
	Error   string
	Size    string
}

type Queue struct {
	mu       sync.Mutex
	Jobs     []Job
	Pending  []int // indices of pending jobs
	running  bool
	paused   bool
	stopped  bool
	Progress QueueProgress
	OnProgress func(QueueProgress)
	OnJobDone   func(Job)
	OnComplete  func(QueueProgress)
}

type QueueProgress struct {
	Total     int     `json:"total"`
	Completed int     `json:"completed"`
	Success   int     `json:"success"`
	Failed    int     `json:"failed"`
	Percent   float64 `json:"percent"`
	Running   bool    `json:"running"`
	Paused    bool    `json:"paused"`
	Elapsed   string  `json:"elapsed"`
	StartTime time.Time `json:"-"`
}

func NewQueue() *Queue {
	return &Queue{}
}

func (q *Queue) Add(jobs []Job) {
	q.mu.Lock()
	defer q.mu.Unlock()

	start := len(q.Jobs)
	for i, j := range jobs {
		j.Index = start + i + 1
		q.Jobs = append(q.Jobs, j)
		q.Pending = append(q.Pending, start+i)
	}
}

func (q *Queue) Start(concurrent int, outputDir, cookies string) {
	q.mu.Lock()
	if q.running {
		q.mu.Unlock()
		return
	}
	q.running = true
	q.stopped = false
	q.paused = false
	q.Progress = QueueProgress{
		Total:     len(q.Pending),
		StartTime: time.Now(),
	}
	q.mu.Unlock()

	// Worker pool
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrent)

	// Fill semaphore for concurrent control
	for i := 0; i < concurrent; i++ {
		sem <- struct{}{}
	}

	go func() {
		for {
			q.mu.Lock()
			if q.stopped || len(q.Pending) == 0 {
				q.mu.Unlock()
				break
			}
			if q.paused {
				q.mu.Unlock()
				time.Sleep(200 * time.Millisecond)
				continue
			}

			idx := q.Pending[0]
			q.Pending = q.Pending[1:]
			q.mu.Unlock()

			<-sem // Wait for available slot

			wg.Add(1)
			go func(jobIdx int) {
				defer wg.Done()
				defer func() { sem <- struct{}{} }()

				q.mu.Lock()
				job := &q.Jobs[jobIdx]
				q.mu.Unlock()

				yt := &Ytdlp{
					Cookies:   cookies,
					OutputDir: outputDir,
				}
				yt.Path = findInPath()
				if yt.Path == "" {
					yt.Path = findInAppData()
				}

				err := yt.DownloadVideo(job.URL, job.Index, q.Progress.Total, nil)
				q.mu.Lock()
				if err != nil {
					job.Error = err.Error()
					q.Progress.Failed++
				} else {
					job.Success = true
					q.Progress.Success++
				}
				q.Progress.Completed++
				elapsed := time.Since(q.Progress.StartTime)
				q.Progress.Elapsed = formatDuration(elapsed)
				q.Progress.Percent = float64(q.Progress.Completed) / float64(q.Progress.Total) * 100
				prog := q.Progress
				j := *job
				q.mu.Unlock()

				if q.OnJobDone != nil {
					q.OnJobDone(j)
				}
				if q.OnProgress != nil {
					q.OnProgress(prog)
				}
			}(idx)
		}

		wg.Wait()
		q.mu.Lock()
		q.running = false
		prog := q.Progress
		q.mu.Unlock()

		if q.OnComplete != nil {
			q.OnComplete(prog)
		}
	}()
}

func (q *Queue) Pause() {
	q.mu.Lock()
	q.paused = true
	q.Progress.Paused = true
	q.mu.Unlock()
}

func (q *Queue) Resume() {
	q.mu.Lock()
	q.paused = false
	q.Progress.Paused = false
	q.mu.Unlock()
}

func (q *Queue) Stop() {
	q.mu.Lock()
	q.stopped = true
	q.running = false
	q.mu.Unlock()
}

func (q *Queue) IsRunning() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.running
}

func formatDuration(d time.Duration) string {
	if d.Hours() >= 1 {
		return fmt.Sprintf("%.1f jam", d.Hours())
	}
	if d.Minutes() >= 1 {
		return fmt.Sprintf("%.0f menit", d.Minutes())
	}
	return fmt.Sprintf("%.0f detik", d.Seconds())
}
