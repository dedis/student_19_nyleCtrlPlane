package membershipchainservice

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"go.dedis.ch/cothority/v3/blscosi/protocol"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"go.dedis.ch/protobuf"
)

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

	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	writeToFile(s.Name+",GetConsencusOnNewSigners,"+strconv.Itoa(len(siList))+","+strconv.Itoa(int(s.e)), "Data/messages.txt")
	writeToFile(fmt.Sprintf("GetConsencusOnNewSigners - 1 - Consensus, %v, %v", int64(time.Now().Sub(timeCons)/time.Millisecond), timeCons), dir+"/"+s.PrefixForReadingFile+"/Data/Timing/"+s.Name+".txt")
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
	writeToFile(fmt.Sprintf("GetConsencusOnNewSigners - 2 - After Send, %v, %v", int64(time.Now().Sub(timeCons)/time.Millisecond), timeCons), dir+"/"+s.PrefixForReadingFile+"/Data/Timing/"+s.Name+".txt")
	return nil
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

	s.CountTwoMessagesPerNodesInRoster(rooted)
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
