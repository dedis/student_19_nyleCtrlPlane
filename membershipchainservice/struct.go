package membershipchainservice

import (
	"fmt"

	gpr "github.com/dedis/student_19_nyleCtrlPlane/gossipregistrationprotocol"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/network"
)

// We need to register all messages so the network knows how to handle them.
func init() {
	network.RegisterMessages(
		GossipArgs{}, GossipReply{}, SignersReply{}, State{},
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

// Return Radius, + list of Server Identity of Nodes
func (g GraphTree) String() string {
	ret := fmt.Sprintf("[Radius=%v", g.Radius)
	ret += ",trees=["
	for i, tn := range g.ListOfNodes {
		ret += fmt.Sprintf("%v", tn.ServerIdentity.Address)
		if i != len(g.ListOfNodes)-1 {
			ret += ","
		}
	}
	ret += "]]"
	return ret
}

// GraphTrees maps name to a list of GraphTrees
type GraphTrees map[string][]GraphTree

func (g GraphTrees) String() string {
	ret := fmt.Sprintf("Name;List of Trees\n")
	for name, listGt := range g {
		ret += name + ";[" + fmt.Sprintf("%v", listGt) + "]\n"
	}
	return ret
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

// ReqHistory is use to request info about the current version
type ReqHistory struct {
	SenderName     string
	SenderIdentity *network.ServerIdentity
}

// ReplyHistory hold the reply to the history request
type ReplyHistory struct {
	SenderName           string
	Servers              map[string]*network.ServerIdentity
	ServerIdentityToName map[network.ServerIdentityID]string
	SignersKey           []network.ServerIdentityID
	SignersValue         []gpr.SignatureResponse
	SignersIndex         []int
}

// SignatureRequest is what the Cosi service is expected to receive from clients.
type SignatureRequest struct {
	Message []byte
	Roster  *onet.Roster
}

// State describes the state of one node
type State struct {
	Signers   []network.ServerIdentityID
	HashPings []byte
	Epoch     Epoch
}
