package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	nylechain "github.com/dedis/student_19_nyleCtrlPlane"
	mbrSer "github.com/dedis/student_19_nyleCtrlPlane/membershipchainservice"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

const EXP = 11
const LOCAL = true

var EXPERIMENT_FOLDER = ""

/*
 * Defines the simulation for the service-template
 */

func init() {
	onet.SimulationRegister("TemplateService", NewSimulationService)
}

// SimulationService only holds the BFTree simulation
type SimulationService struct {
	onet.SimulationBFTree
}

// NewSimulationService returns the new simulation, where all fields are
// initialised using the config-file
func NewSimulationService(config string) (onet.Simulation, error) {
	es := &SimulationService{}
	_, err := toml.Decode(config, es)
	if err != nil {
		return nil, err
	}
	return es, nil
}

// Setup creates the tree used for that simulation
func (s *SimulationService) Setup(dir string, hosts []string) (
	*onet.SimulationConfig, error) {
	sc := &onet.SimulationConfig{}
	// TODO : sort
	// hpc
	/*if EXP == 2 || EXP == 52 {
	//	s.CreateRoster(sc, hosts, 1500)
	//}
	if EXP >= 3 {
		s.CreateRoster(sc, hosts, 1000)
	} else {
		s.CreateRoster(sc, hosts, 2000)
	}*/
	if EXP == 7 || EXP == 8 || EXP == 11 {
		s.CreateRoster(sc, hosts, 20)
	} else if EXP == 10 {
		s.CreateRoster(sc, hosts, 20)
	} else {
		s.CreateRoster(sc, hosts, 999)
	}

	err := s.CreateTree(sc)
	if err != nil {
		return nil, err
	}
	return sc, nil
}

// Node can be used to initialize each node before it will be run
// by the server. Here we call the 'Node'-method of the
// SimulationBFTree structure which will load the roster- and the
// tree-structure to speed up the first round.
func (s *SimulationService) Node(config *onet.SimulationConfig) error {
	name := GetServerIdentityToName(config.Server.ServerIdentity, config.Roster)
	log.LLvl3("Initializing node-index", name, config.Server.ServerIdentity)

	myservice := config.GetService(mbrSer.ServiceName).(*mbrSer.Service)
	myservice.Name = name

	dir, _ := os.Getwd()
	log.LLvl1("Dir from Node : ", dir)

	if strings.HasSuffix(dir, "/simulation/build") {
		myservice.PrefixForReadingFile = "../.."
	}
	if strings.HasSuffix(dir, "/remote") {
		myservice.PrefixForReadingFile = "."
	}
	mbrSer.EXPERIMENT_FOLDER = EXPERIMENT_FOLDER
	if !LOCAL {
		mbrSer.REGISTRATION_DUR = 10 * time.Second
		mbrSer.EPOCH_DUR = 20 * time.Second
	}

	if EXP == 0 || EXP == 11 {
		mbrSer.REGISTRATION_DUR = 10 * time.Second
		mbrSer.EPOCH_DUR = 10 * time.Second
	}

	return s.SimulationBFTree.Node(config)

}

// RunSystemNormally is not an experiment, just run the system normally
func RunSystemNormally(roster *onet.Roster, clients []*nylechain.Client) error {
	size := len(roster.List)
	for i, s := range roster.List {
		log.LLvl1("Roster: ", i, ":", s)

	}
	servers := make(map[*network.ServerIdentity]string)
	nbrFirstSigners := 4
	for i := 0; i < nbrFirstSigners; i++ {
		si := roster.List[i]
		servers[si] = GetServerIdentityToName(si, roster)
		log.LLvl1("Signers 0 : ", si)
	}

	for i := 0; i < size; i++ {
		clients[i].SetGenesisSignersRequest(roster.List[i], servers)
	}

	nbrEpoch := mbrSer.Epoch(20)

	if LOCAL {
		nbrEpoch = mbrSer.Epoch(5)
	}
	joiningPerEpoch := int(1 / float64(nbrEpoch) * float64(size))

	var wg sync.WaitGroup

	for e := mbrSer.Epoch(1); e < nbrEpoch; e++ {
		log.LLvl1("\033[48;5;42mStart of Epoch ", e, "\033[0m ")
		for i := 0; i < joiningPerEpoch*(int(e)+1); i++ {
			log.LLvl1("Nodes : node_", i, " : ", roster.List[i])
			wg.Add(1)
			go func(idx int, ee mbrSer.Epoch) {
				clients[idx].ExecEpochRequest(roster.List[idx], ee)
				wg.Done()
			}(i, e)
		}
		wg.Wait()
	}
	return nil
}

// RunExperiment run a Standard experiment
func RunExperiment(roster *onet.Roster, clients []*nylechain.Client, runs []int, committee int, reg_duration time.Duration) error {
	mbrSer.REGISTRATION_DUR = reg_duration
	var wg sync.WaitGroup

	for i := 0; i < len(clients); i++ {
		wg.Add(1)
		go func(idx int) {
			clients[idx].SetRegistrationDuration(roster.List[idx], reg_duration)
			wg.Done()
		}(i)
	}
	wg.Wait()

	if LOCAL {
		committee = 5
		runs = []int{10, 20}
	}

	dir, _ := os.Getwd()
	add := ""
	if strings.HasSuffix(dir, "/simulation/build") {
		add = "../.."
	}
	if strings.HasSuffix(dir, "/remote") {
		add = "."
	}

	os.MkdirAll(add+"/Data/Throughput/", 0777)
	file, _ := os.OpenFile(add+"/Data/Throughput.txt", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	w := bufio.NewWriter(file)
	w.WriteString("Parameters, Committee : " + strconv.Itoa(committee) + " - Time for registration : " + strconv.Itoa(int(mbrSer.REGISTRATION_DUR/time.Millisecond)) + "\n")
	w.Flush()

	servers := make(map[*network.ServerIdentity]string)
	for i := 0; i < committee; i++ {
		si := roster.List[i]
		servers[si] = GetServerIdentityToName(si, roster)
		log.LLvl1("Signers 0 : ", si)
	}

	for _, r := range runs {
		if r < committee {
			continue
		}
		log.LLvl1("\033[48;5;42mStart Run ", r, "- Committee : ", committee, "Time : ", mbrSer.REGISTRATION_DUR, "\033[0m ")
		w.WriteString("Start run " + strconv.Itoa(r) + "\n")
		w.Flush()

		answChan := make(chan int)
		for i := 0; i < r; i++ {
			go func(idx int, c chan int) {
				clients[idx].SetGenesisSignersRequest(roster.List[idx], servers)
				c <- idx
			}(i, answChan)
		}

		totalJobsLeft := r
		for j := range answChan {
			totalJobsLeft--
			log.LLvl1("Node", j, "Terminated. Jobs Left : ", totalJobsLeft)
			if totalJobsLeft == 0 {
				break
			}
		}
		close(answChan)

		for i := 0; i < r; i++ {
			log.LLvl1("Nodes : node_", i, " : ", roster.List[i])
			wg.Add(1)
			go func(idx int) {
				clients[idx].ExecEpochRequest(roster.List[idx], 1)
				wg.Done()
			}(i)
		}
		wg.Wait()
		clients[0].ExecWriteSigners(roster.List[0], 1)
		time.Sleep(10 * time.Second)
	}
	file.Close()
	return nil
}

// RunExperiment1 : Run the system for 1 epoch, with a fixed number of node as a committee
// and a varying number of joining nodes
// (example of parameters : committee 100 nodes, joining nodes : 100, 200, 300, ... , 1000)
func RunExperiment1(roster *onet.Roster, clients []*nylechain.Client) error {
	committee := 50
	//runs := []int{50, 250, 500, 750, 1000, 1250, 1500, 1750, 2000}
	// MicroCLoud
	runs := []int{50, 250, 500, 750, 1000}

	return RunExperiment(roster, clients, runs, committee, mbrSer.REGISTRATION_DUR)
}

// RunExperiment2 : Same as 1 but 10 computers instead of 20
// reduce the number of computer to see if the drop in registration comes from the ressources
func RunExperiment2(roster *onet.Roster, clients []*nylechain.Client) error {
	committee := 50
	//runs := []int{50, 250, 500, 750, 1000, 1100}
	// MicroCLoud
	runs := []int{50, 250, 500, 750, 1000}

	return RunExperiment(roster, clients, runs, committee, mbrSer.REGISTRATION_DUR)
}

// RunExperiment3 : varying the number of node in the committee and see how its affect the throughput
func RunExperiment3(roster *onet.Roster, clients []*nylechain.Client) error {
	committees := []int{5, 10, 20, 50, 100, 200}
	runs := []int{50, 250, 500, 750, 1000}
	for _, committee := range committees {
		RunExperiment(roster, clients, runs, committee, mbrSer.REGISTRATION_DUR)
	}

	return nil
}

// RunExperiment4 : As the main supposition is that the duration of the registration is the only factor of refusal. Vary the registration duration to put that effect into light.
func RunExperiment4(roster *onet.Roster, clients []*nylechain.Client) error {
	committee := 50
	// runs := []int{50, 250, 500, 750, 1000}
	runs := []int{50, 100, 300, 500}
	durations := []time.Duration{200 * time.Millisecond, 500 * time.Millisecond, 1 * time.Second, 2 * time.Second, 5 * time.Second, 10 * time.Second, 20 * time.Second}
	//durations := []time.Duration{30 * time.Second, 20 * time.Second, 10 * time.Second}

	for _, dur := range durations {
		RunExperiment(roster, clients, runs, committee, dur)
	}

	return nil
}

// RunExperiment5 : Redo some experiment allowing more time
func RunExperiment5(roster *onet.Roster, clients []*nylechain.Client) error {
	committees := []int{100, 200}
	runs := []int{50, 250, 500, 750, 1000}
	durations := []time.Duration{10 * time.Second, 20 * time.Second, 1 * time.Minute, 2 * time.Minute}
	for _, committee := range committees {
		for _, dur := range durations {
			RunExperiment(roster, clients, runs, committee, dur)
		}
	}

	return nil
}

// RunExperiment52 : Redo some experiment allowing more time
func RunExperiment52(roster *onet.Roster, clients []*nylechain.Client) error {
	committee := 50
	runs := []int{50, 250, 500, 750, 1000, 1250, 1500, 1750, 2000}

	return RunExperiment(roster, clients, runs, committee, 2*time.Minute)

	return nil
}

func RunExperimentNodesWantingToJoing(roster *onet.Roster, clients []*nylechain.Client) error {
	nbrNodes := len(clients)
	nbrEpoch := mbrSer.Epoch(20)
	nbFirstSigners := 4
	rmFile("Data/messages.txt")
	rmFile("Data/storage.txt")
	writeToFile("Name,Function,nb_messages,epoch", "Data/messages.txt")
	writeToFile("Name,Function,storage,epoch", "Data/storage.txt")

	stillOut := make(map[int]bool, nbrNodes)
	alreadyIn := make(map[int]bool, nbrNodes)
	alreadyWritten := make(map[int]bool, nbrNodes)

	for i := range clients {
		stillOut[i] = true
		alreadyIn[i] = false
		alreadyWritten[i] = false
	}

	servers := make(map[*network.ServerIdentity]string)

	oldCommittee := make([]int, 0)
	for i := 0; i < nbFirstSigners; i++ {
		si := roster.List[i]
		servers[si] = GetServerIdentityToName(si, roster)
		log.LLvl1("Signers 0 : ", si)
		alreadyIn[i] = true
		alreadyWritten[i] = true
		stillOut[i] = false
		oldCommittee = append(oldCommittee, i)
	}

	var wg sync.WaitGroup

	for i := 0; i < nbrNodes; i++ {
		clients[i].SetGenesisSignersRequest(roster.List[i], servers)
	}

	rmFile("Data/comparison_join.txt")
	writeToFile("Name,Registration,Time,epoch", "Data/comparison_join.txt")
	for i, b := range alreadyIn {
		if b {
			writeToFile(fmt.Sprintf("%v,Manage Normally,%v,%v", GetServerIdentityToName(roster.List[i], roster), 0, 0), "Data/comparison_join.txt")
		}
	}
	startTime := time.Now()

	for e := mbrSer.Epoch(1); e <= nbrEpoch; e++ {

		log.LLvl1("\033[48;5;42mStart of Epoch ", e, " after:  ", int64(time.Now().Sub(startTime)/time.Millisecond), "\033[0m ")

		for i, b := range alreadyIn {
			if b {
				log.LLvl1("Service : ", GetServerIdentityToName(roster.List[i], roster), " : ", roster.List[i])
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
				go func(idx int, ep mbrSer.Epoch) {
					clients[idx].UpdateForNewNode(roster.List[idx], ep)
					wg.Done()
				}(i, e)
			}
		}
		wg.Wait()
		// Registration
		for i, b := range alreadyIn {
			if b {
				wg.Add(1)
				go func(idx int, ep mbrSer.Epoch) {
					writeToFile(fmt.Sprintf("%v,Wants As Usual,%v,%v", GetServerIdentityToName(roster.List[idx], roster), int64(time.Now().Sub(startTime)/time.Millisecond), e), "Data/comparison_join.txt")
					_, err := clients[idx].CreateProofForEpochRequest(roster.List[idx], ep)
					log.LLvl1("Error in create proof", err)
					if !alreadyWritten[idx] {
						writeToFile(fmt.Sprintf("%v,Manage Normally,%v,%v", GetServerIdentityToName(roster.List[idx], roster), int64(time.Now().Sub(startTime)/time.Millisecond), e), "Data/comparison_join.txt")
						alreadyWritten[idx] = true
					}
					wg.Done()
				}(i, e)
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
				log.LLvl1(GetServerIdentityToName(roster.List[idx], roster), "is trying to update")
				clients[idx].UpdateNode(roster.List[idx])
			}(listStillLeft[i])

			go func(idx int, ep mbrSer.Epoch) {
				waitTime := time.Duration(rand.Intn(7)*1000+500) * time.Millisecond

				log.LLvl1("\033[48;5;42mNew Node", GetServerIdentityToName(roster.List[idx], roster), " is waiting :  ", waitTime, "to try to create proof\033[0m ")
				time.Sleep(waitTime)
				writeToFile(fmt.Sprintf("%v,Wants,%v,%v", GetServerIdentityToName(roster.List[idx], roster), int64(time.Now().Sub(startTime)/time.Millisecond), e), "Data/comparison_join.txt")
				_, err := clients[idx].CreateProofForEpochRequest(roster.List[idx], ep)
				if err == nil {
					writeToFile(fmt.Sprintf("%v,Manage,%v,%v", GetServerIdentityToName(roster.List[idx], roster), int64(time.Now().Sub(startTime)/time.Millisecond), e), "Data/comparison_join.txt")
					alreadyWritten[idx] = true
					clients[idx].StartNewEpochRequest(roster.List[idx])
				} else {
					writeToFile(fmt.Sprintf("%v,Don't Manage,%v,%v", GetServerIdentityToName(roster.List[idx], roster), int64(time.Now().Sub(startTime)/time.Millisecond), e), "Data/comparison_join.txt")
				}
			}(listStillLeft[i], e)

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
			clients[oc[leaderID]].GetConsencusOnNewSignersRequest(roster.List[oc[leaderID]])
		}(oldCommittee)

		oldCommittee = make([]int, 0)
		for i, b := range alreadyWritten {
			if b {
				oldCommittee = append(oldCommittee, i)
				wg.Add(1)
				go func(idx int) {
					clients[idx].StartNewEpochRequest(roster.List[idx])
					wg.Done()
				}(i)
			}
		}
		wg.Wait()
	}
	wg.Wait()

	return nil
}

func RunExperimentNodesChurning(roster *onet.Roster, clients []*nylechain.Client) error {
	nbrNodes := len(clients)
	nbrEpoch := mbrSer.Epoch(20)
	nbFirstSigners := nbrNodes
	rmFile("Data/messages.txt")
	rmFile("Data/storage.txt")
	writeToFile("Name,Function,nb_messages,epoch", "Data/messages.txt")
	writeToFile("Name,Function,storage,epoch", "Data/storage.txt")

	stillIn := make(map[int]bool, nbrNodes)
	alreadyOut := make(map[int]bool, nbrNodes)
	alreadyWritten := make(map[int]bool, nbrNodes)

	for i := range clients {
		stillIn[i] = true
		alreadyOut[i] = false
		alreadyWritten[i] = false
	}

	servers := make(map[*network.ServerIdentity]string)

	oldCommittee := make([]int, 0)
	for i := 0; i < nbFirstSigners; i++ {
		si := roster.List[i]
		servers[si] = GetServerIdentityToName(si, roster)
		log.LLvl1("Signers 0 : ", si)
		oldCommittee = append(oldCommittee, i)
	}

	var wg sync.WaitGroup

	for i := 0; i < nbrNodes; i++ {
		wg.Add(1)
		go func(idx int) {
			clients[idx].SetGenesisSignersRequest(roster.List[idx], servers)
			wg.Done()
		}(i)

	}
	wg.Wait()

	rmFile("Data/comparison_churn.txt")
	writeToFile("Name,Registration,Time,Epoch", "Data/comparison_churn.txt")
	for i, b := range stillIn {
		if b {
			writeToFile(fmt.Sprintf("%v,In the system,%v,%v", GetServerIdentityToName(roster.List[i], roster), 0, 0), "Data/comparison_churn.txt")
		}
	}
	startTime := time.Now()

	for e := mbrSer.Epoch(1); e < nbrEpoch; e++ {

		log.LLvl1("\033[48;5;42mStart of Epoch ", e, " after:  ", int64(time.Now().Sub(startTime)/time.Millisecond), "\033[0m ")

		for i, b := range stillIn {
			if b {
				log.LLvl1("Service : ", GetServerIdentityToName(roster.List[i], roster), " : ", roster.List[i])
				if i == 0 {
					writeToFile(fmt.Sprintf("node_0,starts epoch,%v,%v", int64(time.Now().Sub(startTime)/time.Millisecond), e), "Data/comparison_churn.txt")
				}
			}
		}

		// Random Churn :
		var nbNewNodes int
		nbNewNodes = rand.Intn(5)
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
				log.LLvl1("\033[48;5;42mNew Node", GetServerIdentityToName(roster.List[idx], roster), " is waiting :  ", waitTime, "to churn\033[0m ")
				time.Sleep(waitTime)
				writeToFile(fmt.Sprintf("%v,churns,%v,%v", GetServerIdentityToName(roster.List[idx], roster), int64(time.Now().Sub(startTime)/time.Millisecond), e), "Data/comparison_churn.txt")
				clients[idx].SendPause(roster.List[idx])
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

				go func(idx int, ep mbrSer.Epoch) {
					clients[idx].UpdateForNewNode(roster.List[idx], ep)
					wg.Done()
				}(i, e)
			}
		}
		wg.Wait()
		// Registration
		for i, b := range stillIn {
			if b {
				wg.Add(1)

				go func(idx int, ep mbrSer.Epoch) {
					clients[idx].CreateProofForEpochRequest(roster.List[idx], ep)
					wg.Done()
					log.LLvl1(GetServerIdentityToName(roster.List[idx], roster), "Finishing CREATE PROOF -------------------------------------------")
				}(i, e)
			} else {
				if !alreadyWritten[i] {
					writeToFile(fmt.Sprintf("%v,left the system,%v,%v", GetServerIdentityToName(roster.List[i], roster), int64(time.Now().Sub(startTime)/time.Millisecond), e), "Data/comparison_churn.txt")
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
			clients[oc[leaderID]].GetConsencusOnNewSignersRequest(roster.List[oc[leaderID]])
		}(oldCommittee)

		for i, b := range stillIn {
			if b {
				wg.Add(1)
				go func(idx int) {
					clients[idx].StartNewEpochRequest(roster.List[idx])
					wg.Done()
					log.LLvl1(GetServerIdentityToName(roster.List[i], roster), "Finishing EPOCH -------------------------------------------")
				}(i)
			}
		}

		wg.Wait()
	}
	wg.Wait()

	return nil
}

// Run is used on the destination machines and runs a number of
// rounds
func (s *SimulationService) Run(config *onet.SimulationConfig) error {
	size := len(config.Roster.List)

	var clients []*nylechain.Client
	for i := 0; i < size; i++ {
		clients = append(clients, nylechain.NewClient())
	}
	dir, _ := os.Getwd()
	add := ""
	if strings.HasSuffix(dir, "/simulation/build") {
		add = "../.."
	}
	if strings.HasSuffix(dir, "/remote") {
		add = "."
	}
	os.Remove(add + "/Data/Throughput.txt")
	os.RemoveAll(add + "/Data/Throughput/")
	os.RemoveAll(add + "/Data/Timing/")

	if EXP == 1 {
		log.LLvl1("= = = = = = = = = = = = = = = = = = = = = = = = = = = = = = =Experiment #1= = = = = = = = = = = = = = = = = = = = = = = = = = = = = = =")
		return RunExperiment1(config.Roster, clients)
	}
	if EXP == 2 {
		log.LLvl1("= = = = = = = = = = = = = = = = = = = = = = = = = = = = = = =Experiment #2= = = = = = = = = = = = = = = = = = = = = = = = = = = = = = =")
		return RunExperiment2(config.Roster, clients)
	}
	if EXP == 3 {
		log.LLvl1("= = = = = = = = = = = = = = = = = = = = = = = = = = = = = = =Experiment #3= = = = = = = = = = = = = = = = = = = = = = = = = = = = = = =")
		return RunExperiment3(config.Roster, clients)
	}
	if EXP == 4 {
		log.LLvl1("= = = = = = = = = = = = = = = = = = = = = = = = = = = = = = =Experiment #4= = = = = = = = = = = = = = = = = = = = = = = = = = = = = = =")
		return RunExperiment4(config.Roster, clients)
	}
	if EXP == 5 {
		log.LLvl1("= = = = = = = = = = = = = = = = = = = = = = = = = = = = = = =Experiment #5= = = = = = = = = = = = = = = = = = = = = = = = = = = = = = =")
		return RunExperiment5(config.Roster, clients)
	}
	if EXP == 52 {
		log.LLvl1("= = = = = = = = = = = = = = = = = = = = = = = = = = = = = = =Experiment #5 - 2= = = = = = = = = = = = = = = = = = = = = = = = = = = = = = =")
		return RunExperiment52(config.Roster, clients)
	}
	if EXP == 7 {
		return RunExperimentNodesWantingToJoing(config.Roster, clients)
	}
	if EXP == 8 {
		return RunExperimentNodesChurning(config.Roster, clients)
	}

	return RunSystemNormally(config.Roster, clients)
}

//GetServerIdentityToName translate a SI from the global roster to its name
func GetServerIdentityToName(si *network.ServerIdentity, roster *onet.Roster) string {
	index, _ := roster.Search(si.ID)
	if index < 0 {
		log.Fatal("Didn't find this node in roster")
	}
	return "node_" + strconv.Itoa(index)
}
