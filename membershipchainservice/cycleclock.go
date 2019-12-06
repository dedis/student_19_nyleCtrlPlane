package membershipchainservice

import (
	"time"

	"go.dedis.ch/onet/v3/log"
)

type Phase int

// Const needed
const (
	REGISTRATION = iota
	EPOCH
	REGISTRATION_DUR = 2 * time.Second
	EPOCH_DUR        = 2 * time.Second
)

// Cycle describes a sequence of repeating phases starting at a given start time
type Cycle struct {
	Sequence         []time.Duration
	StartTime        time.Time
	RegistrationTick *time.Ticker
	EpochTick        *time.Ticker
	Done             chan bool
}

// TotalCycleTime return the total time of a Cycle
func (c Cycle) TotalCycleTime() time.Duration {
	total := time.Duration(0)
	for _, s := range c.Sequence {
		total += s
	}
	return total
}

// GetCurrentPhase will give the current phase
func (c Cycle) GetCurrentPhase() Phase {
	now := time.Now()
	rest := now.Sub(c.StartTime) % c.TotalCycleTime()
	if rest < REGISTRATION_DUR {
		return REGISTRATION
	} else {
		return EPOCH
	}
}

// GetTimeTillNextCycle will gives the time till the next cycle
func (c Cycle) GetTimeTillNextCycle() time.Duration {
	return c.TotalCycleTime() - (time.Now().Sub(c.StartTime) % c.TotalCycleTime())
}

// GetTimeTillNextEpoch will gives the time till the next epoch
func (c Cycle) GetTimeTillNextEpoch() time.Duration {
	if c.GetCurrentPhase() == EPOCH {
		return time.Duration(0)
	}
	return c.GetTimeTillNextCycle() - EPOCH_DUR
}

// GetEpoch will give the current epoch based on the clock cycle
func (c Cycle) GetEpoch() Epoch {
	return Epoch(time.Now().Sub(c.StartTime) / c.TotalCycleTime())
}

// StartTicking instanciate the tickers
func (c Cycle) StartTicking(registrationFn, epochFn, callback func()) {
	c.Done = make(chan bool)
	log.LLvl1("Set Registration Tick")
	c.StartTime = time.Now()
	c.RegistrationTick = time.NewTicker(c.TotalCycleTime())
	time.Sleep(c.GetTimeTillNextEpoch())
	log.LLvl1("Set Epoch Tick")
	c.EpochTick = time.NewTicker(c.TotalCycleTime())

	go func() {
		for {
			select {
			case <-c.RegistrationTick.C:
				registrationFn()
			case <-c.EpochTick.C:
				epochFn()
			case <-c.Done:
				log.LLvl1("Ticking done.")
				return
			}
		}
	}()

	callback()
}

// StopTicking stops the tickers
func (c Cycle) StopTicking() {
	log.LLvl1(c)
	if c.RegistrationTick != nil {
		c.RegistrationTick.Stop()
	}
	if c.EpochTick != nil {
		c.EpochTick.Stop()
	}
	log.LLvl1("Stop Ticking")
	c.Done <- true
	close(c.Done)

}
