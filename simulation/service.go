package main

import (
	"bufio"
	"os"
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

const EXP = 3
const LOCAL = false

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
	if EXP == 2 {
		s.CreateRoster(sc, hosts, 1500)
	} else {
		s.CreateRoster(sc, hosts, 2000)
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
	log.LLvl3("Initializing node-index", name)

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

	return s.SimulationBFTree.Node(config)

}

// RunSystemNormally is not an experiment, just run the system normally
func RunSystemNormally(roster *onet.Roster, clients []*nylechain.Client) error {
	size := len(roster.List)
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

	nbrEpoch := mbrSer.Epoch(10)
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

// RunExperiment1 : Run the system for 1 epoch, with a fixed number of node as a committee
// and a varying number of joining nodes
// (example of parameters : committee 100 nodes, joining nodes : 100, 200, 300, ... , 1000)
func RunExperiment1(roster *onet.Roster, clients []*nylechain.Client) error {
	committee := 50
	runs := []int{50, 250, 500, 750, 1000, 1250, 1500, 1750, 2000}

	if LOCAL {
		committee = 5
		runs = []int{10, 20}
	}

	dir, _ := os.Getwd()
	log.LLvl1("Dir from Node : ", dir)

	add := ""
	if strings.HasSuffix(dir, "/simulation/build") {
		add = "../.."
	}
	if strings.HasSuffix(dir, "/remote") {
		add = "."
	}

	os.Remove(add + "/Data/Throughput.txt")
	os.RemoveAll(add + "/Data/Throughput/")
	os.MkdirAll(add+"/Data/Throughput/", 0777)
	file, _ := os.OpenFile(add+"/Data/Throughput.txt", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	w := bufio.NewWriter(file)

	servers := make(map[*network.ServerIdentity]string)
	for i := 0; i < committee; i++ {
		si := roster.List[i]
		servers[si] = GetServerIdentityToName(si, roster)
		log.LLvl1("Signers 0 : ", si)
	}

	for _, r := range runs {

		log.LLvl1("\033[48;5;42mStart Run ", r, "\033[0m ")
		w.WriteString("Start run " + strconv.Itoa(r) + "\n")
		w.Flush()

		answChan := make(chan int)
		for i := 0; i < r; i++ {
			go func(idx int, c chan int) {
				log.LLvl1("Sending Gen request to server", idx, roster.List[idx])
				clients[idx].SetGenesisSignersRequest(roster.List[idx], servers)
				log.LLvl1("Idx : ", idx, "sends to the channel")
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
		var wg sync.WaitGroup
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

func RunExperiment2(roster *onet.Roster, clients []*nylechain.Client) error {
	committee := 50
	runs := []int{50, 250, 500, 750, 1000, 1100}

	if LOCAL {
		committee = 5
		runs = []int{10, 20}
	}

	dir, _ := os.Getwd()
	log.LLvl1("Dir from Node : ", dir)

	add := ""
	if strings.HasSuffix(dir, "/simulation/build") {
		add = "../.."
	}
	if strings.HasSuffix(dir, "/remote") {
		add = "."
	}

	os.Remove(add + "/Data/Throughput.txt")
	os.RemoveAll(add + "/Data/Throughput/")
	os.MkdirAll(add+"/Data/Throughput/", 0777)
	file, _ := os.OpenFile(add+"/Data/Throughput.txt", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	w := bufio.NewWriter(file)

	servers := make(map[*network.ServerIdentity]string)
	for i := 0; i < committee; i++ {
		si := roster.List[i]
		servers[si] = GetServerIdentityToName(si, roster)
		log.LLvl1("Signers 0 : ", si)
	}

	for _, r := range runs {

		log.LLvl1("\033[48;5;42mStart Run ", r, "\033[0m ")
		w.WriteString("Start run " + strconv.Itoa(r) + "\n")
		w.Flush()

		answChan := make(chan int)
		for i := 0; i < r; i++ {
			go func(idx int, c chan int) {
				log.LLvl1("Sending Gen request to server", idx, roster.List[idx])
				clients[idx].SetGenesisSignersRequest(roster.List[idx], servers)
				log.LLvl1("Idx : ", idx, "sends to the channel")
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
		var wg sync.WaitGroup
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

func RunExperiment3(roster *onet.Roster, clients []*nylechain.Client) error {
	committee := 10
	runs := []int{50, 250, 500, 750, 1000}

	if LOCAL {
		committee = 5
		runs = []int{10, 20}
	}

	dir, _ := os.Getwd()
	log.LLvl1("Dir from Node : ", dir)

	add := ""
	if strings.HasSuffix(dir, "/simulation/build") {
		add = "../.."
	}
	if strings.HasSuffix(dir, "/remote") {
		add = "."
	}

	os.Remove(add + "/Data" + EXPERIMENT_FOLDER + "/Throughput.txt")
	os.RemoveAll(add + "/Data" + EXPERIMENT_FOLDER + "/Throughput/")
	os.MkdirAll(add+"/Data"+EXPERIMENT_FOLDER+"/Throughput/", 0777)
	file, err := os.OpenFile(add+"/Data"+EXPERIMENT_FOLDER+"/Throughput.txt", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		log.LLvl1("Cannot open the file ", add+"/Data"+EXPERIMENT_FOLDER+"/Throughput.txt")
		return err
	}
	file2, err := os.OpenFile(add+"/Data"+EXPERIMENT_FOLDER+"/Throughput/test.txt", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		log.LLvl1("Cannot open the file ", add+"/Data"+EXPERIMENT_FOLDER+"/Throughput/test.txt")
		return err
	}
	file2.Close()
	os.Remove(add + "/Data" + EXPERIMENT_FOLDER + "/Throughput/test.txt")
	w := bufio.NewWriter(file)

	servers := make(map[*network.ServerIdentity]string)
	for i := 0; i < committee; i++ {
		si := roster.List[i]
		servers[si] = GetServerIdentityToName(si, roster)
		log.LLvl1("Signers 0 : ", si)
	}

	for _, r := range runs {

		log.LLvl1("\033[48;5;42mStart Run ", r, "\033[0m ")
		w.WriteString("Start run " + strconv.Itoa(r) + "\n")
		w.Flush()

		answChan := make(chan int)
		for i := 0; i < r; i++ {
			go func(idx int, c chan int) {
				log.LLvl1("Sending Gen request to server", idx, roster.List[idx])
				clients[idx].SetGenesisSignersRequest(roster.List[idx], servers)
				log.LLvl1("Idx : ", idx, "sends to the channel")
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
		var wg sync.WaitGroup
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
	}
	file.Close()
	return nil
}

func RunExperiment4(roster *onet.Roster, clients []*nylechain.Client) error {
	committee := 50
	runs := []int{50, 250, 500, 750, 1000, 1100}

	if LOCAL {
		committee = 5
		runs = []int{10, 20}
	}

	dir, _ := os.Getwd()
	log.LLvl1("Dir from Node : ", dir)

	add := ""
	if strings.HasSuffix(dir, "/simulation/build") {
		add = "../.."
	}
	if strings.HasSuffix(dir, "/remote") {
		add = "."
	}

	os.Remove(add + "/Data/Throughput.txt")
	os.RemoveAll("/Data/Throughput/")
	os.MkdirAll("/Data/Throughput/", 0777)
	file, _ := os.OpenFile(add+"/Data/Throughput.txt", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	w := bufio.NewWriter(file)

	servers := make(map[*network.ServerIdentity]string)
	for i := 0; i < committee; i++ {
		si := roster.List[i]
		servers[si] = GetServerIdentityToName(si, roster)
		log.LLvl1("Signers 0 : ", si)
	}

	for _, r := range runs {

		log.LLvl1("\033[48;5;42mStart Run ", r, "\033[0m ")
		w.WriteString("Start run " + strconv.Itoa(r) + "\n")
		w.Flush()

		answChan := make(chan int)
		for i := 0; i < r; i++ {
			go func(idx int, c chan int) {
				log.LLvl1("Sending Gen request to server", idx, roster.List[idx])
				clients[idx].SetGenesisSignersRequest(roster.List[idx], servers)
				log.LLvl1("Idx : ", idx, "sends to the channel")
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
		var wg sync.WaitGroup
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

// Run is used on the destination machines and runs a number of
// rounds
func (s *SimulationService) Run(config *onet.SimulationConfig) error {
	size := len(config.Roster.List)

	var clients []*nylechain.Client
	for i := 0; i < size; i++ {
		clients = append(clients, nylechain.NewClient())
	}

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
