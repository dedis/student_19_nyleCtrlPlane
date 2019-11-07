package membershipchainservice

import (
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/network"
)

// We need to register all messages so the network knows how to handle them.
func init() {
	network.RegisterMessages(
		GossipArgs{}, GossipReply{}, SignersReply{},
	)
}

// GossipArgs will give the roster on which a the node will gossip
type GossipArgs struct {
	Roster *onet.Roster
}

// GossipReply give the status #TODO : change
type GossipReply struct {
	Status int
}

// SignersReply is used to communicate the registrations that are stored on one node
type SignersReply struct {
	Set SignersSet
}

// SignersSet describes the type used to store the signers on nodes
type SignersSet map[network.ServerIdentityID]bool

// Epoch corresponds for now only to the number of the Epoch
type Epoch int

// ServiceFn is used to pass the service to the registration protocol
type ServiceFn func() *Service
