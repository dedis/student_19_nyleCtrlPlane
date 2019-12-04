package membershipchainservice

/*
The service.go defines what to do for each API-call. This part of the service
runs on the node.
*/

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/dedis/student_19_nyleCtrlPlane/gentree"
	"github.com/dedis/student_19_nyleCtrlPlane/getserversprotocol"
	gpr "github.com/dedis/student_19_nyleCtrlPlane/gossipregistrationprotocol"
	"go.dedis.ch/cothority/v3/blscosi/protocol"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/kyber/v3/suites"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"go.dedis.ch/protobuf"
)

// For Blscosi
const protocolTimeout = 20 * time.Second

var suite = suites.MustFind("bn256.adapter").(*pairing.SuiteBn256)

// Used for tests
var MembershipID onet.ServiceID

// ServiceName is used for registration on the onet.
const ServiceName = "MemberchainService"

var execReqHistoryMsgID network.MessageTypeID
var execReplyHistoryMsgID network.MessageTypeID

func init() {
	var err error
	MembershipID, err = onet.RegisterNewServiceWithSuite(ServiceName, suite, newService)
	log.ErrFatal(err)
	network.RegisterMessages(&storage{}, &gpr.Announce{})
	execReqPingsMsgID = network.RegisterMessage(&ReqPings{})
	execReplyPingsMsgID = network.RegisterMessage(&ReplyPings{})
	execReqHistoryMsgID = network.RegisterMessage(&ReqHistory{})
	execReplyHistoryMsgID = network.RegisterMessage(&ReplyHistory{})

}

// Service is our template-service
type Service struct {
	// We need to embed the ServiceProcessor, so that incoming messages
	// are correctly handled.
	*onet.ServiceProcessor
	storage *storage
	e       Epoch
	Proof   *gpr.SignatureResponse
	Cycle   Cycle

	// All services maintain a list of the servers it heard of.
	// Helps recreate the roster for each epoch
	Name                 string
	ServersMtx           sync.Mutex
	Servers              map[string]*network.ServerIdentity
	ServerIdentityToName map[network.ServerIdentityID]string

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

	PrefixForReadingFile string
	DoneUpdate           bool
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
func (s *Service) SetGenesisSigners(servers map[*network.ServerIdentity]string) {
	s.Cycle.Sequence = []time.Duration{REGISTRATION_DUR, EPOCH_DUR}
	s.Cycle.StartTime = time.Now()

	s.ServerIdentityToName = make(map[network.ServerIdentityID]string)
	s.ServersMtx.Lock()
	s.Servers = make(map[string]*network.ServerIdentity)
	s.Servers[s.Name] = s.ServerIdentity()
	signers := make(SignersSet)
	for si, name := range servers {
		s.Servers[name] = si
		s.ServerIdentityToName[si.ID] = name
		signers[si.ID] = gpr.SignatureResponse{Hash: []uint8{}, Signature: []uint8{}}
	}
	s.ServersMtx.Unlock()

	s.e = 0
	s.storage.Lock()
	s.storage.Signers = append(s.storage.Signers, make(SignersSet))
	s.storage.Signers[0] = signers
	s.storage.Unlock()
}

func (s *Service) addSignerFromMessage(ann gpr.Announce) error {
	s.ServersMtx.Lock()
	s.Servers[ann.Name] = ann.Server
	s.ServerIdentityToName[ann.Signer] = ann.Name
	s.ServersMtx.Unlock()
	return s.addSigner(ann.Signer, ann.Proof, ann.Epoch)
}

// addSigner will add one signer to the storage if the proof is convincing
func (s *Service) addSigner(signer network.ServerIdentityID, proof *gpr.SignatureResponse, e int) error {
	if proof != nil {
		if e < 0 {
			return errors.New("Epoch cannot be negative")
		}
		s.storage.Lock()

		if e > len(s.storage.Signers) {
			log.LLvl1(" Error in add signer ? ")
			return errors.New("Epoch is too in the future")
		}

		if e == len(s.storage.Signers) {
			s.storage.Signers = append(s.storage.Signers, make(SignersSet))
		}
		s.storage.Signers[Epoch(e)][signer] = *proof
		s.storage.Unlock()
		return nil
	}
	return fmt.Errorf("Addsigner cannot be completed for %v as %v did not send a signature", s.Name, signer)

}

// GetEpoch returns the current epoch
func (s *Service) GetEpoch() Epoch {
	return s.e
}

// GetGlobalServers gossips on existing info to get info about all the Servers
func (s *Service) GetGlobalServers() map[string]*network.ServerIdentity {
	ro := s.getGlobalRoster()

	nbrNodes := len(ro.List) - 1
	tree := ro.GenerateNaryTreeWithRoot(nbrNodes, s.ServerIdentity())
	pi, err := s.CreateProtocol(getserversprotocol.Name, tree)
	if err != nil {
		panic(errors.New("Couldn't make new protocol: " + err.Error()))
	}

	p := pi.(*getserversprotocol.GetServersProtocol)
	p.Start()

	r := <-p.ConfirmationsChan
	s.ServersMtx.Lock()
	for name, si := range r.Servers {
		s.Servers[name] = si
	}
	s.ServersMtx.Unlock()

	return s.Servers

}

// GetSigners gives the registrations that are stored on this node
func (s *Service) GetSigners(e Epoch) *SignersReply {
	s.storage.Lock()
	defer s.storage.Unlock()
	if e < 0 || e >= Epoch(len(s.storage.Signers)) {
		return &SignersReply{Set: nil}
	}
	return &SignersReply{Set: s.storage.Signers[e]}

}

func getKeys(m SignersSet) []network.ServerIdentityID {
	var keys []network.ServerIdentityID
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool {
		aB := [16]byte(keys[a])
		bB := [16]byte(keys[b])
		return bytes.Compare(aB[:], bB[:]) < 0
	})
	return keys
}
func (s *Service) getGlobalRoster() *onet.Roster {
	sis := []*network.ServerIdentity{}
	s.ServersMtx.Lock()
	for _, v := range s.Servers {
		sis = append(sis, v)
	}
	s.ServersMtx.Unlock()

	return onet.NewRoster(sis)
}

func (s *Service) getServerIdentityFromSignersSet(m SignersSet) ([]*network.ServerIdentity, error) {
	mbrsIDs := getKeys(m)
	var mbrs []*network.ServerIdentity
	ro := s.getGlobalRoster()
	for _, mID := range mbrsIDs {
		_, si := ro.Search(mID)
		if si == nil {
			return nil, errors.New("Server Identity not found in Roster")
		}
		mbrs = append(mbrs, si)
	}
	return mbrs, nil

}

// CreateProofForEpoch will get signatures from Signers from previous epoch
func (s *Service) CreateProofForEpoch(e Epoch) error {
	if s.Cycle.GetCurrentPhase() != REGISTRATION {
		log.LLvl1(s.Name, "is waiting ", s.Cycle.GetTimeTillNextCycle(), "s to register")
		time.Sleep(s.Cycle.GetTimeTillNextCycle())
	}

	log.Lvl1(s.ServerIdentity(), " is creating proof for Epoch : ", e)
	if s.e != e-1 {
		log.LLvl1(s.ServerIdentity(), "is having an error")
		return fmt.Errorf("Cannot register for epoch %d, as system is at epoch", s.e)
	}

	// Get proof from the signer of epoch e-1
	msg := []byte("Register me !")

	s.storage.Lock()
	if len(s.storage.Signers) <= int(e-1) {
		return fmt.Errorf("Storage not up-to-date, No signers for epoch %d", e)
	}

	mbrs, err := s.getServerIdentityFromSignersSet(s.storage.Signers[e-1])
	if err != nil {
		return err
	}
	if len(mbrs) == 0 {
		return fmt.Errorf("No signers for epoch %d", e)
	}
	if _, ok := s.storage.Signers[e-1][s.ServerIdentity().ID]; !ok {
		mbrs = append(mbrs, s.ServerIdentity())
	}
	s.storage.Unlock()

	ro := onet.NewRoster(mbrs)

	buf, err := s.SignatureRequest(&SignatureRequest{Message: msg, Roster: ro})
	if err != nil {
		return err
	}

	s.Proof = buf.(*gpr.SignatureResponse)
	if s.Proof == nil {
		log.LLvl1(s.Name, " :cannot share proof as it did not manage to get one")
		return fmt.Errorf("%v cannot share proof as it did not manage to get one", s.Name)
	}

	// Share first to the old signers. That way they will have a view of the global system that they can transmit to the others
	err = s.ShareProof()
	return err

}

// ShareProof will send the proof created in CreateProofForEpoch to all the nodes it is aware of
// It starts by getting informations about the other servers
func (s *Service) ShareProof() error {
	if s.Cycle.GetCurrentPhase() == EPOCH {
		log.LLvl1(s.Name, " is waiting for the end of Epoch :", s.Cycle.GetEpoch())
		time.Sleep(s.Cycle.GetTimeTillNextCycle())
	}

	// Get info about all the servers in the system
	s.GetGlobalServers()
	roForPropa := s.getGlobalRoster().NewRosterWithRoot(s.ServerIdentity())

	log.Lvl1("Roster for Propagation : len : ", len(roForPropa.List), " Values : ", roForPropa.List)

	// Send them the proof
	tree := roForPropa.GenerateBinaryTree()
	pi, err := s.CreateProtocol(gpr.Name, tree)
	if err != nil {
		return errors.New("Couldn't make new protocol: " + err.Error())
	}

	p := pi.(*gpr.GossipRegistationProtocol)
	p.Msg = gpr.Announce{
		Name:   s.Name,
		Server: s.ServerIdentity(),
		Signer: s.ServerIdentity().ID,
		Proof:  s.Proof,
		Epoch:  int(s.e + 1),
	}

	p.Start()

	<-p.ConfirmationsChan
	return nil

}

// StartNewEpoch stop the registration for nodes and run CRUX
func (s *Service) StartNewEpoch() error {
	if s.Cycle.GetCurrentPhase() != EPOCH {
		log.LLvl1(s.Name, "is waiting ", s.Cycle.GetTimeTillNextEpoch(), "s to start the new Epoch")
		time.Sleep(s.Cycle.GetTimeTillNextEpoch())
	}
	if s.e != s.Cycle.GetEpoch() {
		return fmt.Errorf("Its not the time for epoch %d. The clock says its %d", s.e+1, s.Cycle.GetEpoch())
	}

	s.e++
	s.storage.Lock()
	mbrs, err := s.getServerIdentityFromSignersSet(s.storage.Signers[s.e])
	if err != nil {
		defer s.storage.Unlock()
		return err
	}
	if _, ok := s.storage.Signers[s.e][s.ServerIdentity().ID]; !ok {
		defer s.storage.Unlock()
		return errors.New("One node cannot start a new Epoch if it didn't registrate")
	}
	s.storage.Unlock()

	ro := onet.NewRoster(mbrs)
	// Agree on Signers
	err = s.AgreeOnState(ro, SIGNERSMSG)
	if err != nil {
		log.LLvl1(" \033[38;5;1m", s.Name, " is not passing the Signers Agree, Error :   ", err, " \033[0m")
		return err
	}

	si2name := make(map[*network.ServerIdentity]string)
	for _, serv := range ro.List {
		si2name[serv] = s.ServerIdentityToName[serv.ID]
	}
	s.Setup(&InitRequest{
		ServerIdentityToName: si2name,
	})

	err = s.AgreeOnState(ro, PINGSMSG)
	if err != nil {
		log.LLvl1("\033[39;5;1m", s.Name, " is not passing the PINGS Agree, Error :   ", err, " \033[0m")
		return err
	}
	log.Lvl1("\033[45;5;1m", s.Name, " Finished Epoch ", s.e, " Successfully.\033[0m")
	return err
}

func (s *Service) Deregistrate() error {
	return errors.New("Unimplemented Error")
}

func (s *Service) ChangeLatencies(ic float64) {
}

// AgreeOnState checks that the members of the roster have the same signers + same maps
func (s *Service) AgreeOnState(roster *onet.Roster, msg []byte) error {
	// generate the tree
	nNodes := len(roster.List)
	rooted := roster.NewRosterWithRoot(s.ServerIdentity())
	if rooted == nil {
		return errors.New("we're not in the roster")
	}
	tree := rooted.GenerateNaryTree(nNodes)
	if tree == nil {
		return errors.New("failed to generate tree")
	}

	// configure the BlsCosi protocol
	pi, err := s.CreateProtocol(agreeProtocolName, tree)
	if err != nil {
		return errors.New("Couldn't make new protocol: " + err.Error())
	}
	p := pi.(*protocol.BlsCosi)
	p.CreateProtocol = s.CreateProtocol
	p.Timeout = s.Timeout
	p.Msg = msg

	st := State{
		Signers:   getKeys(s.GetSigners(s.e).Set),
		HashPings: s.getHashPings(),
		Epoch:     s.e,
	}

	p.Data, err = protobuf.Encode(&st)
	if err != nil {
		return err
	}

	// Threshold before the subtrees so that we can optimize situation
	// like a threshold of one
	if s.Threshold > 0 {
		p.Threshold = s.Threshold
	}

	if s.NSubtrees > 0 {
		err = p.SetNbrSubTree(s.NSubtrees)
		if err != nil {
			p.Done()
			return err
		}
	}

	// start the protocol
	log.Lvl3("Cosi Service starting up root protocol")
	if err = p.Start(); err != nil {
		return err
	}
	// wait for reply. This will always eventually return.
	sig := <-p.FinalSignature

	if sig == nil {
		log.LLvl1(s.Name, s.PingDistances, s.getHashPings())
		return errors.New("Protocol output an empty signature")
	}

	res := protocol.BlsSignature(sig)
	publics := rooted.ServicePublics(ServiceName)

	return res.Verify(suite, msg, publics)
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

func (s *Service) getServers() map[string]*network.ServerIdentity {
	s.ServersMtx.Lock()
	dst := make(map[string]*network.ServerIdentity, len(s.Servers))

	for k, v := range s.Servers {
		dst[k] = v
	}
	s.ServersMtx.Unlock()
	return dst
}

// UpdateHistoryWith will send an ReqHistory to the service in parameter
func (s *Service) UpdateHistoryWith(name string) error {
	log.Lvl1("Updating ", s.Name, "with ", name)
	s.ServersMtx.Lock()
	si, ok := s.Servers[name]
	if !ok {
		return fmt.Errorf("%s is not aware of server named %s", s.ServerIdentity(), name)
	}
	s.ServersMtx.Unlock()
	s.DoneUpdate = false

	err := s.SendRaw(si, &ReqHistory{SenderIdentity: s.ServerIdentity()})

	// TODO : Fix race condition using Channel :
	// https://quii.gitbook.io/learn-go-with-tests/go-fundamentals/concurrency
	for !s.DoneUpdate {
		time.Sleep(50 * time.Millisecond)
	}

	return err

}

// ExecReqHistory will send back the node's version of history
func (s *Service) ExecReqHistory(env *network.Envelope) error {
	req, ok := env.Msg.(*ReqHistory)
	if !ok {
		log.Error(s.ServerIdentity(), "failed to cast to ReqHistory")
		return errors.New("failed to cast to ReqHistory")
	}

	s.storage.Lock()

	// Sending directely a []SignerSet is not working,
	// This solution flatten the data and reconstruct it afterwards
	// If protobuf.Encode is corrected it might not be needed anymore
	var signersKey []network.ServerIdentityID
	var signersValue []gpr.SignatureResponse
	var signersIndex []int

	for idx, signerMap := range s.storage.Signers {
		for k, v := range signerMap {
			signersIndex = append(signersIndex, idx)
			signersKey = append(signersKey, k)
			signersValue = append(signersValue, v)
		}
	}

	e := s.SendRaw(req.SenderIdentity, &ReplyHistory{
		SenderName:           s.Name,
		Servers:              s.Servers,
		ServerIdentityToName: s.ServerIdentityToName,
		SignersKey:           signersKey,
		SignersValue:         signersValue,
		SignersIndex:         signersIndex,
	})
	s.storage.Unlock()
	if e != nil {
		panic(e)
	}
	return e
}

// ExecReplyHistory will update the node's version of history based on the answer
// Assume nodes will not use that for malicious reasons
// No check for now
func (s *Service) ExecReplyHistory(env *network.Envelope) error {
	req, ok := env.Msg.(*ReplyHistory)
	if !ok {
		log.Error(s.ServerIdentity(), "failed to cast to ReplyHistory")
		return errors.New("failed to cast to ReplyHistory")
	}

	for k, v := range req.Servers {
		s.ServersMtx.Lock()
		s.Servers[k] = v
		s.ServersMtx.Unlock()
	}
	for k, v := range req.ServerIdentityToName {
		s.ServerIdentityToName[k] = v
	}

	// Reconstruction []SignerSet see ExecReqHistory
	signers := make([]SignersSet, req.SignersIndex[len(req.SignersIndex)-1]+1)

	for i := 0; i < len(req.SignersIndex); i++ {
		if len(signers[req.SignersIndex[i]]) == 0 {
			signers[req.SignersIndex[i]] = make(SignersSet)
		}
		signers[req.SignersIndex[i]][req.SignersKey[i]] = req.SignersValue[i]
	}

	s.storage.Lock()
	s.storage.Signers = signers
	l := len(s.storage.Signers)
	s.storage.Unlock()

	log.Lvl1(s.Name, "is now at Epoch", l-1)
	// Catching up on Epochs
	if s.e < Epoch(l-1) {
		s.e = Epoch(l - 1)
	}

	s.DoneUpdate = true
	return nil
}

// newService receives the context that holds information about the node it's
// running on. Saving and loading can be done using the context. The data will
// be stored in memory for tests and simulations, and on disk for real deployments.
func newService(c *onet.Context) (onet.Service, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	s := &Service{
		ServiceProcessor:     onet.NewServiceProcessor(c),
		Timeout:              protocolTimeout,
		suite:                suite,
		PrefixForReadingFile: dir + "/..",
		Servers:              make(map[string]*network.ServerIdentity),
		ServerIdentityToName: make(map[network.ServerIdentityID]string),
	}

	// Register function from one service to another
	s.RegisterProcessorFunc(execReqHistoryMsgID, s.ExecReqHistory)
	s.RegisterProcessorFunc(execReplyHistoryMsgID, s.ExecReplyHistory)
	s.RegisterProcessorFunc(execReqPingsMsgID, s.ExecReqPings)
	s.RegisterProcessorFunc(execReplyPingsMsgID, s.ExecReplyPings)

	// Register protocol (exec on tree)
	_, err = s.ProtocolRegister(gpr.Name, gpr.NewGossipProtocol(s.addSignerFromMessage))
	if err != nil {
		return nil, err
	}
	_, err = s.ProtocolRegister(agreeSubProtocolName, s.AgreeStateSubProtocol)
	if err != nil {
		return nil, err
	}
	_, err = s.ProtocolRegister(agreeProtocolName, AgreeStateProtocol)
	if err != nil {
		return nil, err
	}
	_, err = s.ProtocolRegister(getserversprotocol.Name, getserversprotocol.NewGetServersProtocol(s.getServers))
	if err != nil {
		return nil, err
	}

	if err = s.tryLoad(); err != nil {
		log.Error(err)
		return nil, err
	}
	return s, nil
}
