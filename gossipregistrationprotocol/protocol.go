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
	"time"

	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
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
	TimeOut           time.Duration
}

// Check that *TemplateProtocol implements onet.ProtocolInstance
var _ onet.ProtocolInstance = (*GossipRegistationProtocol)(nil)

// NewGossipProtocol initialises the structure for use in one round
func NewGossipProtocol(addSigners AddSignersCallback) func(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	return func(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
		t := &GossipRegistationProtocol{
			TreeNodeInstance: n,
			// buffered channel does not block
			ConfirmationsChan: make(chan int, 1),
			addSigners:        addSigners,
			TimeOut:           200 * time.Millisecond,
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
	startTime := time.Now()
	nConf := 1
	ann := <-p.announceChan
	err := p.addSigners(ann.Announce)
	if err != nil {
		log.LLvl1("Add Signer error in gossip protocol : ", err)
		nConf = 0
	}
	log.LLvl1(p.Info(), "Add Signers", p.Host(), p.Index(), "after", time.Now().Sub(startTime))
	if p.IsLeaf() {
		log.LLvl1(p.Info(), "is leaf", p.Host(), p.Index(), "after", time.Now().Sub(startTime))
		return p.SendToParent(&Reply{nConf})
	}
	p.SendToChildren(&ann.Announce)
	select {
	case replies := <-p.repliesChan:

		for _, r := range replies {
			nConf += r.Confirmations
		}
		if !p.IsRoot() {
			return p.SendToParent(&Reply{nConf})
		}
		log.LLvl1(p.Info(), "Finished", p.Host(), p.Index(), "after", time.Now().Sub(startTime))
		p.ConfirmationsChan <- nConf
	case <-time.After(p.TimeOut):
		log.LLvl1(p.Info(), "Time Out", p.Host(), p.Index(), "after", time.Now().Sub(startTime))
		p.ConfirmationsChan <- nConf
	}

	return nil
}

// Shutdown close the cahan at the end of the protocol
func (p *GossipRegistationProtocol) Shutdown() error {
	return nil
}
