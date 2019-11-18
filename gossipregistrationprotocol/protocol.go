package gossipregistrationprotocol

/*
The `NewProtocol` method is used to define the protocol and to register
the handlers that will be called if a certain type of message is received.
The handlers will be treated according to their signature.

The protocol-file defines the actions that the protocol needs to do in each
step. The root-node will call the `Start`-method of the protocol. Each
node will only use the `Handle`-methods, and not call `Start` again.
*/

import (
	"go.dedis.ch/onet/v3"
)

// AddSignersCallback is a callback function that is called in the protocol
type AddSignersCallback func(Announce) error

// GossipRegistationProtocol holds the state of gossip
type GossipRegistationProtocol struct {
	*onet.TreeNodeInstance
	Msg               Announce
	addSigners        AddSignersCallback
	announceChan      chan announceWrapper
	repliesChan       chan []replyWrapper
	ConfirmationsChan chan int
}

// Check that *TemplateProtocol implements onet.ProtocolInstance
var _ onet.ProtocolInstance = (*GossipRegistationProtocol)(nil)

// NewGossipProtocol initialises the structure for use in one round
func NewGossipProtocol(addSigners AddSignersCallback) func(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	return func(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
		t := &GossipRegistationProtocol{
			TreeNodeInstance:  n,
			ConfirmationsChan: make(chan int),
			addSigners:        addSigners,
		}
		if err := n.RegisterChannels(&t.announceChan, &t.repliesChan); err != nil {
			return nil, err
		}
		return t, nil
	}
}

// Start sends the Announce-message to all children
func (p *GossipRegistationProtocol) Start() error {
	return p.SendTo(p.TreeNode(), &p.Msg)
}

// Dispatch implements the main logic of the protocol. The function is only
// called once. The protocol is considered finished when Dispatch returns and
// Done is called.
func (p *GossipRegistationProtocol) Dispatch() error {

	defer p.Done()
	nConf := 1

	ann := <-p.announceChan
	p.addSigners(ann.Announce)

	if p.IsLeaf() {
		return p.SendToParent(&Reply{nConf})
	}
	p.SendToChildren(&ann.Announce)

	replies := <-p.repliesChan
	for _, r := range replies {
		nConf += r.Confirmations
	}

	if !p.IsRoot() {
		return p.SendToParent(&Reply{nConf})
	}
	p.ConfirmationsChan <- nConf
	return nil
}
