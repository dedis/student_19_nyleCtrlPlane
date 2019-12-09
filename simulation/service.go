package main

import (
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/dedis/student_19_nyleCtrlPlane/membershipchainservice"
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

	myservice := config.GetService(membershipchainservice.ServiceName).(*membershipchainservice.Service)
	myservice.Name = name

	return s.SimulationBFTree.Node(config)

}

// Run is used on the destination machines and runs a number of
// rounds
func (s *SimulationService) Run(config *onet.SimulationConfig) error {
	size := config.Tree.Size()

	nbrFirstSigners := 4

	servers := make(map[*network.ServerIdentity]string)

	for i := 0; i < nbrFirstSigners; i++ {
		si := config.Roster.List[i]
		servers[si] = GetServerIdentityToName(si, config.Roster)
		log.LLvl1("Signers 0 : ", si)
	}

	myservice := config.GetService(membershipchainservice.ServiceName).(*membershipchainservice.Service)
	log.Lvl2("Size is:", size, "my name", myservice.Name)
	myservice.SetGenesisSigners(servers)

	err := myservice.CreateProofForEpoch(1)
	if err != nil {
		return err
	}

	if myservice.Name == "node_0" {
		err = myservice.GetConsencusOnNewSigners()
		if err != nil {
			return err
		}
	}
	err = myservice.StartNewEpoch()
	if err != nil {
		return err
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
