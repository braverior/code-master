package codegen

import (
	"sync"
)

type TaskFunc func()

type Pool struct {
	maxWorkers int
	taskQueue  chan TaskFunc
	wg         sync.WaitGroup
	quit       chan struct{}
}

func NewPool(maxWorkers int) *Pool {
	p := &Pool{
		maxWorkers: maxWorkers,
		taskQueue:  make(chan TaskFunc, 100),
		quit:       make(chan struct{}),
	}
	p.start()
	return p
}

func (p *Pool) start() {
	for i := 0; i < p.maxWorkers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case task, ok := <-p.taskQueue:
					if !ok {
						return
					}
					task()
				case <-p.quit:
					return
				}
			}
		}()
	}
}

func (p *Pool) Submit(task TaskFunc) int {
	p.taskQueue <- task
	return len(p.taskQueue)
}

func (p *Pool) QueueSize() int {
	return len(p.taskQueue)
}

func (p *Pool) Shutdown() {
	close(p.quit)
	p.wg.Wait()
}
