package routines

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestScheduleAndWait(t *testing.T) {
	var workDone [500]int32

	pool := NewPool(5)

	for i := range workDone {
		iPtr := &workDone[i]
		pool.Queue(func() {
			atomic.StoreInt32(iPtr, 1)
		})
	}

	pool.Wait()

	for i := range workDone {
		assert.Equal(t, int32(1), atomic.LoadInt32(&workDone[i]), "work %d not done", i)
	}
}

func TestQueuePanicsAfterWait(t *testing.T) {
	pool := NewPool(1)
	pool.Wait()

	assert.Panics(t, func() {
		pool.Queue(func() {})
	})
}

func TestExecutionOrder(t *testing.T) {
	var executedorder []int
	var mu sync.Mutex

	pool := NewPool(1)
	workCnt := 10

	for i := 0; i < workCnt; i++ {
		iCopy := i
		pool.Queue(func() {
			// sleep a bit to make it very likely that all work was
			// queued before it started to execute
			time.Sleep(10 * time.Millisecond)
			mu.Lock()
			defer mu.Unlock()
			executedorder = append(executedorder, iCopy)
		})
	}

	pool.Wait()

	for i := range executedorder {
		assert.Equal(t, i, executedorder[i])
	}
}
