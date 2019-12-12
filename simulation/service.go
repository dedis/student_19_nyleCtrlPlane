package main

import (
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	nylechain "github.com/dedis/student_19_nyleCtrlPlane"
	mbrSer "github.com/dedis/student_19_nyleCtrlPlane/membershipchainservice"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

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
	s.CreateRoster(sc, hosts, 2000)
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
		myservice.PrefixForReadingFile = "../../"
	}
	if strings.HasSuffix(dir, "/remote") {
		myservice.PrefixForReadingFile = "./"
	}

	return s.SimulationBFTree.Node(config)

}

// Run is used on the destination machines and runs a number of
// rounds
func (s *SimulationService) Run(config *onet.SimulationConfig) error {
	size := len(config.Roster.List)
	nbrFirstSigners := 4

	listServers := config.Roster.List

	var clients []*nylechain.Client
	for i := 0; i < size; i++ {
		clients = append(clients, nylechain.NewClient())
	}

	servers := make(map[*network.ServerIdentity]string)
	for i := 0; i < nbrFirstSigners; i++ {
		si := config.Roster.List[i]
		servers[si] = GetServerIdentityToName(si, config.Roster)
		log.LLvl1("Signers 0 : ", si)
	}

	for i := 0; i < size; i++ {
		clients[i].SetGenesisSignersRequest(listServers[i], servers)
	}

	nbrEpoch := mbrSer.Epoch(10)
	joiningPerEpoch := int(1 / float64(nbrEpoch) * float64(size))

	var wg sync.WaitGroup

	for e := mbrSer.Epoch(1); e < nbrEpoch; e++ {
		log.LLvl1("\033[48;5;42mStart of Epoch ", e, "\033[0m ")

		for i := 0; i < joiningPerEpoch*(int(e)+1); i++ {
			log.LLvl1("Nodes : node_", i, " : ", listServers[i])
			wg.Add(1)
			go func(idx int, ee mbrSer.Epoch) {
				clients[idx].ExecEpochRequest(listServers[idx], ee)
				wg.Done()
			}(i, e)
		}
		wg.Wait()

	}

	return nil
}

//GetServerIdentityToName translate a SI from the global roster to its name
func GetServerIdentityToName(si *network.ServerIdentity, roster *onet.Roster) string {
	index, _ := roster.Search(si.ID)
	if index < 0 {
		log.Fatal("Didn't find this node in roster")
	}
	return "node_" + strconv.Itoa(index)
}
