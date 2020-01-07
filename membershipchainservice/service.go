package membershipchainservice

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"
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

// MembershipID is used for tests
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
	GraphTree         GraphTrees
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
	EpochChan            chan Epoch
}

// storageID reflects the data we're storing - we could store more
// than one structure.
var storageID = []byte("main")

// storage is used to save our data.
type storage struct {
	Signers []SignersSet
	sync.Mutex
}

// SetGenesisSignersRequest handles requests for the function
func (s *Service) SetGenesisSignersRequest(req *SetGenesisSignersRequest) (*SetGenesisSignersReply, error) {
	s.SetGenesisSigners(req.Servers)
	return &SetGenesisSignersReply{}, nil
}

//ExecEpochRequest handles requests for the function
func (s *Service) ExecEpochRequest(req *ExecEpochRequest) (*ExecEpochReply, error) {
	var err error
	if s.e != req.Epoch-1 {
		err = s.UpdateHistoryWith(s.GetRandomName())
		if err != nil {
			return nil, err
		}
	}

	err = s.CreateProofForEpoch(req.Epoch)
	if err != nil {
		return nil, err
	}

	// TODO change with a random leader
	if s.Name == "node_0" {
		err = s.GetConsencusOnNewSigners()
		if err != nil {
			return nil, err
		}
	}
	err = s.StartNewEpoch()
	if err != nil {
		return nil, err
	}
	log.LLvl1("PASSS: ", s.Name, "is ending epoch", s.e)

	return &ExecEpochReply{}, nil
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
	log.LLvl1(s.Name, "is set ..")
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

func (s *Service) getRosterForEpoch(e Epoch) (*onet.Roster, error) {
	s.storage.Lock()
	mbrs, err := s.getServerIdentityFromSignersSet(s.storage.Signers[s.e])
	if err != nil {
		defer s.storage.Unlock()
		return nil, err
	}
	if _, ok := s.storage.Signers[s.e][s.ServerIdentity().ID]; !ok {
		defer s.storage.Unlock()
		return nil, errors.New("One node cannot start a new Epoch if it didn't registrate")
	}
	s.storage.Unlock()

	return onet.NewRoster(mbrs), nil

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

	log.Lvl1(s.Name, " is creating proof for Epoch : ", e)
	if s.e != e-1 {
		log.LLvl1(s.ServerIdentity(), "is having an error")
		return fmt.Errorf("Cannot register for epoch %d, as system is at epoch %d", e-1, s.e)
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

	ro = ro.NewRosterWithRoot(s.ServerIdentity())

	writeToFile(s.Name+",CreateProofForEpoch,"+strconv.Itoa(len(ro.List))+","+strconv.Itoa(int(s.e)), "Data/messages.txt")
	// Share first to the old signers. That way they will have a view of the global system that they can transmit to the others
	tree := ro.GenerateNaryTree(len(mbrs))
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

	gossipTimeOut := 2 * time.Second

	select {
	case <-p.ConfirmationsChan:
		return nil
	case <-time.After(gossipTimeOut):
		p.Shutdown()
		p.Done()
		return fmt.Errorf("%v got a TimeOut in the Gossip Protocol", s.Name)
	}
}

// GetConsencusOnNewSigners is run by the previous commitee, the signed result is sent to the new nodes.
func (s *Service) GetConsencusOnNewSigners() error {

	if s.Cycle.GetCurrentPhase() != EPOCH {
		log.LLvl1(s.Name, "is waiting ", s.Cycle.GetTimeTillNextEpoch()-TIME_FOR_CONSENCUS, "s to Get the Consencus")
		time.Sleep(s.Cycle.GetTimeTillNextEpoch() - TIME_FOR_CONSENCUS)
	}
	timeCons := time.Now()
	log.Lvl1("\033[48;5;33m", s.Name, " Starts Consensus after", time.Now().Sub(timeCons), " \033[0m")
	ro, err := s.getRosterForEpoch(s.e)
	if err != nil {
		return err
	}
	log.Lvl1("\033[48;5;33m", s.Name, " Starts Agree on State after", time.Now().Sub(timeCons), " \033[0m")
	// Agree on Signers
	_, err = s.AgreeOnState(ro, SIGNERSMSG)
	if err != nil {
		log.LLvl1(" \033[38;5;1m", s.Name, " is not passing the Signers Agree, Error :   ", err, " \033[0m")
		return err
	}

	//log.LLvl1("Send Signature after", time.Now().Sub(timeCons), sign)
	newSigners := s.GetSigners(s.e + 1)

	var siList []*network.ServerIdentity
	s.ServersMtx.Lock()
	for sID := range newSigners.Set {
		if sID != s.ServerIdentity().ID {
			name := s.ServerIdentityToName[sID]
			siList = append(siList, s.Servers[name])
		}
	}
	s.ServersMtx.Unlock()

	writeToFile(s.Name+",GetConsencusOnNewSigners,"+strconv.Itoa(len(siList))+","+strconv.Itoa(int(s.e)), "Data/messages.txt")

	for _, si := range siList {
		//log.LLvl1("Send History to ", si, " after", time.Now().Sub(timeCons))
		go func(SI *network.ServerIdentity) {
			panicErr := s.SendHistory(SI)
			if panicErr != nil {
				panic(panicErr)
			}
		}(si)
	}
	s.EpochChan <- s.e + 1
	return nil
}

// StartNewEpoch stops the registration for nodes and run CRUX
func (s *Service) StartNewEpoch() error {
	if s.Cycle.GetCurrentPhase() != EPOCH {
		log.LLvl1(s.Name, "is waiting ", s.Cycle.GetTimeTillNextEpoch(), "s to start the new Epoch")
		time.Sleep(s.Cycle.GetTimeTillNextEpoch())
	}
	if s.e != s.Cycle.GetEpoch() {
		log.Lvl1("\033[48;5;1m", s.Name, " Does not start Epoch ", s.e, " as the clock says it is ", s.Cycle.GetEpoch(), ".\033[0m")
		return fmt.Errorf("%s : Its not the time for epoch %d. The clock says its %d", s.Name, s.e, s.Cycle.GetEpoch())
	}

	s.e = <-s.EpochChan

	log.Lvl1("\033[48;5;33m", s.Name, " Starts Epoch ", s.e, " Successfully.\033[0m")

	ro, err := s.getRosterForEpoch(s.e)
	if err != nil {
		return err
	}

	file, _ := os.OpenFile("Data/members.txt", os.O_RDWR|os.O_CREATE, 0660)
	w := bufio.NewWriter(file)

	if s.Name == "node_0" {
		w.WriteString("Name,Address")
		w.WriteString("\n")
	}
	si2name := make(map[*network.ServerIdentity]string)
	for _, serv := range ro.List {
		si2name[serv] = s.ServerIdentityToName[serv.ID]
		if s.Name == "node_0" {
			w.WriteString(s.ServerIdentityToName[serv.ID] + "," + fmt.Sprintf("%v", serv.Address) + "\n")
		}

	}
	w.Flush()
	file.Close()

	s.Setup(&InitRequest{
		ServerIdentityToName: si2name,
	})

	writeToFile(s.Name+",Pings,"+getMemoryUsage(s.PingDistances)+","+strconv.Itoa(int(s.e)), "Data/storage.txt")
	if s.Name == "node_0" {
		writeToFile(fmt.Sprintf("%v", s.GraphTree), "Data/maps_graphTree_"+s.Name+"_epoch"+strconv.Itoa(int(s.e))+".txt")
	}
	// Wait that all the other services have set up.
	time.Sleep(1 * time.Second)
	_, err = s.AgreeOnState(ro, PINGSMSG)
	if err != nil {
		log.LLvl1("\033[39;5;1m", s.Name, " is not passing the PINGS Agree, Error :   ", err, " \033[0m")
		return err
	}
	log.Lvl1("\033[48;5;33m", s.Name, " Finished Epoch ", s.e, " Successfully.\033[0m")
	return err
}

// AgreeOnState checks that the members of the roster have the same signers + same maps
func (s *Service) AgreeOnState(roster *onet.Roster, msg []byte) (protocol.BlsSignature, error) {
	// generate the tree
	nNodes := len(roster.List)
	rooted := roster.NewRosterWithRoot(s.ServerIdentity())
	if rooted == nil {
		return nil, errors.New("we're not in the roster")
	}
	tree := rooted.GenerateNaryTree(nNodes)
	if tree == nil {
		return nil, errors.New("failed to generate tree")
	}

	writeToFile(s.Name+",AgreeOnState,"+strconv.Itoa(nNodes)+","+strconv.Itoa(int(s.e)), "Data/messages.txt")

	// configure the BlsCosi protocol
	pi, err := s.CreateProtocol(agreeProtocolName, tree)
	if err != nil {
		return nil, errors.New("Couldn't make new protocol: " + err.Error())
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
		return nil, err
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

	if sig == nil {
		log.LLvl1(s.Name, s.PingDistances, s.getHashPings())
		return nil, errors.New("Protocol output an empty signature")
	}

	res := protocol.BlsSignature(sig)
	publics := rooted.ServicePublics(ServiceName)

	return res, res.Verify(suite, msg, publics)
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

func (s *Service) getepochOfEntryMap() map[string]Epoch {
	temp := make(map[network.ServerIdentityID]Epoch)
	s.storage.Lock()
	for i := len(s.storage.Signers) - 1; i >= 0; i-- {
		for id := range s.storage.Signers[i] {
			temp[id] = Epoch(i)
		}
	}
	s.storage.Unlock()

	ret := make(map[string]Epoch)
	s.ServersMtx.Lock()
	for id, e := range temp {
		ret[s.ServerIdentityToName[id]] = e
	}
	s.ServersMtx.Unlock()
	return ret
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

	err := s.SendRaw(si, &ReqHistory{SenderIdentity: s.ServerIdentity()})
	sendHistoryTimeOut := 2 * time.Second
	select {
	case s.e = <-s.EpochChan:
		writeToFile(s.Name+",UpdateHistoryWith, 1"+","+strconv.Itoa(int(s.e)), "Data/messages.txt")
		return err
	case <-time.After(sendHistoryTimeOut):
		newName := "node_0"
		if s.Name == "node_0" {
			newName = "node_1"
		}
		log.LLvl1(name, "HAS CHURN AFTER REQUEST OF ", s.Name, " UPDATING WITH ", newName, " ------------@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@")
		err := s.UpdateHistoryWith(newName)
		return err
	}
}

// SendHistory send my version of History to the given SI
func (s *Service) SendHistory(si *network.ServerIdentity) error {
	if s.ServerIdentity().ID == si.ID {
		return fmt.Errorf("%v is asked to send History to itself", s.Name)
	}

	log.LLvl1(s.ServerIdentity(), " is sending History to ", si)

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
	s.storage.Unlock()

	s.ServersMtx.Lock()
	e := s.SendRaw(si, &ReplyHistory{
		SenderName:           s.Name,
		Servers:              s.Servers,
		ServerIdentityToName: s.ServerIdentityToName,
		SignersKey:           signersKey,
		SignersValue:         signersValue,
		SignersIndex:         signersIndex,
	})
	s.ServersMtx.Unlock()

	if e != nil {
		panic(e)
	}
	return e

}

// ExecReqHistory will send back the node's version of history
func (s *Service) ExecReqHistory(env *network.Envelope) error {
	req, ok := env.Msg.(*ReqHistory)
	if !ok {
		log.Error(s.ServerIdentity(), "failed to cast to ReqHistory")
		return errors.New("failed to cast to ReqHistory")
	}
	return s.SendHistory(req.SenderIdentity)
}

// ExecReplyHistory will update the node's version of history based on the answer
// Assume nodes will not use that for malicious reasons
// No check for now
func (s *Service) ExecReplyHistory(env *network.Envelope) error {
	log.LLvl1(s.Name, " is executing history.")
	req, ok := env.Msg.(*ReplyHistory)
	if !ok {
		log.Error(s.ServerIdentity(), "failed to cast to ReplyHistory")
		return errors.New("failed to cast to ReplyHistory")
	}
	s.ServersMtx.Lock()
	for k, v := range req.Servers {
		s.Servers[k] = v
	}
	for k, v := range req.ServerIdentityToName {
		s.ServerIdentityToName[k] = v
	}
	s.ServersMtx.Unlock()
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
		s.EpochChan <- Epoch(l - 1)
	} else {
		s.EpochChan <- s.e
	}

	log.LLvl1(s.Name, "is done updating.")
	return nil
}

// GetRandomName get a random name from the list of Server
func (s *Service) GetRandomName() string {
	var names []string
	s.ServersMtx.Lock()
	for name := range s.Servers {
		if name != s.Name {
			names = append(names, name)
		}
	}
	log.LLvl1(s.Name, "has a this list ", names, " of random names.")

	s.ServersMtx.Unlock()
	index := rand.Intn(len(names))
	return names[index]
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
		// TODO : invesitgate why is blocking with 1
		// One reason could be that, for one node it get two value, (one for the update one for Consensus)
		// But that sould not happen
		EpochChan: make(chan Epoch, 2),
	}
	log.ErrFatal(s.RegisterHandlers(s.SetGenesisSignersRequest, s.ExecEpochRequest))

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
