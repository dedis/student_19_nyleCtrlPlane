package membershipchainservice

/*
The service.go defines what to do for each API-call. This part of the service
runs on the node.
*/

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dedis/student_19_nyleCtrlPlane/gentree"
	gpr "github.com/dedis/student_19_nyleCtrlPlane/gossipregistrationprotocol"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/kyber/v3/suites"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

// For Blscosi
const protocolTimeout = 20 * time.Second

var suite = suites.MustFind("bn256.adapter").(*pairing.SuiteBn256)

// Used for tests
var MembershipID onet.ServiceID

// ServiceName is used for registration on the onet.
const ServiceName = "MemberchainService"

func init() {
	var err error
	MembershipID, err = onet.RegisterNewServiceWithSuite(ServiceName, suite, newService)
	log.ErrFatal(err)
	network.RegisterMessages(&storage{}, &gpr.Announce{})
	execReqPingsMsgID = network.RegisterMessage(&ReqPings{})
	execReplyPingsMsgID = network.RegisterMessage(&ReplyPings{})

}

// Service is our template-service
type Service struct {
	// We need to embed the ServiceProcessor, so that incoming messages
	// are correctly handled.
	*onet.ServiceProcessor
	storage *storage
	e       Epoch

	// From BlsCoSi
	Threshold int
	NSubtrees int
	Timeout   time.Duration
	suite     pairing.Suite

	// From Crux
	Nodes             gentree.LocalityNodes
	GraphTree         map[string][]GraphTree
	BinaryTree        map[string][]*onet.Tree
	alive             bool
	Distances         map[*gentree.LocalityNode]map[*gentree.LocalityNode]gentree.Compact
	PingDistances     map[string]map[string]float64
	ShortestDistances map[string]map[string]float64
	OwnPings          map[string]float64
	DonePing          bool
	PingMapMtx        sync.Mutex
	PingAnswerMtx     sync.Mutex
	NrPingAnswers     int
}

// storageID reflects the data we're storing - we could store more
// than one structure.
var storageID = []byte("main")

// storage is used to save our data.
type storage struct {
	Signers []SignersSet
	sync.Mutex
}

// SetGenesisSigners is used to let now to the node what are the first signers.
func (s *Service) SetGenesisSigners(p SignersSet) {
	s.e = 0
	s.storage.Lock()
	s.storage.Signers = append(s.storage.Signers, p)
	s.storage.Unlock()
}

func (s *Service) addSignerFromMessage(ann gpr.Announce) error {
	return s.addSigner(ann.Signer, ann.Proof, ann.Epoch)
}

// addSigner will add one signer to the storage if the proof is convincing
func (s *Service) addSigner(signer network.ServerIdentityID, proof *gpr.SignatureResponse, e int) error {
	if proof.Signature != nil {
		if e < 0 {
			return errors.New("Epoch cannot be negative")
		}
		if e > len(s.storage.Signers) {
			return errors.New("Epoch is too in the future")
		}
		s.storage.Lock()
		if e == len(s.storage.Signers) {
			s.storage.Signers = append(s.storage.Signers, make(SignersSet))
		}
		s.storage.Signers[Epoch(e)][signer] = true
		s.storage.Unlock()
		return nil
	}
	return errors.New("No signature")

}

// GetSigners gives the registrations that are stored on this node
func (s *Service) GetSigners(e Epoch) *SignersReply {
	if e < 0 || e >= Epoch(len(s.storage.Signers)) {
		return &SignersReply{Set: nil}
	}
	s.storage.Lock()
	defer s.storage.Unlock()
	return &SignersReply{Set: s.storage.Signers[e]}

}

func getKeys(m SignersSet) []network.ServerIdentityID {
	var keys []network.ServerIdentityID
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (s *Service) getServerIdentityFromSignersSet(m SignersSet, ro *onet.Roster) ([]*network.ServerIdentity, error) {
	mbrsIDs := getKeys(m)
	var mbrs []*network.ServerIdentity
	for _, mID := range mbrsIDs {
		_, si := ro.Search(mID)
		if si == nil {
			return nil, errors.New("Server Identity not found in Roster")
		}
		mbrs = append(mbrs, si)
	}
	return mbrs, nil

}

// Registrate will get signatures from Signers,
// then propagate the signed block to all the nodes it is aware of to be registred as new Signers
func (s *Service) Registrate(roster *onet.Roster, e Epoch) error {
	if s.e != e-1 {
		return fmt.Errorf("Cannot register for epoch %d, as system is at epoch", s.e)
	}

	msg := []byte("Register me !")

	s.storage.Lock()
	mbrs, err := s.getServerIdentityFromSignersSet(s.storage.Signers[e-1], roster)
	if err != nil {
		return err
	}
	if _, ok := s.storage.Signers[e-1][s.ServerIdentity().ID]; !ok {
		mbrs = append(mbrs, s.ServerIdentity())
	}

	// Register itself
	if e == Epoch(len(s.storage.Signers)) {
		s.storage.Signers = append(s.storage.Signers, make(SignersSet))
	}
	s.storage.Signers[e][s.ServerIdentity().ID] = true
	s.storage.Unlock()

	if len(mbrs) == 1 {
		return fmt.Errorf("No signers for epoch %d", e)
	}
	ro := onet.NewRoster(mbrs)

	log.LLvl1("Start Signing")
	buf, err := s.SignatureRequest(&SignatureRequest{Message: msg, Roster: ro})
	if err != nil {
		return err
	}
	log.LLvl1("End Signing")

	nbrNodes := len(roster.List) - 1
	tree := roster.GenerateNaryTreeWithRoot(nbrNodes, s.ServerIdentity())
	pi, err := s.CreateProtocol(gpr.Name, tree)
	if err != nil {
		return errors.New("Couldn't make new protocol: " + err.Error())
	}

	signResp := buf.(*gpr.SignatureResponse)

	log.LLvl1(signResp)

	p := pi.(*gpr.GossipRegistationProtocol)
	p.Msg = gpr.Announce{
		Signer: s.ServerIdentity().ID,
		Proof:  signResp,
		Epoch:  int(e),
	}

	log.LLvl1(p.Msg)

	p.Start()

	select {
	case <-p.ConfirmationsChan:
		return nil
	}

}

// StartNewEpoch stop the registration for nodes and run CRUX
func (s *Service) StartNewEpoch(roster *onet.Roster, nodes []*gentree.LocalityNode) error {
	s.e++

	mbrs, err := s.getServerIdentityFromSignersSet(s.storage.Signers[s.e], roster)
	ro := onet.NewRoster(mbrs)

	si2name := make(map[*network.ServerIdentity]string)
	for _, n := range nodes {
		si2name[n.ServerIdentity] = n.Name
	}

	s.Setup(&InitRequest{
		Nodes:                nodes,
		ServerIdentityToName: si2name,
		NrOps:                10,
		OpIdxStart:           0,
		Roster:               ro,
	})

	/*
		log.LLvl1("HERE : ", s.ServerIdentity(), "\n\n")

		log.LLvl1("Nodes", s.Nodes, "\n")
		log.LLvl1("Graph Tree ", s.GraphTree)
		log.LLvl1("Binary", s.BinaryTree)
		log.LLvl1("Alive", s.alive)
		log.LLvl1("Distances", s.Distances)
		log.LLvl1("Ping Distances", s.PingDistances)
		log.LLvl1("Shortest Distances", s.ShortestDistances)
		log.LLvl1("Own Pings", s.OwnPings)
		log.LLvl1("Done Ping", s.DonePing)
		log.LLvl1("Ping MAP MTX", s.PingMapMtx)
		log.LLvl1("Ping MTX", s.PingAnswerMtx)
		log.LLvl1("Ping NrPingAnswer", s.NrPingAnswers)

		log.LLvl1("\n\n")
	*/

	return err
}

func (s *Service) Deregistrate() error {
	return errors.New("Unimplemented Error")
}

func (s *Service) ChangeLatencies(ic float64) {

}

// NewProtocol is called on all nodes of a Tree (except the root, since it is
// the one starting the protocol) so it's the Service that will be called to
// generate the PI on all others node.
// If you use CreateProtocolOnet, this will not be called, as the Onet will
// instantiate the protocol on its own. If you need more control at the
// instantiation of the protocol, use CreateProtocolService, and you can
// give some extra-configuration to your protocol in here.
func (s *Service) NewProtocol(tn *onet.TreeNodeInstance, conf *onet.GenericConfig) (onet.ProtocolInstance, error) {
	log.Lvl3("Not templated yet")
	return nil, nil
}

// saves all data.
func (s *Service) save() {
	s.storage.Lock()
	defer s.storage.Unlock()
	err := s.Save(storageID, s.storage)
	if err != nil {
		log.Error("Couldn't save data:", err)
	}
}

// Tries to load the configuration and updates the data in the service
// if it finds a valid config-file.
func (s *Service) tryLoad() error {
	s.storage = &storage{}
	msg, err := s.Load(storageID)
	if err != nil {
		return err
	}
	if msg == nil {
		return nil
	}
	var ok bool
	s.storage, ok = msg.(*storage)
	if !ok {
		return errors.New("Data of wrong type")
	}
	return nil
}

// newService receives the context that holds information about the node it's
// running on. Saving and loading can be done using the context. The data will
// be stored in memory for tests and simulations, and on disk for real deployments.
func newService(c *onet.Context) (onet.Service, error) {
	s := &Service{
		ServiceProcessor: onet.NewServiceProcessor(c),
		Timeout:          protocolTimeout,
		suite:            suite,
	}

	// configure the Gossiping protocol
	s.RegisterProcessorFunc(execReqPingsMsgID, s.ExecReqPings)
	s.RegisterProcessorFunc(execReplyPingsMsgID, s.ExecReplyPings)

	_, err := s.ProtocolRegister(gpr.Name, gpr.NewGossipProtocol(s.addSignerFromMessage))
	if err != nil {
		return nil, err
	}

	if err = s.tryLoad(); err != nil {
		log.Error(err)
		return nil, err
	}
	return s, nil
}
