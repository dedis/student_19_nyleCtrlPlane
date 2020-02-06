package membershipchainservice

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	gpr "github.com/dedis/student_19_nyleCtrlPlane/gossipregistrationprotocol"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

// UpdateHistoryWith will send an ReqHistory to the service in parameter
func (s *Service) UpdateHistoryWith(name string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	log.Lvl3("Updating ", s.Name, "with ", name)
	startTime := time.Now()
	s.ServersMtx.Lock()
	si, ok := s.Servers[name]
	if !ok {
		return fmt.Errorf("%s is not aware of server named %s", s.ServerIdentity(), name)
	}
	s.ServersMtx.Unlock()
	writeToFile(fmt.Sprintf("UpdateHistoryWith - 1 - ServerLock, %v, %v", int64(time.Now().Sub(startTime)/time.Millisecond), startTime), dir+"/"+s.PrefixForReadingFile+"/Data/Timing/"+s.Name+".txt")

	err = s.SendRaw(si, &ReqHistory{SenderIdentity: s.ServerIdentity()})
	sendHistoryTimeOut := 10 * time.Second
	select {
	case s.e = <-s.EpochChan:
		writeToFile(fmt.Sprintf("UpdateHistoryWith - 2 - Answer, %v, %v", int64(time.Now().Sub(startTime)/time.Millisecond), startTime), dir+"/"+s.PrefixForReadingFile+"/Data/Timing/"+s.Name+".txt")
		s.InteractionMtx.Lock()
		if len(s.CountInteractions) <= int(s.GetEpoch()) {
			// TODO : refactor
			for len(s.CountInteractions) <= int(s.GetEpoch()) {
				s.CountInteractions = append(s.CountInteractions, make(map[string]int))
			}
		}
		s.CountInteractions[s.GetEpoch()][name]++
		s.InteractionMtx.Unlock()
		writeToFile(fmt.Sprintf("UpdateHistoryWith - 3 - Answer Processed, %v, %v", int64(time.Now().Sub(startTime)/time.Millisecond), startTime), dir+"/"+s.PrefixForReadingFile+"/Data/Timing/"+s.Name+".txt")
		writeToFile(s.Name+",UpdateHistoryWith, 1"+","+strconv.Itoa(int(s.e)), "Data/messages.txt")
		return err
	case <-time.After(sendHistoryTimeOut):
		writeToFile(fmt.Sprintf("UpdateHistoryWith - 2 - CHURNED, %v, %v", int64(time.Now().Sub(startTime)/time.Millisecond), startTime), dir+"/"+s.PrefixForReadingFile+"/Data/Timing/"+s.Name+".txt")
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

	if name, ok := s.ServerIdentityToName[si.ID]; ok {
		log.LLvl1(s.Name, "-", s.ServerIdentity(), " is sending History to ", si, "-", name)
	} else {
		log.LLvl1(s.Name, "-", s.ServerIdentity(), " is sending History to new SI :", si)
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
	//log.LLvl1(s.Name, "Finish sending to ", si, "with error : ", e)
	return e

}

// ExecReqHistory will send back the node's version of history
func (s *Service) ExecReqHistory(env *network.Envelope) error {
	req, ok := env.Msg.(*ReqHistory)
	if !ok {
		log.Error(s.ServerIdentity(), "failed to cast to ReqHistory")
		return errors.New("failed to cast to ReqHistory")
	}
	s.InteractionMtx.Lock()
	s.CountInteractions[s.GetEpoch()][req.SenderName] += 2
	s.InteractionMtx.Unlock()

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

	s.InteractionMtx.Lock()
	if len(s.CountInteractions) != 0 && s.GetEpoch() != 0 {
		s.CountInteractions[s.GetEpoch()-1][req.SenderName]++
	}
	s.InteractionMtx.Unlock()

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

	log.LLvl1("\033[51;5;33m", s.Name, "is done updating. It remains ", s.Cycle.GetTimeTillNextEpoch(), " before the next Epoch \033[0m")
	return nil
}
