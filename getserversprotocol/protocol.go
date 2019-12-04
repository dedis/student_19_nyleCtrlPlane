package getserversprotocol

import (
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/network"
)

// GetServersProtocol holds the state of GetServer
type GetServersProtocol struct {
	*onet.TreeNodeInstance
	GetServers        func() map[string]*network.ServerIdentity
	announceChan      chan announceWrapper
	repliesChan       chan []replyWrapper
	ConfirmationsChan chan Reply
}

// Check that *GetServersProtocol implements onet.ProtocolInstance
var _ onet.ProtocolInstance = (*GetServersProtocol)(nil)

// NewGetServersProtocol initialises the structure for use in one round
func NewGetServersProtocol(getServers func() map[string]*network.ServerIdentity) func(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	return func(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
		t := &GetServersProtocol{
			TreeNodeInstance:  n,
			ConfirmationsChan: make(chan Reply),
			GetServers:        getServers,
		}
		if err := n.RegisterChannels(&t.announceChan, &t.repliesChan); err != nil {
			return nil, err
		}
		return t, nil
	}
}

// Start sends the Announce-message to all children
func (p *GetServersProtocol) Start() error {
	return p.SendTo(p.TreeNode(), &Announce{"Let's go"})
}

// Dispatch implements the main logic of the protocol. The function is only
// called once. The protocol is considered finished when Dispatch returns and
// Done is called.
func (p *GetServersProtocol) Dispatch() error {

	defer p.Done()
	nConf := 1
	servers := p.GetServers()

	ann := <-p.announceChan

	if p.IsLeaf() {
		return p.SendToParent(&Reply{nConf, servers})
	}
	p.SendToChildren(&ann.Announce)

	replies := <-p.repliesChan
	for _, r := range replies {
		nConf += r.Confirmations
		for k, v := range r.Servers {
			servers[k] = v
		}
	}

	if !p.IsRoot() {
		return p.SendToParent(&Reply{nConf, servers})
	}
	p.ConfirmationsChan <- Reply{nConf, servers}
	return nil
}
