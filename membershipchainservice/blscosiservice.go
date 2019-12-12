package membershipchainservice

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"sort"

	"go.dedis.ch/protobuf"

	gpr "github.com/dedis/student_19_nyleCtrlPlane/gossipregistrationprotocol"
	"go.dedis.ch/cothority/v3/blscosi/protocol"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

var (
	PINGSMSG   = []byte("Do we agree on Pings ?")
	SIGNERSMSG = []byte("Do we agree on Signers ?")
)

// SignatureRequest treats external request to this service.
func (s *Service) SignatureRequest(req *SignatureRequest) (network.Message, error) {
	if s.Cycle.GetCurrentPhase() != REGISTRATION {
		return nil, errors.New("Registration was not made in time")
	}
	// generate the tree
	nNodes := len(req.Roster.List)
	//s.Threshold = nNodes / 4
	//s.NSubtrees = 2

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

func (s *Service) getHashPings() []byte {
	h := s.suite.Hash()
	str := "map["
	s.PingMapMtx.Lock()
	// As Ping Distance is a map, one have to print in a sorted way, to have the same Hash.
	keys := []string{}
	for k := range s.PingDistances {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		str += k + ":["
		subkeys := []string{}
		for k := range s.PingDistances[k] {
			subkeys = append(subkeys, k)
		}
		sort.Strings(subkeys)

		for _, sk := range subkeys {
			str += " " + sk + "-"
			str += fmt.Sprintf("%.2f", s.PingDistances[k][sk]) + ", "
		}
		str += "]"
	}

	str += "]"
	s.PingMapMtx.Unlock()
	buf := []byte(str)
	h.Write(buf)
	return h.Sum(nil)
}

// AgreeStateSubProtocol will get a signed message if the states of the nodes are the same
func (s *Service) AgreeStateSubProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	vf := func(a, b []byte) bool {
		var st State
		err := protobuf.Decode(b, &st)
		if err != nil {
			panic(err)
		}

		if bytes.Equal(a, SIGNERSMSG) {
			if !reflect.DeepEqual(getKeys(s.GetSigners(st.Epoch).Set), st.Signers) {
				log.LLvl1(" \033[38;5;1m", s.Name, " recieved different Signers\033[0m")
			}
			return reflect.DeepEqual(getKeys(s.GetSigners(st.Epoch).Set), st.Signers)
		}

		if !bytes.Equal(st.HashPings, s.getHashPings()) {

			h := s.suite.Hash()
			buf := []byte("map[]")
			h.Write(buf)
			emptyHash := h.Sum(nil)

			if bytes.Equal(st.HashPings, emptyHash) {
				log.LLvl1(" \033[38;5;1m", s.Name, " recieved empty Pings \n", emptyHash, "\n", st.HashPings, " \033[0m")
			}
			if bytes.Equal(s.getHashPings(), emptyHash) {
				log.LLvl1(" \033[38;5;1m", s.Name, " has an empty Pings \n", emptyHash, "\n", s.getHashPings(), " \033[0m")
			}
			log.LLvl1(" \033[38;5;1m", s.Name, " recieved different Pings : 1 own. 2 recieved \n", s.getHashPings(), "\n", st.HashPings, "\033[0m")
			log.LLvl1(s.Name, s.PingDistances, s.getHashPings())
		}

		return bytes.Equal(st.HashPings, s.getHashPings())

	}
	return protocol.NewSubBlsCosi(n, vf, pairing.NewSuiteBn256())
}

// AgreeStateProtocol will call the AgreeStateSubProtocol
func AgreeStateProtocol(n *onet.TreeNodeInstance) (onet.ProtocolInstance, error) {
	vf := func(a, b []byte) bool { return true }
	return protocol.NewBlsCosi(n, vf, agreeSubProtocolName, pairing.NewSuiteBn256())
}
