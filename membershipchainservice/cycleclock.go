package membershipchainservice

import (
	"time"
)

type Phase int

// const needed
var (
	REGISTRATION       = Phase(0)
	EPOCH              = Phase(1)
	REGISTRATION_DUR   = 4 * time.Second
	EPOCH_DUR          = 5 * time.Second
	TIME_FOR_CONSENCUS = 2 * time.Second
	MARGIN_FOR_CYCLE   = 1 * time.Second
)

// Cycle describes a sequence of repeating phases starting at a given start time
type Cycle struct {
	Sequence   []time.Duration
	StartTime  time.Time
	ShiftEpoch Epoch
}

// TotalCycleTime return the total time of a Cycle
func (c *Cycle) TotalCycleTime() time.Duration {
	total := time.Duration(0)
	for _, s := range c.Sequence {
		total += s
	}
	return total
}

// GetCurrentPhase will give the current phase
func (c *Cycle) GetCurrentPhase() Phase {
	now := time.Now()
	rest := now.Sub(c.StartTime) % c.TotalCycleTime()
	if rest < REGISTRATION_DUR {
		return REGISTRATION
	} else {
		return EPOCH
	}
}

// GetTimeTillNextCycle will gives the time till the next cycle
func (c *Cycle) GetTimeTillNextCycle() time.Duration {
	return c.TotalCycleTime() - (time.Now().Sub(c.StartTime) % c.TotalCycleTime())
}

// GetTimeTillNextEpoch will gives the time till the next epoch
func (c *Cycle) GetTimeTillNextEpoch() time.Duration {
	if c.GetCurrentPhase() == EPOCH {
		return time.Duration(0)
	}
	return c.GetTimeTillNextCycle() - EPOCH_DUR
}

// GetEpoch will give the current epoch based on the clock cycle
func (c *Cycle) GetEpoch() Epoch {
	return Epoch(time.Now().Sub(c.StartTime)/c.TotalCycleTime()) + c.ShiftEpoch
}

// Set can be used to change the lenght of a cycle during execution
// Should be executed before registration + just before changing the duration
func (c *Cycle) Set() {
	c.Sequence = []time.Duration{REGISTRATION_DUR, EPOCH_DUR}
	c.StartTime = time.Now()
}

// CheckPoint can be use to store the number of epoch before setting the clock again
func (c *Cycle) CheckPoint() {
	if c.GetCurrentPhase() == REGISTRATION {
		c.ShiftEpoch = c.GetEpoch()
	}
	if c.GetCurrentPhase() == EPOCH {
		c.ShiftEpoch = c.GetEpoch() + 1
	}
}
