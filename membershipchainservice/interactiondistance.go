package membershipchainservice

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

var execReqInteractionsMsgID network.MessageTypeID
var execReplyInteractionsMsgID network.MessageTypeID

func init() {
	execReqInteractionsMsgID = network.RegisterMessage(&ReqInteractions{})
	execReplyInteractionsMsgID = network.RegisterMessage(&ReplyInteractions{})
}

// GetPingDistances return the mutex protected map PingDistances for test purposes
func (s *Service) GetPingDistances() map[string]map[string]float64 {
	s.PingMapMtx.Lock()
	defer s.PingMapMtx.Unlock()
	return s.PingDistances
}

func (s *Service) countOwnInteraction() {
	s.InteractionMtx.Lock()
	str := s.Name + ":    - "
	for _, node := range s.Nodes.All {
		peerName := node.Name
		if peerName != s.Name {
			str += node.Name + ","
			if _, ok := s.CountInteractions[s.GetEpoch()-1][peerName]; !ok {
				s.CountInteractions[s.GetEpoch()-1][peerName] = 1
			}
			s.OwnInteractions[peerName] = 1.0 / float64(s.CountInteractions[s.GetEpoch()-1][peerName]) * 1000
		}
	}
	s.OwnInteractions[s.Name] = 0
	str += s.Name + ","
	s.InteractionMtx.Unlock()
	//log.LLvl1(str)
}

// GetInteractionDistances measure the intraction distance between the services and all the other nodes
// Then it communicates the results with the other nodes
func (s *Service) GetInteractionDistances() {
	s.countOwnInteraction()
	s.DoneInteraction = true

	s.PingMapMtx.Lock()
	for name, dist := range s.OwnInteractions {
		src := s.Nodes.GetServerIdentityToName(s.ServerIdentity())
		dst := name

		if _, ok := s.PingDistances[src]; !ok {
			s.PingDistances[src] = make(map[string]float64)
		}

		s.PingDistances[src][dst] = dist
		s.PingDistances[src][src] = 0.0
	}
	s.PingMapMtx.Unlock()

	// Count sending
	s.InteractionMtx.Lock()
	for _, n := range s.Nodes.All {
		s.CountInteractions[s.GetEpoch()][n.Name]++
	}
	s.InteractionMtx.Unlock()

	// ask for Interaction from others
	for _, node := range s.Nodes.All {
		if s.Name != node.Name {
			e := s.SendRaw(node.ServerIdentity, &ReqInteractions{SenderName: s.Name})
			if e != nil {
				log.LLvl1("\033[94m Error ? : ", e, "\033[39m ")
				panic(e)
			}
		}
	}

	// wait for ping replies from everyone but myself
	for s.NrInteractionAnswers != len(s.Nodes.All)-1 {
		log.LLvl1(s.Name, " \033[32m  is WAITING ------------------------------------------ ", s.NrInteractionAnswers, len(s.Nodes.All)-1, "\033[39m ")
		time.Sleep(100 * time.Millisecond)
	}
	s.NrInteractionAnswers = 0

	// check that there are enough pings
	if len(s.PingDistances) < len(s.Nodes.All) {
		log.Lvl1(s.Nodes.GetServerIdentityToName(s.ServerIdentity()), " too few pings 1", len(s.PingDistances), len(s.Nodes.All))

	}
	for nodeName, m := range s.PingDistances {
		if len(m) < len(s.Nodes.All) {
			log.Lvl1(s.Nodes.GetServerIdentityToName(s.ServerIdentity()), " too few pings 2", len(m), len(s.Nodes.All))
			log.LLvl1(s.Nodes.GetServerIdentityToName(s.ServerIdentity()), nodeName, m)
		}
	}

	file, _ := os.Create("Data/SpaceTime/Distances-" + s.Nodes.GetServerIdentityToName(s.ServerIdentity()) + "-epoch" + strconv.Itoa(int(s.e)))
	w := bufio.NewWriter(file)
	w.WriteString("Name1,Name2,Distances\n")
	for n1, m := range s.PingDistances {
		for n2, d := range m {
			w.WriteString(n1 + "," + n2 + "," + fmt.Sprintf("%.2f", d) + "\n")
		}
	}
	w.Flush()
	file.Close()

}

// ReqInteractions is use to request pings
type ReqInteractions struct {
	SenderName string
}

// ReplyInteractions hold the reply form the ping request
type ReplyInteractions struct {
	Interactions string
	SenderName   string
}

// ExecReqInteractions is executed if the service recieve a ReqInteraction
// It is registered in Service.go
func (s *Service) ExecReqInteractions(env *network.Envelope) error {
	// Parse message
	req, ok := env.Msg.(*ReqInteractions)
	if !ok {
		log.Error(s.ServerIdentity(), "failed to cast to ReqInteractions")
		return errors.New("failed to cast to ReqInteractions")
	}

	// wait for pings to be finished
	for !s.DoneInteraction {
		log.LLvl1(s.Name, "is waiting for interactions to answer to", req.SenderName)
		time.Sleep(50 * time.Millisecond)

	}

	reply := ""
	myName := s.Nodes.GetServerIdentityToName(s.ServerIdentity())
	// build reply
	for peerName, pingTime := range s.OwnInteractions {
		reply += myName + " " + peerName + " " + fmt.Sprintf("%f", pingTime) + "\n"
	}
	//log.Lvl1(s.Name, req.SenderName, s.Nodes.All)
	requesterIdentity := s.Nodes.GetByName(req.SenderName).ServerIdentity

	// I recieve a request and I answer
	s.InteractionMtx.Lock()
	s.CountInteractions[s.GetEpoch()][req.SenderName] += 2
	s.InteractionMtx.Unlock()
	e := s.SendRaw(requesterIdentity, &ReplyInteractions{Interactions: reply, SenderName: myName})
	if e != nil {
		panic(e)
	}
	return e
}

// ExecReplyInteractions is executed if the service recieve a ReplyInteraction
func (s *Service) ExecReplyInteractions(env *network.Envelope) error {
	// Parse message
	req, ok := env.Msg.(*ReplyInteractions)
	if !ok {
		log.Error(s.ServerIdentity(), "failed to cast to ReplyInteractions")
		return errors.New("failed to cast to ReplyInteractions")
	}

	s.InteractionMtx.Lock()
	s.CountInteractions[s.GetEpoch()][req.SenderName]++
	s.InteractionMtx.Unlock()

	s.PingMapMtx.Lock()

	//log.LLvl1(s.Name, "\n_ _ _ _ _ _ _ _ _ _ _ _ _ \n", req.Interactions)

	lines := strings.Split(req.Interactions, "\n")
	for _, line := range lines {
		if line != "" {
			words := strings.Split(line, " ")
			src := words[0]
			dst := words[1]
			pingRes, err := strconv.ParseFloat(words[2], 64)
			if err != nil {
				log.Error("Problem when parsing pings")
			}

			if _, ok := s.PingDistances[src]; !ok {
				s.PingDistances[src] = make(map[string]float64)
			}

			s.PingDistances[src][dst] += pingRes
			s.PingDistances[src][src] = 0.0

		}
	}
	s.PingMapMtx.Unlock()

	s.PingAnswerMtx.Lock()
	s.NrInteractionAnswers++
	s.PingAnswerMtx.Unlock()

	return nil
}

// CountTwoMessagesPerNodesInRoster will add two interactions (send recieve) by nodes in the roster
// It is an approximation as node can churn, however digging in the protocol to anaylse which nodes fails
// was too complex
func (s *Service) CountTwoMessagesPerNodesInRoster(ro *onet.Roster) {
	listOfNames := make([]string, len(ro.List))
	s.ServersMtx.Lock()
	for _, n := range ro.List {
		listOfNames = append(listOfNames, s.ServerIdentityToName[n.ID])
	}
	s.ServersMtx.Unlock()
	s.InteractionMtx.Lock()
	for _, n := range listOfNames {
		s.CountInteractions[s.GetEpoch()][n] += 2
	}
	s.InteractionMtx.Unlock()
}
