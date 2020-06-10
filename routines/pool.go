package routines

import (
	"sync"
)

// Pool is a FIFO go-routine pool.
type Pool struct {
	wq        []*work
	terminate bool
	wqMutex   sync.Mutex // protects wq and terminate

	workChan chan *work

	schedulerNotifyChan chan struct{}

	terminateWg sync.WaitGroup
}

type work struct {
	fn       WorkFn
	userData interface{}
}

// WorkFn is a function that is executed by the pool workers.
type WorkFn func(userData interface{})

// NewPool creates and start a new go-routine pool.
// The pool starts <routines> number of workers.
func NewPool(routines uint) *Pool {

	p := Pool{
		workChan:            make(chan *work, routines),
		schedulerNotifyChan: make(chan struct{}, 1),
	}

	p.terminateWg.Add(1)
	go p.scheduler()

	for i := uint(0); i < routines; i++ {
		p.terminateWg.Add(1)
		go p.worker()
	}

	return &p
}

func (p *Pool) scheduler() {
	defer p.terminateWg.Done()

	for {
		<-p.schedulerNotifyChan

		work := p.popWork()
		if work == nil {
			p.wqMutex.Lock()
			terminate := p.terminate
			p.wqMutex.Unlock()

			if terminate {
				close(p.workChan)

				return
			}

			continue
		}

		p.workChan <- work
	}
}

func (p *Pool) popWork() *work {
	p.wqMutex.Lock()
	defer p.wqMutex.Unlock()

	if len(p.wq) == 0 {
		return nil
	}

	w := p.wq[len(p.wq)-1]
	p.wq = p.wq[:len(p.wq)-1]

	return w
}

func (p *Pool) worker() {
	defer p.terminateWg.Done()

	for {
		w, open := <-p.workChan
		if !open {
			return
		}

		p.notifyScheduler()

		w.fn(w.userData)
	}
}

func (p *Pool) notifyScheduler() {
	select {
	case p.schedulerNotifyChan <- struct{}{}:

	default:
	}
}

// Queue queues new work for the pool.
// The userData parameter is passed as first parameter to workFn.
// If Queue() is called after Wait(), the method panics.
// The method never blocks.
func (p *Pool) Queue(workFn WorkFn, userData interface{}) {
	p.wqMutex.Lock()
	defer p.wqMutex.Unlock()

	if p.terminate {
		panic("work was queued on a closed pool")
	}

	p.wq = append(p.wq, &work{fn: workFn, userData: userData})
	p.notifyScheduler()
}

// Wait waits until the workqueue is empty and then terminates the worker
// goroutines.
// After Wait() was called, no further work must be queued.
func (p *Pool) Wait() {
	p.wqMutex.Lock()
	p.terminate = true
	p.wqMutex.Unlock()

	p.notifyScheduler()

	p.terminateWg.Wait()
	close(p.schedulerNotifyChan)
}
