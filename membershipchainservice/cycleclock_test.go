package membershipchainservice

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.dedis.ch/onet/v3/log"
)

func TestTotalCycleTime(t *testing.T) {
	var c Cycle
	c.Sequence = []time.Duration{4 * time.Second, 6 * time.Second}

	assert.Equal(t, 10*time.Second, c.TotalCycleTime())
}

func TestStartTicking(t *testing.T) {
	nbCycle := time.Duration(10)
	var c Cycle
	c.Sequence = []time.Duration{REGISTRATION_DUR, EPOCH_DUR}
	c.StartTime = time.Now()

	var wg sync.WaitGroup
	wg.Add(1)
	c.StartTicking(
		func() { log.LLvl1("Registrate") },
		func() { log.LLvl1("Epoch") },
		func() {
			time.Sleep(nbCycle * time.Second)
			c.StopTicking()
			wg.Done()
		})
	wg.Wait()

	time.Sleep(time.Second)

}
