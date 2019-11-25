package membershipchainservice

import (
	"errors"
	"reflect"

	"go.dedis.ch/protobuf"

	gpr "github.com/dedis/student_19_nyleCtrlPlane/gossipregistrationprotocol"
	"go.dedis.ch/cothority/v3/blscosi/protocol"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

// SignatureRequest treats external request to this service.
func (s *Service) SignatureRequest(req *SignatureRequest) (network.Message, error) {
	// generate the tree
	nNodes := len(req.Roster.List)
	rooted := req.Roster.NewRosterWithRoot(s.ServerIdentity())
	if rooted == nil {
		return nil, errors.New("we're not in the roster")
	}
	tree := rooted.GenerateNaryTree(nNodes)
	if tree == nil {
		return nil, errors.New("failed to generate tree")
	}

	// configure the BlsCosi protocol
	pi, err := s.CreateProtocol(protocol.DefaultProtocolName, tree)
	if err != nil {
		return nil, errors.New("Couldn't make new protocol: " + err.Error())
	}
	p := pi.(*protocol.BlsCosi)
	p.CreateProtocol = s.CreateProtocol
	p.Timeout = s.Timeout
	p.Msg = req.Message

	// Threshold before the subtrees so that we can optimize situation
	// like a threshold of one
	if s.Threshold > 0 {
		p.Threshold = s.Threshold
	}

	if s.NSubtrees > 0 {
		err = p.SetNbrSubTree(s.NSubtrees)
		if err != nil {
			p.Done()
			return nil, err
		}
	}

	// start the protocol
	log.Lvl3("Cosi Service starting up root protocol")
	if err = p.Start(); err != nil {
		return nil, err
	}

	// wait for reply. This will always eventually return.
	sig := <-p.FinalSignature

	// The hash is the message blscosi actually signs, we recompute it the
	// same way as blscosi and then return it.
	h := s.suite.Hash()
	h.Write(req.Message)
	return &gpr.SignatureResponse{h.Sum(nil), sig}, nil
}

const agreeProtocolName = "AgreeProtocol"
const agreeSubProtocolName = "AgreeSubProtocol"

// AgreeStateSubProtocol will get a signed message if the states of the nodes are the same
func (s *Service) AgreeStateSubProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	vf := func(a, b []byte) bool {
		var st State
		err := protobuf.Decode(b, &st)
		if err != nil {
			panic(err)
		}
		return reflect.DeepEqual(getKeys(s.GetSigners(st.Epoch).Set), st.Signers) && reflect.DeepEqual(s.GraphTree, st.GraphTree)
	}
	return protocol.NewSubBlsCosi(n, vf, pairing.NewSuiteBn256())
}

// AgreeStateProtocol will call the AgreeStateSubProtocol
func AgreeStateProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	vf := func(a, b []byte) bool { return true }
	return protocol.NewBlsCosi(n, vf, agreeSubProtocolName, pairing.NewSuiteBn256())
}
