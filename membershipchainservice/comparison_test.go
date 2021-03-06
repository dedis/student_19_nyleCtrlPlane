package membershipchainservice

import (
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

func TestNodesWantingToJoin(t *testing.T) {
	t.Skip("Skipping Joining Test as sometimes it fails for time reasons")
	WAITING_FOR_REGISTRATION = false
	nbrNodes := 20
	nbrEpoch := Epoch(10)
	nbFirstSigners := 4
	rmFile("Data/messages.txt")
	rmFile("Data/storage.txt")
	writeToFile("Name,Function,nb_messages,epoch", "Data/messages.txt")
	writeToFile("Name,Function,storage,epoch", "Data/storage.txt")

	stillOut := make(map[int]bool, nbrNodes)
	alreadyIn := make(map[int]bool, nbrNodes)
	alreadyWritten := make(map[int]bool, nbrNodes)

	local := onet.NewTCPTest(tSuite)

	hosts, _, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()
	services := local.GetServices(hosts, MembershipID)
	for i, s := range services {
		s.(*Service).Name = "node_" + strconv.Itoa(i)
		stillOut[i] = true
		alreadyIn[i] = false
		alreadyWritten[i] = false
	}

	servers := make(map[*network.ServerIdentity]string)

	oldCommittee := make([]int, 0)
	for i := 0; i < nbFirstSigners; i++ {
		servers[hosts[i].ServerIdentity] = services[i].(*Service).Name
		log.LLvl1("Signers 0 : ", hosts[i].ServerIdentity)
		alreadyIn[i] = true
		alreadyWritten[i] = true
		stillOut[i] = false
		oldCommittee = append(oldCommittee, i)
	}

	var wg sync.WaitGroup

	for _, s := range services {
		wg.Add(1)
		go func(serv *Service) {
			serv.SetGenesisSigners(servers)
			wg.Done()
		}(s.(*Service))

	}
	wg.Wait()
	rmFile("Data/comparison_join.txt")
	writeToFile("Name,Registration,Time,epoch", "Data/comparison_join.txt")
	for i, b := range alreadyIn {
		if b {
			writeToFile(fmt.Sprintf("%v,Manage Normally,%v,%v", services[i].(*Service).Name, 0, 0), "Data/comparison_join.txt")
		}
	}
	startTime := time.Now()

	for e := Epoch(1); e <= nbrEpoch; e++ {

		log.LLvl1("\033[48;5;42mStart of Epoch ", e, " after:  ", int64(time.Now().Sub(startTime)/time.Millisecond), "\033[0m ")

		for i, b := range alreadyIn {
			if b {
				log.LLvl1("Service : ", services[i].(*Service).Name, " : ", services[i].(*Service).ServerIdentity())
				if i == 0 {
					writeToFile(fmt.Sprintf("node_0,starts epoch,%v,%v", int64(time.Now().Sub(startTime)/time.Millisecond), e), "Data/comparison_join.txt")
				}
			}
		}

		log.LLvl1("\033[48;5;43mRegistration : ", e, " for ", alreadyIn, " nodes\033[0m ")

		// Update for new nodes.
		for i, b := range alreadyIn {
			if b {
				wg.Add(1)
				go func(idx int) {
					s := services[idx].(*Service)

					ro, err := s.getRosterForEpoch(e)

					if s.GetEpoch() != e-1 || err != nil || ro == nil {
						log.LLvl1(s.Name, "is trying to update")
						name := s.GetRandomName()
						assert.NoError(t, s.UpdateHistoryWith(name))

					}
					if s.Cycle.GetCurrentPhase() != REGISTRATION {
						log.LLvl1(s.Name, "is waiting ", s.Cycle.GetTimeTillNextCycle(), "s to register")
						time.Sleep(s.Cycle.GetTimeTillNextCycle() + 100*time.Millisecond)
					}
					wg.Done()
				}(i)
			}
		}
		wg.Wait()
		// Registration
		for i, b := range alreadyIn {
			if b {
				wg.Add(1)
				go func(idx int) {
					writeToFile(fmt.Sprintf("%v,Wants As Usual,%v,%v", services[idx].(*Service).Name, int64(time.Now().Sub(startTime)/time.Millisecond), e), "Data/comparison_join.txt")
					err := services[idx].(*Service).CreateProofForEpoch(e)
					log.LLvl1("Error in create proof", err)
					if !alreadyWritten[idx] {
						writeToFile(fmt.Sprintf("%v,Manage Normally,%v,%v", services[idx].(*Service).Name, int64(time.Now().Sub(startTime)/time.Millisecond), e), "Data/comparison_join.txt")
						alreadyWritten[idx] = true
					}
					wg.Done()
				}(i)
			}
		}
		// Random registration :
		var nbNewNodes int
		nbNewNodes = rand.Intn(5)
		nbStillLeft := 0
		listStillLeft := make([]int, nbStillLeft)
		for i, b := range stillOut {
			if b {
				nbStillLeft++
				listStillLeft = append(listStillLeft, i)
			}
		}
		if nbNewNodes > nbStillLeft {
			if nbStillLeft == 0 {
				nbNewNodes = 0
			} else {
				nbNewNodes = rand.Intn(nbStillLeft)
			}
		}
		log.LLvl1("\033[48;5;42mRandom Registration of  ", nbNewNodes, " after:  ", int64(time.Now().Sub(startTime)/time.Millisecond), "ms\033[0m ")

		sort.Ints(listStillLeft)

		for i := 0; i < nbNewNodes; i++ {
			go func(idx int) {
				s := services[idx].(*Service)
				log.LLvl1(s.Name, "is trying to update")
				if s.GetEpoch() != e-1 {
					name := s.GetRandomName()
					assert.NoError(t, s.UpdateHistoryWith(name))
				}
				log.LLvl1(s.Name, "is trying to update")
			}(listStillLeft[i])

			go func(idx int) {

				waitTime := time.Duration(rand.Intn(7)*1000+500) * time.Millisecond

				log.LLvl1("\033[48;5;42mNew Node", services[idx].(*Service).Name, " is waiting :  ", waitTime, "to try to create proof\033[0m ")
				time.Sleep(waitTime)
				writeToFile(fmt.Sprintf("%v,Wants,%v,%v", services[idx].(*Service).Name, int64(time.Now().Sub(startTime)/time.Millisecond), e), "Data/comparison_join.txt")
				err := services[idx].(*Service).CreateProofForEpoch(e)
				if err == nil {
					writeToFile(fmt.Sprintf("%v,Manage,%v,%v", services[idx].(*Service).Name, int64(time.Now().Sub(startTime)/time.Millisecond), e), "Data/comparison_join.txt")
					alreadyWritten[idx] = true
					services[idx].(*Service).StartNewEpoch()
				} else {
					writeToFile(fmt.Sprintf("%v,Don't Manage,%v,%v", services[idx].(*Service).Name, int64(time.Now().Sub(startTime)/time.Millisecond), e), "Data/comparison_join.txt")
				}
			}(listStillLeft[i])

		}
		wg.Wait()

		for i := 0; i < nbNewNodes; i++ {
			idx := listStillLeft[i]
			alreadyIn[idx] = true
			stillOut[idx] = false
		}
		log.LLvl1("\033[48;5;45mStarting :", e, "\033[0m ")
		log.LLvl1("OLD COMMITTEE: ", oldCommittee, len(oldCommittee))

		// Running consensus - pick a random leader in the previous committee
		go func(oc []int) {
			log.LLvl1("OLD COMMITTEE in go process: ", oc, len(oc))
			leaderID := rand.Intn(len(oc))
			log.LLvl1("Leader", leaderID, oc[leaderID])
			assert.NoError(t, services[oc[leaderID]].(*Service).GetConsencusOnNewSigners())
		}(oldCommittee)

		oldCommittee = make([]int, 0)
		for i, b := range alreadyWritten {
			if b {
				oldCommittee = append(oldCommittee, i)
				wg.Add(1)
				go func(idx int) {
					assert.NoError(t, services[idx].(*Service).StartNewEpoch())
					wg.Done()
				}(i)
			}
		}
		wg.Wait()
	}
	wg.Wait()
}

func TestNodesChurning(t *testing.T) {
	t.Skip("Skipping Churning Test as it left things littering.")
	nbrNodes := 20
	nbrEpoch := Epoch(10)
	nbFirstSigners := nbrNodes
	rmFile("Data/messages.txt")
	rmFile("Data/storage.txt")
	writeToFile("Name,Function,nb_messages,epoch", "Data/messages.txt")
	writeToFile("Name,Function,storage,epoch", "Data/storage.txt")

	stillIn := make(map[int]bool, nbrNodes)
	alreadyOut := make(map[int]bool, nbrNodes)
	alreadyWritten := make(map[int]bool, nbrNodes)

	local := onet.NewTCPTest(tSuite)

	hosts, _, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()
	services := local.GetServices(hosts, MembershipID)
	for i, s := range services {
		s.(*Service).Name = "node_" + strconv.Itoa(i)
		stillIn[i] = true
		alreadyOut[i] = false
		alreadyWritten[i] = false
	}

	servers := make(map[*network.ServerIdentity]string)

	oldCommittee := make([]int, 0)
	for i := 0; i < nbFirstSigners; i++ {
		servers[hosts[i].ServerIdentity] = services[i].(*Service).Name
		log.LLvl1("Signers 0 : ", hosts[i].ServerIdentity)
		oldCommittee = append(oldCommittee, i)
	}

	var wg sync.WaitGroup

	for _, s := range services {
		wg.Add(1)
		go func(serv *Service) {
			serv.SetGenesisSigners(servers)
			wg.Done()
		}(s.(*Service))

	}
	wg.Wait()

	rmFile("Data/comparison_churn.txt")
	writeToFile("Name,Registration,Time,Epoch", "Data/comparison_churn.txt")
	for i, b := range stillIn {
		if b {
			writeToFile(fmt.Sprintf("%v,In the system,%v,%v", services[i].(*Service).Name, 0, 0), "Data/comparison_churn.txt")
		}
	}
	startTime := time.Now()

	for e := Epoch(1); e < nbrEpoch; e++ {

		log.LLvl1("\033[48;5;42mStart of Epoch ", e, " after:  ", int64(time.Now().Sub(startTime)/time.Millisecond), "\033[0m ")

		for i, b := range stillIn {
			if b {
				log.LLvl1("Service : ", services[i].(*Service).Name, " : ", services[i].(*Service).ServerIdentity())
				if i == 0 {
					writeToFile(fmt.Sprintf("node_0,starts epoch,%v,%v", int64(time.Now().Sub(startTime)/time.Millisecond), e), "Data/comparison_churn.txt")
				}
			}
		}

		// Random Churn :
		var nbNewNodes int
		nbNewNodes = rand.Intn(3)
		nbStillLeft := 0
		listStillLeft := make([]int, nbStillLeft)
		for i, b := range stillIn {
			if b {
				nbStillLeft++
				listStillLeft = append(listStillLeft, i)
			}
		}
		if nbNewNodes > nbStillLeft-4 {
			if nbStillLeft == 0 {
				nbNewNodes = 0
			} else {
				nbNewNodes = rand.Intn(nbStillLeft)
			}
		}
		log.LLvl1("\033[48;5;42mRandom Churn of  ", nbNewNodes, " after:  ", int64(time.Now().Sub(startTime)/time.Millisecond), "ms\033[0m ")

		sort.Sort(sort.Reverse(sort.IntSlice(listStillLeft)))

		for i := 0; i < nbNewNodes; i++ {

			go func(idx int) {
				waitTime := time.Duration(rand.Intn(3)*1000+500) * time.Millisecond
				log.LLvl1("\033[48;5;42mNew Node", services[idx].(*Service).Name, " is waiting :  ", waitTime, "to churn\033[0m ")
				time.Sleep(waitTime)
				writeToFile(fmt.Sprintf("%v,churns,%v,%v", services[idx].(*Service).Name, int64(time.Now().Sub(startTime)/time.Millisecond), e), "Data/comparison_churn.txt")
				hosts[idx].Pause()
			}(listStillLeft[i])

		}

		for i := 0; i < nbNewNodes; i++ {
			idx := listStillLeft[i]
			stillIn[idx] = false
			alreadyOut[idx] = true
		}
		log.LLvl1("\033[48;5;43mRegistration : ", e, " for ", stillIn, " nodes\033[0m ")

		oldCommittee = make([]int, 0)
		// Update for new nodes.
		for i, b := range stillIn {
			if b {
				oldCommittee = append(oldCommittee, i)
				wg.Add(1)

				go func(idx int) {
					s := services[idx].(*Service)

					ro, err := s.getRosterForEpoch(e)

					if s.GetEpoch() != e-1 || err != nil || ro == nil {
						log.LLvl1(s.Name, "is trying to update")
						name := s.GetRandomName()
						assert.NoError(t, s.UpdateHistoryWith(name))
					}
					wg.Done()
					log.LLvl1(s.Name, "Finishing update -------------------------------------------")
				}(i)
			}
		}
		wg.Wait()
		// Registration
		for i, b := range stillIn {
			if b {
				wg.Add(1)

				go func(idx int) {
					assert.NoError(t, services[idx].(*Service).CreateProofForEpoch(e))
					wg.Done()
					log.LLvl1(services[idx].(*Service).Name, "Finishing CREATE PROOF -------------------------------------------")
				}(i)
			} else {
				if !alreadyWritten[i] {
					writeToFile(fmt.Sprintf("%v,left the system,%v,%v", services[i].(*Service).Name, int64(time.Now().Sub(startTime)/time.Millisecond), e), "Data/comparison_churn.txt")
					alreadyWritten[i] = true
				}
			}
		}
		wg.Wait()

		log.LLvl1("\033[48;5;45mStarting :", e, "\033[0m ")

		// Running consensus - pick a random leader in the previous committee
		go func(oc []int) {
			log.LLvl1("OLD COMMITTEE in go process: ", oc, len(oc))
			leaderID := rand.Intn(len(oc))
			log.LLvl1("Leader", leaderID, oc[leaderID])
			assert.NoError(t, services[oc[leaderID]].(*Service).GetConsencusOnNewSigners())
		}(oldCommittee)

		for i, b := range stillIn {
			if b {
				wg.Add(1)
				go func(idx int) {
					assert.NoError(t, services[idx].(*Service).StartNewEpoch())
					wg.Done()
					log.LLvl1(services[idx].(*Service), "Finishing EPOCH -------------------------------------------")
				}(i)
			}
		}

		wg.Wait()
	}
	wg.Wait()
}
