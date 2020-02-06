package membershipchainservice

import (
	"time"

	"go.dedis.ch/onet/v3/network"
)

// SetGenesisSignersRequest is used to send a request from api to a service
type SetGenesisSignersRequest struct {
	Servers map[*network.ServerIdentity]string
}

//SetGenesisSignersReply is the reply from the service to the api
type SetGenesisSignersReply struct{}

// ExecEpochRequest is used to send a request from api to a service
type ExecEpochRequest struct {
	Epoch Epoch
}

//ExecEpochReply is the reply from the service to the api
type ExecEpochReply struct{}

// ExecWriteSigners is used to send a request from api to a service
type ExecWriteSigners struct {
	Epoch Epoch
}

//ExecWriteSignersReply is the reply from the service to the api
type ExecWriteSignersReply struct{}

//SetDurationRequest is used to force a client to set the registration duration
type SetDurationRequest struct {
	Duration time.Duration
}

// SetDurationReply is the reply
type SetDurationReply struct {
}

//UpdateForNewNodeRequest is used to update a new node
type UpdateForNewNodeRequest struct {
	Epoch Epoch
}

// UpdateForNewNodeReply is the reply
type UpdateForNewNodeReply struct{}

//UpdateRequest is used to update a new node
type UpdateRequest struct{}

// UpdateReply is the reply
type UpdateReply struct{}

//CreateProofForEpochRequest is used to send a request to Create Proof
type CreateProofForEpochRequest struct {
	Epoch Epoch
}

// CreateProofForEpochReply is the reply
type CreateProofForEpochReply struct{}

//StartNewEpochRequest is used to send a request to Start Epoch
type StartNewEpochRequest struct{}

// StartNewEpochReply is the reply
type StartNewEpochReply struct{}

//GetConsencusOnNewSignersRequest is used to send a request to get consencus
type GetConsencusOnNewSignersRequest struct{}

// GetConsencusOnNewSignersReply is the reply
type GetConsencusOnNewSignersReply struct{}

//SendPauseRequest is used to Pause a service
type SendPauseRequest struct{}

// SendPauseReply is the reply
type SendPauseReply struct{}
