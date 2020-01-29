package membershipchainservice

import "go.dedis.ch/onet/v3/network"

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
