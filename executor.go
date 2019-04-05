package executor

import (
	"time"
)

type Job struct {
	Key  string
	Data interface{}
}

type Handler func(Job)

// Executor executes job in parallel
type Executor struct {
	workers        []*Worker
	maxWorkers     uint
	maxJobsInQueue uint // per worker
	handler        Handler
	jobCount       uint
}

// maxJobsInQueue > 1
func NewExecutor(maxWorkers, maxJobsInQueue uint, handler Handler) *Executor {
	if maxJobsInQueue < 2 {
		panic("maxJobsInQueue must greater than 1")
	}

	e := &Executor{
		workers:        make([]*Worker, 0, maxWorkers),
		maxWorkers:     maxWorkers,
		maxJobsInQueue: maxJobsInQueue,
		handler:        handler,
	}

	// creates and runs workers
	for i := uint(0); i < e.maxWorkers; i++ {
		worker := NewWorker(i, e.maxJobsInQueue, e.handler)
		e.workers = append(e.workers, worker)
		go worker.start()
	}

	return e
}

func New(nworkers, maxJobsInQueue uint, f func(string, interface{})) *Executor {
	return NewExecutor(nworkers, maxJobsInQueue, func(j Job) { f(j.Key, j.Data) })
}

// AddJob adds new job
// block if one of the queue is full
func (e *Executor) AddJob(job Job) {
	e.jobCount++
	worker := e.workers[getWorkerID(job.Key, e.maxWorkers)]
	worker.jobChannel <- job
}

func (e *Executor) Add(key string, data interface{}) {
	e.AddJob(Job{Key: key, Data: data})
}

func (e *Executor) Stop() {
	for _, worker := range e.workers {
		worker.stop()
	}
}

func (e *Executor) Info() map[int]uint {
	info := map[int]uint{}

	for i, w := range e.workers {
		info[i] = w.jobCount
	}

	return info
}

// Wait wait until all jobs is done
func (e *Executor) Wait() {
	for {
		njob, ndone := e.Count()
		if ndone == njob {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// Count return job count, done count
func (e *Executor) Count() (uint, uint) {
	var doneCount uint
	for _, w := range e.workers {
		doneCount += w.doneCount
	}
	return e.jobCount, doneCount
}
