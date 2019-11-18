package membershipchainservice

import (
	gpr "github.com/dedis/student_19_nyleCtrlPlane/gossipregistrationprotocol"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/network"
)

// We need to register all messages so the network knows how to handle them.
func init() {
	network.RegisterMessages(
		GossipArgs{}, GossipReply{}, SignersReply{},
	)
	network.RegisterMessage(&SignatureRequest{})
	network.RegisterMessage(&gpr.SignatureResponse{})
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
type SignersSet map[network.ServerIdentityID]gpr.SignatureResponse

// Epoch corresponds for now only to the number of the Epoch
type Epoch int

// ServiceFn is used to pass the service to the registration protocol
type ServiceFn func() *Service

// InitRequest is used to pass information to create the trees in CRUX protocol.
type InitRequest struct {
	ServerIdentityToName map[*network.ServerIdentity]string
}

//GraphTree represents The actual graph that will be linked to the Binary Tree of the Protocol
type GraphTree struct {
	Tree        *onet.Tree
	ListOfNodes []*onet.TreeNode
	Parents     map[*onet.TreeNode][]*onet.TreeNode
	Radius      float64
}

// ReqPings is use to request pings
type ReqPings struct {
	SenderName string
}

// ReplyPings hold the reply form the ping request
type ReplyPings struct {
	Pings      string
	SenderName string
}

// SignatureRequest is what the Cosi service is expected to receive from clients.
type SignatureRequest struct {
	Message []byte
	Roster  *onet.Roster
}
