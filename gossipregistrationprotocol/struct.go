package gossipregistrationprotocol

/*
Struct holds the messages that will be sent around in the protocol. You have
to define each message twice: once the actual message, and a second time
with the `*onet.TreeNode` embedded. The latter is used in the handler-function
so that it can find out who sent the message.
*/

import (
	"go.dedis.ch/cothority/v3/blscosi/protocol"
	"go.dedis.ch/onet/network"
	"go.dedis.ch/onet/v3"
)

// Name can be used from other packages to refer to this protocol.
const Name = "GossipRegistrationProtocol"

// announceWrapper just contains Announce and the data necessary to identify
// and process the message in onet.
type announceWrapper struct {
	*onet.TreeNode
	Announce
}

// Reply returns the sum of all children random number
type Reply struct {
	Confirmations int
}

// replyWrapper just contains Reply and the data necessary to identify and
// process the message in onet.
type replyWrapper struct {
	*onet.TreeNode
	Reply
}

// Announce is used by the gossip protocole
type Announce struct {
	Signer network.ServerIdentityID
	Proof  *SignatureResponse
	Epoch  int
}

// SignatureResponse is what the Cosi service will reply to clients.
type SignatureResponse struct {
	Hash      []byte
	Signature protocol.BlsSignature
}
