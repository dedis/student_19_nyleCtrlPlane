package membershipchainservice

import (
	rand "math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	gpr "github.com/dedis/student_19_nyleCtrlPlane/gossipregistrationprotocol"
	"github.com/stretchr/testify/assert"
	"go.dedis.ch/kyber/v3/suites"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"go.dedis.ch/protobuf"
)

var tSuite = suites.MustFind("bn256.adapter")

func TestMain(m *testing.M) {
	log.MainTest(m)
}

var emptySign = gpr.SignatureResponse{Hash: []uint8{}, Signature: []uint8{}}

func TestSetGenesisSigners(t *testing.T) {

	local := onet.NewTCPTest(tSuite)

	// Generate 10 nodes, the first 2 are the first signers
	nbrNodes := 10
	hosts, _, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()
	services := local.GetServices(hosts, MembershipID)
	for i, s := range services {
		s.(*Service).Name = "node_" + strconv.Itoa(i)
	}

	servers := make(map[*network.ServerIdentity]string)
	compareSet := make(SignersSet)

	for i := 0; i < 2; i++ {
		servers[hosts[i].ServerIdentity] = services[i].(*Service).Name
		compareSet[hosts[i].ServerIdentity.ID] = emptySign
	}

	for _, s := range services {
		service := s.(*Service)

		reply := service.GetSigners(0)
		assert.Equal(t, len(reply.Set), 0)

		service.SetGenesisSigners(servers)

		reply = service.GetSigners(0)
		assert.Equal(t, reply.Set, compareSet)

		reply = service.GetSigners(1)
		assert.Equal(t, len(reply.Set), 0)

	}
}

func TestRegisterNewSigners(t *testing.T) {
	local := onet.NewTCPTest(tSuite)

	// Generate 10 nodes, the first 2 are the first signers
	nbrNodes := 10
	hosts, _, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()
	services := local.GetServices(hosts, MembershipID)
	for i, s := range services {
		s.(*Service).Name = "node_" + strconv.Itoa(i)
	}

	servers := make(map[*network.ServerIdentity]string)
	compareSet := make(SignersSet)

	for i := 0; i < 2; i++ {
		servers[hosts[i].ServerIdentity] = services[i].(*Service).Name
		compareSet[hosts[i].ServerIdentity.ID] = emptySign
	}

	for _, s := range services {
		s.(*Service).SetGenesisSigners(servers)
	}

	for i := 0; i < nbrNodes; i++ {
		err := services[i].(*Service).CreateProofForEpoch(2)
		assert.NotNil(t, err)
		assert.NoError(t, services[i].(*Service).CreateProofForEpoch(1))

		err = services[i].(*Service).CreateProofForEpoch(0)
		assert.NotNil(t, err)
	}

	// Running consensus - pick a random leader in the previous committee
	leaderID := rand.Intn(2)
	assert.NoError(t, services[leaderID].(*Service).GetConsencusOnNewSigners())

	time.Sleep(500 * time.Millisecond)
	for i := 0; i < nbrNodes; i++ {
		for _, s := range services[:i] {
			service := s.(*Service)
			reply := service.GetSigners(1)

			assert.Contains(t, reply.Set, services[i].(*Service).ServerIdentity().ID)

			// Does not change the signers of Epoch 0
			reply = service.GetSigners(0)
			assert.Equal(t, reply.Set, compareSet)

		}
	}

}

func TestExecHistoryRequestAndReply(t *testing.T) {
	local := onet.NewTCPTest(tSuite)
	nbrNodes := 2

	hosts, _, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()
	services := local.GetServices(hosts, MembershipID)
	for i, s := range services {
		s.(*Service).Name = "node_" + strconv.Itoa(i)
	}

	servers := make(map[*network.ServerIdentity]string)
	compareSet := make(SignersSet)

	for i := 0; i < 2; i++ {
		servers[hosts[i].ServerIdentity] = services[i].(*Service).Name
		compareSet[hosts[i].ServerIdentity.ID] = emptySign
	}
	s0 := services[0].(*Service)
	s1 := services[1].(*Service)

	s0.SetGenesisSigners(servers)

	assert.NotEqual(t, s0.Servers, s1.Servers)
	assert.NotEqual(t, s0.ServerIdentityToName, s1.ServerIdentityToName)

	s1.Servers = make(map[string]*network.ServerIdentity)
	s1.Servers[s0.Name] = s0.ServerIdentity()
	assert.NoError(t, s1.UpdateHistoryWith(s0.Name))
	reply := s1.GetSigners(0)
	assert.Equal(t, compareSet, reply.Set)

	assert.Equal(t, s0.ServerIdentityToName, s1.ServerIdentityToName)

}

func TestAgreeOn(t *testing.T) {
	local := onet.NewTCPTest(tSuite)
	nbrNodes := 10
	hosts, roster, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()

	services := local.GetServices(hosts, MembershipID)
	for i, s := range services {
		s.(*Service).Name = "node_" + strconv.Itoa(i)
	}

	servers := make(map[*network.ServerIdentity]string)
	compareSet := make(SignersSet)

	for i := 0; i < 2; i++ {
		servers[hosts[i].ServerIdentity] = services[i].(*Service).Name
		compareSet[hosts[i].ServerIdentity.ID] = gpr.SignatureResponse{}
	}
	// Set same state
	var wg sync.WaitGroup
	for _, s := range services {
		wg.Add(1)
		go func(serv *Service) {
			serv.SetGenesisSigners(servers)
			wg.Done()
		}(s.(*Service))
	}
	wg.Wait()
	// Should work if all have the same state
	for _, s := range services {
		wg.Add(1)
		go func(serv *Service) {
			_, err := serv.AgreeOnState(roster, SIGNERSMSG)
			assert.NoError(t, err)
			_, err = serv.AgreeOnState(roster, PINGSMSG)
			assert.NoError(t, err)
			wg.Done()
		}(s.(*Service))
	}
	wg.Wait()

	// Set different State
	for _, s := range services[0:5] {
		wg.Add(1)
		go func(serv *Service) {
			serv.SetGenesisSigners(nil)
			wg.Done()
		}(s.(*Service))
	}
	wg.Wait()

	// Should fail for some
	for _, s := range services {
		wg.Add(1)
		go func(serv *Service) {
			_, err := serv.AgreeOnState(roster, SIGNERSMSG)
			assert.Error(t, err)
			_, err = serv.AgreeOnState(roster, PINGSMSG)
			assert.NoError(t, err)
			wg.Done()
		}(s.(*Service))
	}
	wg.Wait()

}

func TestGetConsensusOnNewSigners(t *testing.T) {
	local := onet.NewTCPTest(tSuite)
	nbrNodes := 10
	hosts, _, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()

	services := local.GetServices(hosts, MembershipID)
	for i, s := range services {
		s.(*Service).Name = "node_" + strconv.Itoa(i)
	}

	servers := make(map[*network.ServerIdentity]string)
	compareSet := make(SignersSet)

	for i := 0; i < 2; i++ {
		servers[hosts[i].ServerIdentity] = services[i].(*Service).Name
		compareSet[hosts[i].ServerIdentity.ID] = emptySign
	}

	var wg sync.WaitGroup
	wg.Add(len(services))
	for _, s := range services {
		go func(serv *Service) {
			serv.SetGenesisSigners(servers)
			wg.Done()
		}(s.(*Service))
	}
	wg.Wait()

	for i := 0; i < nbrNodes; i++ {
		wg.Add(1)
		go func(idx int) {
			services[idx].(*Service).CreateProofForEpoch(1)
			wg.Done()
		}(i)
	}
	wg.Wait()

	log.LLvl1("________---------- CONSENCUS ----------_________")
	// Each node of the previous committee should be able to get consensus
	for i := 0; i < 2; i++ {
		assert.NoError(t, services[i].(*Service).GetConsencusOnNewSigners())
	}

}

func TestNewEpoch(t *testing.T) {
	local := onet.NewTCPTest(tSuite)
	nbrNodes := 10
	hosts, roster, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()

	services := local.GetServices(hosts, MembershipID)
	for i, s := range services {
		s.(*Service).Name = "node_" + strconv.Itoa(i)
	}

	servers := make(map[*network.ServerIdentity]string)
	compareSet := make(SignersSet)

	for i := 0; i < 2; i++ {
		servers[hosts[i].ServerIdentity] = services[i].(*Service).Name
		compareSet[hosts[i].ServerIdentity.ID] = emptySign
	}

	var wg sync.WaitGroup
	wg.Add(len(services))
	for _, s := range services {
		go func(serv *Service) {
			serv.SetGenesisSigners(servers)
			wg.Done()
		}(s.(*Service))
	}
	wg.Wait()

	for i := 0; i < nbrNodes; i++ {
		wg.Add(1)
		go func(idx int) {
			services[idx].(*Service).CreateProofForEpoch(1)
			wg.Done()
		}(i)
	}
	wg.Wait()

	// Running consensus - pick a random leader in the previous committee
	leaderID := rand.Intn(2)
	assert.NoError(t, services[leaderID].(*Service).GetConsencusOnNewSigners())

	for i := 0; i < nbrNodes; i++ {
		wg.Add(1)
		go func(idx int) {
			assert.NoError(t, services[idx].(*Service).StartNewEpoch())
			wg.Done()
		}(i)

	}
	wg.Wait()

	log.LLvl1("\033[48;5;20mPassing new epoch but can we agree on GraphTree ?\033[0m")
	// Should work if all have the same state
	for _, s := range services {
		wg.Add(1)
		go func(serv *Service) {
			_, err := serv.AgreeOnState(roster, SIGNERSMSG)
			assert.NoError(t, err)
			_, err = serv.AgreeOnState(roster, PINGSMSG)
			assert.NoError(t, err)
			wg.Done()
		}(s.(*Service))
	}
	wg.Wait()

}
func TestWholeSystemOverFewEpochs(t *testing.T) {
	//t.Skip("A lot of function are not implemented for now")
	nbrNodes := 20
	nbrEpoch := Epoch(10)
	nbFirstSigners := 4
	writeToFile("Name,Function,nb_messages,epoch", "Data/messages.txt")
	writeToFile("Name,Function,storage,epoch", "Data/storage.txt")
	joiningPerEpoch := int(0.1 * float64(nbrNodes))

	local := onet.NewTCPTest(tSuite)

	hosts, _, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()
	services := local.GetServices(hosts, MembershipID)
	for i, s := range services {
		s.(*Service).Name = "node_" + strconv.Itoa(i)
	}

	servers := make(map[*network.ServerIdentity]string)

	for i := 0; i < nbFirstSigners; i++ {
		servers[hosts[i].ServerIdentity] = services[i].(*Service).Name
		log.LLvl1("Signers 0 : ", hosts[i].ServerIdentity)
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

	for e := Epoch(1); e < nbrEpoch; e++ {
		log.LLvl1("\033[48;5;42mStart of Epoch ", e, "\033[0m ")

		for i := 0; i < joiningPerEpoch*(int(e)+1); i++ {
			log.LLvl1("Service : ", services[i].(*Service).Name, " : ", services[i].(*Service).ServerIdentity())
		}

		log.LLvl1("\033[48;5;43mRegistration : ", e, " for ", joiningPerEpoch*(int(e)+1), " nodes\033[0m ")

		// Update for new nodes.
		for i := 0; i < joiningPerEpoch*(int(e)+1); i++ {
			wg.Add(1)
			go func(idx int) {
				s := services[idx].(*Service)

				if s.GetEpoch() != e-1 {
					name := s.GetRandomName()
					assert.NoError(t, s.UpdateHistoryWith(name))
				}
				wg.Done()
			}(i)
		}
		wg.Wait()

		// Registration
		for i := 0; i < joiningPerEpoch*(int(e)+1); i++ {
			wg.Add(1)
			go func(idx int) {
				assert.NoError(t, services[idx].(*Service).CreateProofForEpoch(e))
				wg.Done()
			}(i)
		}
		wg.Wait()

		log.LLvl1("\033[48;5;45mStarting :", e, "\033[0m ")

		// Running consensus - pick a random leader in the previous committee
		go func() {
			leaderID := rand.Intn(joiningPerEpoch * int(e))
			assert.NoError(t, services[leaderID].(*Service).GetConsencusOnNewSigners())
		}()

		for i := 0; i < joiningPerEpoch*(int(e)+1); i++ {
			wg.Add(1)
			go func(idx int) {
				assert.NoError(t, services[idx].(*Service).StartNewEpoch())
				wg.Done()
			}(i)
		}
		wg.Wait()
	}
}

func TestFailingBLSCOSI(t *testing.T) {
	local := onet.NewTCPTest(tSuite)

	// nbrNodes := 200 is failing
	nbrNodes := 100

	hosts, _, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()
	services := local.GetServices(hosts, MembershipID)
	for i, s := range services {
		s.(*Service).Name = "node_" + strconv.Itoa(i)
		s.(*Service).Cycle.Sequence = []time.Duration{1 * time.Minute, 1 * time.Minute}
	}

	servers := make(map[*network.ServerIdentity]string)

	for i := 0; i < nbrNodes/2; i++ {
		servers[hosts[i].ServerIdentity] = services[i].(*Service).Name
		log.LLvl1("Signers 0 : ", hosts[i].ServerIdentity)
	}

	var wg sync.WaitGroup

	for _, s := range services {
		wg.Add(1)
		go func(serv *Service) {
			serv.SetGenesisSigners(servers)
			serv.Cycle.Sequence = []time.Duration{2 * time.Minute, 5 * time.Minute}
			serv.Cycle.StartTime = time.Now()
			wg.Done()
		}(s.(*Service))
	}
	wg.Wait()

	// Registration
	for i := 0; i < nbrNodes; i++ {
		wg.Add(1)
		go func(idx int) {
			assert.NoError(t, services[idx].(*Service).CreateProofForEpoch(1))
			wg.Done()
		}(i)
	}

	wg.Wait()
}

func TestFailingProtobufEncode(t *testing.T) {

	local := onet.NewTCPTest(tSuite)
	nbrNodes := 10
	hosts, _, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()

	services := local.GetServices(hosts, MembershipID)
	for i, s := range services {
		s.(*Service).Name = "node_" + strconv.Itoa(i)
	}

	servers := make(map[*network.ServerIdentity]string)
	compareSet := make(SignersSet)

	for i := 0; i < 2; i++ {
		servers[hosts[i].ServerIdentity] = services[i].(*Service).Name
		compareSet[hosts[i].ServerIdentity.ID] = emptySign
	}

	var wg sync.WaitGroup
	wg.Add(len(services))
	for _, s := range services {
		go func(serv *Service) {
			serv.SetGenesisSigners(servers)
			wg.Done()
		}(s.(*Service))
	}
	wg.Wait()

	// Error for empty map
	_, err := protobuf.Encode(&services[0].(*Service).PingDistances)
	log.LLvl1("An error is expected for Protobuf encode, here is its value : ", err)
	assert.NotNil(t, err)

	for i := 0; i < nbrNodes; i++ {
		wg.Add(1)
		go func(idx int) {
			services[idx].(*Service).CreateProofForEpoch(1)
			wg.Done()
		}(i)
	}
	wg.Wait()

	// Running consensus - pick a random leader in the previous committee
	leaderID := rand.Intn(2)
	assert.NoError(t, services[leaderID].(*Service).GetConsencusOnNewSigners())

	for i := 0; i < nbrNodes; i++ {
		wg.Add(1)
		go func(idx int) {
			assert.NoError(t, services[idx].(*Service).StartNewEpoch())
			wg.Done()
		}(i)

	}
	wg.Wait()

	// Error for map of map
	_, err = protobuf.Encode(&services[0].(*Service).PingDistances)
	log.LLvl1("An error is expected for Protobuf encode, here is its value : ", err)
	assert.NotNil(t, err)

}
