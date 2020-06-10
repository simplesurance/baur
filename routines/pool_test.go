package routines

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScheduleAndWait(t *testing.T) {
	var workDone [500]int32

	pool := NewPool(5)

	for i := range workDone {
		pool.Queue(func(done interface{}) {
			workDonePtr := done.(*int32)

			atomic.StoreInt32(workDonePtr, 1)
		}, &workDone[i])
	}

	pool.Wait()

	for i := range workDone {
		assert.Equal(t, int32(1), atomic.LoadInt32(&workDone[i]), "work %d not done", i)
	}

}

func TestQueuePanicsAfterWait(t *testing.T) {
	pool := NewPool(1)
	pool.Wait()

	assert.Panics(t, func() { pool.Queue(func(_ interface{}) {}, nil) })
}
