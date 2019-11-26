package membershipchainservice

import "time"

type Phase int

// Const needed
const (
	REGISTRATION = iota
	SHARE
	EPOCH
	REGISTRATION_DUR = 1 * time.Second
	SHARE_DUR        = 1 * time.Second
	EPOCH_DUR        = 10 * time.Second
)

// Cycle describes a sequence of repeating phases starting at a given start time
type Cycle struct {
	Sequence  []time.Duration
	StartTime time.Time
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
	} else if rest < REGISTRATION_DUR+SHARE_DUR {
		return SHARE
	} else {
		return EPOCH
	}
}

// GetEpoch will give the current epoch based on the clock cycle
func (c Cycle) GetEpoch() Epoch {
	return Epoch(time.Now().Sub(c.StartTime) / c.TotalCycleTime())
}