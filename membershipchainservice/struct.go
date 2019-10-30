package membershipchainservice

import (
	"go.dedis.ch/onet"
	"go.dedis.ch/onet/network"
)

// We need to register all messages so the network knows how to handle them.
func init() {
	network.RegisterMessages(
		GossipArgs{}, GossipReply{}, RegistrationsListReply{},
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

// RegistrationsListReply is used to communicate the registrations that are stored on one node
type RegistrationsListReply struct {
	List map[string]bool
}
