package membershipchainservice

import (
	"go.dedis.ch/onet"
	"go.dedis.ch/onet/network"
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
	Set map[string]bool
}
