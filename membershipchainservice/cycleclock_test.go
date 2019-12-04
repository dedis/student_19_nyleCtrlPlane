package membershipchainservice

import (
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
	nbCycle := time.Duration(2)
	var c Cycle
	c.Sequence = []time.Duration{REGISTRATION_DUR, EPOCH_DUR}
	c.StartTime = time.Now()

	done := make(chan bool)

	log.LLvl1("Set Registration Tick")
	time.Sleep(c.GetTimeTillNextCycle())
	c.RegistrationTick = time.NewTicker(c.TotalCycleTime())
	log.LLvl1("Set Epoch Tick")
	time.Sleep(c.GetTimeTillNextEpoch())
	c.EpochTick = time.NewTicker(c.TotalCycleTime())

	go func(test *testing.T, cy Cycle) {
		for {
			select {
			case tt := <-cy.EpochTick.C:
				log.LLvl1("Epoch Tick", tt)
				assert.Equal(test, Phase(EPOCH), c.GetCurrentPhase())
			case tt := <-cy.RegistrationTick.C:
				log.LLvl1("Registration Tick", tt)
				assert.Equal(test, Phase(REGISTRATION), c.GetCurrentPhase())
			case <-done:
				return
			}
		}
	}(t, c)

	time.Sleep(nbCycle * c.TotalCycleTime())
	c.StopTicking()
	done <- true
}
