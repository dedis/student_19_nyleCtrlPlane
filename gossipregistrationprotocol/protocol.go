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
	"go.dedis.ch/onet/v3/log"
)

func init() {
	_, err := onet.GlobalProtocolRegister(Name, NewProtocol)
	if err != nil {
		panic(err)
	}
}

// GossipRegistationProtocol holds the state of gossip
type GossipRegistationProtocol struct {
	*onet.TreeNodeInstance
	announceChan      chan announceWrapper
	repliesChan       chan []replyWrapper
	ConfirmationsChan chan int

	ParticipantsList []string
}

// Check that *TemplateProtocol implements onet.ProtocolInstance
var _ onet.ProtocolInstance = (*GossipRegistationProtocol)(nil)

// NewProtocol initialises the structure for use in one round
func NewProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	t := &GossipRegistationProtocol{
		TreeNodeInstance:  n,
		ConfirmationsChan: make(chan int),
	}
	if err := n.RegisterChannels(&t.announceChan, &t.repliesChan); err != nil {
		return nil, err
	}
	return t, nil
}

// Start sends the Announce-message to all children
func (p *GossipRegistationProtocol) Start() error {
	log.LLvl3(p.ServerIdentity(), "Starting Gossip")
	return p.SendTo(p.TreeNode(), &Announce{string(p.ServerIdentity().Address)})
}

// Dispatch implements the main logic of the protocol. The function is only
// called once. The protocol is considered finished when Dispatch returns and
// Done is called.
func (p *GossipRegistationProtocol) Dispatch() error {
	defer p.Done()

	nConf := 1

	ann := <-p.announceChan
	p.ParticipantsList = append(p.ParticipantsList, ann.Announce.Message)
	log.LLvl3("Participants List : ", p.ParticipantsList)

	if p.IsLeaf() {
		return p.SendToParent(&Reply{nConf})
	}
	p.SendToChildren(&ann.Announce)

	replies := <-p.repliesChan
	for _, c := range replies {
		nConf += c.Confirmations
	}

	if !p.IsRoot() {
		return p.SendToParent(&Reply{nConf})
	}

	p.ConfirmationsChan <- nConf
	return nil
}
