package membershipchainservice

import (
	"bufio"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
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

var WAITING_FOR_REGISTRATION = false

// ServiceName is used for registration on the onet.
const ServiceName = "MemberchainService"

// For Blscosi
const protocolTimeout = 20 * time.Minute

// MembershipID is used for tests
var MembershipID onet.ServiceID
var execReqHistoryMsgID network.MessageTypeID
var execReplyHistoryMsgID network.MessageTypeID
var suite = suites.MustFind("bn256.adapter").(*pairing.SuiteBn256)

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
	Nodes                gentree.LocalityNodes
	GraphTree            GraphTrees
	BinaryTree           map[string][]*onet.Tree
	alive                bool
	Distances            map[*gentree.LocalityNode]map[*gentree.LocalityNode]gentree.Compact
	PingDistances        map[string]map[string]float64
	ShortestDistances    map[string]map[string]float64
	OwnPings             map[string]float64
	DonePing             bool
	PingMapMtx           sync.Mutex
	PingAnswerMtx        sync.Mutex
	NrPingAnswers        int
	PrefixForReadingFile string
	PrefixForWritingFile string
	EpochChan            chan Epoch

	// From Interaction
	OwnInteractions      map[string]float64
	DoneInteraction      bool
	NrInteractionAnswers int
	CountInteractions    []map[string]int
	InteractionMtx       sync.Mutex

	//For Timing
	Cycle               Cycle
	LastEpochDur        time.Duration
	LastRegistrationDur time.Duration
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
	s.Cycle.Set()

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

	s.InteractionMtx.Lock()
	for _, name := range servers {
		s.CountInteractions[0][name]++
	}
	s.InteractionMtx.Unlock()

	s.storage.Lock()
	s.storage.Signers = append(s.storage.Signers, make(SignersSet))
	s.storage.Signers[0] = signers
	s.storage.Unlock()
	log.LLvl1(s.Name, "is set ..", s.Cycle)
}

// CreateProofForEpoch will get signatures from Signers from previous epoch
func (s *Service) CreateProofForEpoch(e Epoch) error {
	if s.Cycle.GetCurrentPhase() != REGISTRATION && WAITING_FOR_REGISTRATION {
		log.LLvl1(s.Name, "is waiting ", s.Cycle.GetTimeTillNextCycle(), "s to register")
		time.Sleep(s.Cycle.GetTimeTillNextCycle())
	}
	startTime := time.Now()

	log.Lvl1(s.Name, " is creating proof for Epoch : ", e)
	if s.e != e-1 {
		log.LLvl1(s.ServerIdentity(), "is having an error")
		return fmt.Errorf("Cannot register for epoch %d, as system is at epoch %d", e-1, s.e)
	}

	// Reset for the next epoch
	s.DoneInteraction = false

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

	buf, err := s.SignatureRequest(&SignatureRequest{Message: msg, Roster: ro, Epoch: e})
	if err != nil {
		return err
	}
	s.Proof = buf.(*gpr.SignatureResponse)
	if s.Proof == nil {
		log.LLvl1(s.Name, " :cannot share proof as it did not manage to get one")
		return fmt.Errorf("%v cannot share proof as it did not manage to get one", s.Name)
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	writeToFile(fmt.Sprintf("CreateProofForEpoch - 1 - SignatureRequest, %v, %v", int64(time.Now().Sub(startTime)/time.Millisecond), startTime), dir+"/"+s.PrefixForReadingFile+"/Data/Timing/"+s.Name+".txt")

	ro = ro.NewRosterWithRoot(s.ServerIdentity())
	s.CountTwoMessagesPerNodesInRoster(ro)
	writeToFile(s.Name+",CreateProofForEpoch,"+strconv.Itoa(len(ro.List))+","+strconv.Itoa(int(s.e)), "Data/messages.txt")
	// Share first to the old signers. That way they will have a view of the global system that they can transmit to the others
	tree := ro.GenerateNaryTree(len(mbrs))
	pi, err := s.CreateProtocol(gpr.Name, tree)
	if err != nil {
		return errors.New("Couldn't make new protocol: " + err.Error())
	}

	const baseCommunication = 200 * time.Millisecond
	nbNodes := len(ro.List)

	gossipTimeOut := baseCommunication * time.Duration(nbNodes) * 2
	gossipTimeOut = s.Cycle.GetTimeTillNextEpoch() / 2

	p := pi.(*gpr.GossipRegistationProtocol)
	p.TimeOut = gossipTimeOut
	p.Msg = gpr.Announce{
		Name:   s.Name,
		Server: s.ServerIdentity(),
		Signer: s.ServerIdentity().ID,
		Proof:  s.Proof,
		Epoch:  int(s.e + 1),
	}

	p.Start()

	select {
	case numConf := <-p.ConfirmationsChan:
		log.LLvl1("\033[51;5;33m", s.Name, "'s Registration took", time.Now().Sub(startTime), "Recieved", numConf, "/", nbNodes, " Confirmations\033[0m")
		s.LastRegistrationDur = time.Now().Sub(startTime)
		writeToFile(fmt.Sprintf("CreateProofForEpoch - 2 - Gossip, %v, %v", int64(time.Now().Sub(startTime)/time.Millisecond), startTime), dir+"/"+s.PrefixForReadingFile+"/Data/Timing/"+s.Name+".txt")
		if numConf != nbNodes {
			return fmt.Errorf("Create Proof recieved only %v confirmations out of %v", numConf, nbNodes)
		}
		return nil
	case <-time.After(gossipTimeOut * 2):
		log.LLvl1(s.Name, " got a TimeOut in the Gossip Protocol")
		writeToFile(fmt.Sprintf("CreateProofForEpoch - 2 - Gossip TimeOut, %v, %v", int64(time.Now().Sub(startTime)/time.Millisecond), startTime), dir+"/"+s.PrefixForReadingFile+"/Data/Timing/"+s.Name+".txt")
		return fmt.Errorf("%v got a TimeOut in the Gossip Protocol", s.Name)
	}

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
	startTime := time.Now()
	s.e = <-s.EpochChan
	s.InteractionMtx.Lock()
	s.CountInteractions = append(s.CountInteractions, make(map[string]int))
	s.InteractionMtx.Unlock()

	log.Lvl1("\033[48;5;33m", s.Name, " Starts Epoch ", s.e, " Successfully. It took", time.Now().Sub(startTime), "\033[0m")
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	writeToFile(fmt.Sprintf("StartNewEpoch - 1 - Recieve List, %v, %v", int64(time.Now().Sub(startTime)/time.Millisecond), startTime), dir+"/"+s.PrefixForReadingFile+"/Data/Timing/"+s.Name+".txt")

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
	writeToFile(fmt.Sprintf("StartNewEpoch - 2 - After Setup, %v, %v", int64(time.Now().Sub(startTime)/time.Millisecond), startTime), dir+"/"+s.PrefixForReadingFile+"/Data/Timing/"+s.Name+".txt")
	if s.Name == "node_0" {
		writeToFile(fmt.Sprintf("%v", s.GraphTree), "Data/maps_graphTree_"+s.Name+"_epoch"+strconv.Itoa(int(s.e))+".txt")
	}

	randSrc := rand.New(rand.NewSource(10))
	leaderID := randSrc.Intn(len(ro.List))

	if s.Name == "node_"+strconv.Itoa(leaderID) {
		// Wait that all the other services have set up.
		time.Sleep(1 * time.Second)
		log.Lvl1("\033[78;5;33m", s.Name, " STARTED AGREEING ON STATE. It took", time.Now().Sub(startTime), " \033[0m")
		_, err = s.AgreeOnState(ro, PINGSMSG)
		if err != nil {
			log.LLvl1("\033[39;5;1m", s.Name, " is not passing the PINGS Agree, Error :   ", err, " \033[0m")
			return err
		}
	}
	log.Lvl1("\033[48;5;33m", s.Name, " Finished Epoch ", s.e, " Successfully. It took", time.Now().Sub(startTime), " \033[0m")
	writeToFile(fmt.Sprintf("StartNewEpoch - 2 - End Epoch, %v, %v", int64(time.Now().Sub(startTime)/time.Millisecond), startTime), dir+"/"+s.PrefixForReadingFile+"/Data/Timing/"+s.Name+".txt")
	s.LastEpochDur = time.Now().Sub(startTime)
	return err
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
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	s := &Service{
		ServiceProcessor:     onet.NewServiceProcessor(c),
		Timeout:              protocolTimeout,
		suite:                suite,
		PrefixForReadingFile: dir + "/..",
		PrefixForWritingFile: "",
		Servers:              make(map[string]*network.ServerIdentity),
		ServerIdentityToName: make(map[network.ServerIdentityID]string),
		CountInteractions:    make([]map[string]int, 0),
		OwnInteractions:      make(map[string]float64),
		PingDistances:        make(map[string]map[string]float64),
		// TODO : invesitgate why is blocking with 1
		// One reason could be that, for one node it get two value, (one for the update one for Consensus)
		// But that sould not happen
		EpochChan: make(chan Epoch, 3),
	}
	s.Cycle.Set()
	log.ErrFatal(s.RegisterHandlers(s.SetGenesisSignersRequest, s.ExecEpochRequest, s.ExecWriteSigners, s.ExecSetDuration, s.ExecUpdateForNewNode, s.ExecCreateProofForEpoch, s.ExecUpdate, s.ExecStartNewEpoch, s.ExecGetConsencusOnNewSigners, s.ExecPause))

	// Register function from one service to another
	s.RegisterProcessorFunc(execReqHistoryMsgID, s.ExecReqHistory)
	s.RegisterProcessorFunc(execReplyHistoryMsgID, s.ExecReplyHistory)
	s.RegisterProcessorFunc(execReqPingsMsgID, s.ExecReqPings)
	s.RegisterProcessorFunc(execReplyPingsMsgID, s.ExecReplyPings)
	s.RegisterProcessorFunc(execReqInteractionsMsgID, s.ExecReqInteractions)
	s.RegisterProcessorFunc(execReplyInteractionsMsgID, s.ExecReplyInteractions)

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

	if err = s.tryLoad(); err != nil {
		log.Error(err)
		return nil, err
	}

	s.CountInteractions = append(s.CountInteractions, make(map[string]int))
	s.e = 0

	return s, nil
}
