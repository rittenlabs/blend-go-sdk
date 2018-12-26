package async

import "runtime"

// NewBatch creates a new batch processor.
func NewBatch(action func(interface{}) error, workItems ...interface{}) *Batch {
	return &Batch{
		latch:      &Latch{},
		workItems:  workItems,
		action:     action,
		numWorkers: runtime.NumCPU(),
	}
}

// Batch is a batch of work executed by a fixed count of workers.
type Batch struct {
	latch      *Latch
	numWorkers int
	action     func(interface{}) error
	workItems  []interface{}
	errors     chan error
	workers    chan *Worker
}

// WithNumWorkers sets the number of workers.
// It defaults to `runtime.NumCPU()`
func (b *Batch) WithNumWorkers(numWorkers int) *Batch {
	b.numWorkers = numWorkers
	return b
}

// NumWorkers returns the number of worker route
func (b *Batch) NumWorkers() int {
	return b.numWorkers
}

// Latch returns the worker latch.
func (b *Batch) Latch() *Latch {
	return b.latch
}

// WithErrors sets the error channel.
func (b *Batch) WithErrors(errors chan error) *Batch {
	b.errors = errors
	return b
}

// Errors returns a channel to read action errors from.
func (b *Batch) Errors() chan error {
	return b.errors
}

// Process exeuctes the action for all the work items.
func (b *Batch) Process() {
	// initialize the workers
	b.workers = make(chan *Worker, b.numWorkers)
	for x := 0; x < b.numWorkers; x++ {
		worker := &Worker{
			latch:  &Latch{},
			work:   make(chan interface{}),
			errors: b.errors,
		}
		worker.action = b.andReturn(worker, b.action)
		worker.Start()
		b.workers <- worker
	}

	numWorkItems := len(b.workItems)
	var worker *Worker
	var workItem interface{}
	for x := 0; x < numWorkItems; x++ {
		workItem = b.workItems[x]
		select {
		case worker = <-b.workers:
			worker.Enqueue(workItem)
		case <-b.latch.NotifyStopping():
			b.latch.Stopped()
			return
		}
	}

	for x := 0; x < b.numWorkers; x++ {
		worker := <-b.workers
		worker.Stop()
	}
}

// AndReturn creates an action handler that returns a given worker to the worker queue.
// It wraps any action provided to the queue.
func (b *Batch) andReturn(worker *Worker, action QueueAction) QueueAction {
	return func(workItem interface{}) error {
		defer func() {
			b.workers <- worker
		}()
		return action(workItem)
	}
}