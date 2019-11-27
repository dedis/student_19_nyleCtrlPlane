package membershipchainservice

import (
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
)

var tSuite = suites.MustFind("bn256.adapter")

func TestMain(m *testing.M) {
	log.MainTest(m)
}

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
		compareSet[hosts[i].ServerIdentity.ID] = gpr.SignatureResponse{}
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
		compareSet[hosts[i].ServerIdentity.ID] = gpr.SignatureResponse{}
	}

	for _, s := range services {
		s.(*Service).SetGenesisSigners(servers)
	}

	for i := 0; i < nbrNodes; i++ {
		assert.NoError(t, services[i].(*Service).CreateProofForEpoch(1))
		err := services[i].(*Service).CreateProofForEpoch(2)
		assert.NotNil(t, err)
		err = services[i].(*Service).CreateProofForEpoch(0)
		assert.NotNil(t, err)
	}

	for i := 0; i < nbrNodes; i++ {
		assert.NoError(t, services[i].(*Service).ShareProof())
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

func TestGetServer(t *testing.T) {
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

	// Gives everybody different genesis set and try to reconstruct the whole system
	for i := 0; i < nbrNodes; i++ {
		servers[hosts[i].ServerIdentity] = services[i].(*Service).Name
		compareSet[hosts[i].ServerIdentity.ID] = gpr.SignatureResponse{}
		services[nbrNodes-i-1].(*Service).SetGenesisSigners(servers)
	}

	for i := 0; i < nbrNodes; i++ {
		retServ := services[i].(*Service).GetGlobalServers()
		for _, serv := range services {
			assert.Contains(t, retServ, serv.(*Service).Name)
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
		compareSet[hosts[i].ServerIdentity.ID] = gpr.SignatureResponse{Hash: []uint8{}, Signature: []uint8{}}
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
			assert.NoError(t, serv.AgreeOnState(roster))
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
			assert.Error(t, serv.AgreeOnState(roster))
			wg.Done()
		}(s.(*Service))
	}
	wg.Wait()

}

func TestNewEpoch(t *testing.T) {
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
		compareSet[hosts[i].ServerIdentity.ID] = gpr.SignatureResponse{}
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

	for i := 0; i < nbrNodes; i++ {
		wg.Add(1)
		go func(idx int) {
			services[idx].(*Service).ShareProof()
			wg.Done()
		}(i)
	}
	wg.Wait()

	for i := 0; i < nbrNodes; i++ {
		wg.Add(1)
		go func(idx int) {
			assert.NoError(t, services[idx].(*Service).StartNewEpoch())
			wg.Done()
		}(i)

	}
	wg.Wait()

}

func TestClockRegistrateShareAndNewEpoch(t *testing.T) {
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
		compareSet[hosts[i].ServerIdentity.ID] = gpr.SignatureResponse{}
	}

	var wg sync.WaitGroup
	wg.Add(len(services))
	for _, s := range services {
		go func(serv *Service) {
			serv.SetGenesisSigners(servers)
			serv.StartClock()
			wg.Done()
		}(s.(*Service))
	}
	wg.Wait()

	for i := 0; i < nbrNodes/2; i++ {
		wg.Add(1)
		go func(idx int) {
			assert.Error(t, services[idx].(*Service).StartNewEpoch())
			wg.Done()
		}(i)

	}
	wg.Wait()

	for i := 0; i < nbrNodes/2; i++ {
		wg.Add(1)
		go func(idx int) {
			assert.NoError(t, services[idx].(*Service).CreateProofForEpoch(1))
			wg.Done()
		}(i)
	}
	wg.Wait()

	for i := 0; i < nbrNodes/2; i++ {
		wg.Add(1)
		go func(idx int) {
			assert.NoError(t, services[idx].(*Service).ShareProof())
			wg.Done()
		}(i)
	}
	wg.Wait()

	time.Sleep(REGISTRATION_DUR)

	for i := nbrNodes / 2; i < nbrNodes; i++ {
		wg.Add(1)
		go func(idx int) {
			assert.Error(t, services[idx].(*Service).CreateProofForEpoch(1))
			wg.Done()
		}(i)
	}
	wg.Wait()

	for i := 0; i < nbrNodes/2; i++ {
		wg.Add(1)
		go func(idx int) {
			assert.NoError(t, services[idx].(*Service).ShareProof())
			wg.Done()
		}(i)
	}
	wg.Wait()

	for i := 0; i < nbrNodes/2; i++ {
		wg.Add(1)
		go func(idx int) {
			assert.Error(t, services[idx].(*Service).StartNewEpoch())
			wg.Done()
		}(i)

	}
	wg.Wait()

	time.Sleep(SHARE_DUR)
	log.LLvl1("Start of Epoch 1")

	for i := 0; i < nbrNodes/2; i++ {
		wg.Add(1)
		go func(idx int) {
			assert.Error(t, services[idx].(*Service).ShareProof())
			wg.Done()
		}(i)
	}
	wg.Wait()

	for i := 0; i < nbrNodes/2; i++ {
		wg.Add(1)
		go func(idx int) {
			assert.NoError(t, services[idx].(*Service).StartNewEpoch())
			wg.Done()
		}(i)

	}
	wg.Wait()

}

func TestWholeSystemOverFewEpochs(t *testing.T) {
	t.Skip("A lot of function are not implemented for now")
	nbrNodes := 20
	nbrEpoch := Epoch(10)

	joiningPerEpoch := int(0.1 * float64(nbrNodes))

	local := onet.NewTCPTest(tSuite)

	hosts, _, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()
	services := local.GetServices(hosts, MembershipID)
	for i, s := range services {
		s.(*Service).Name = "node_" + strconv.Itoa(i)
	}

	servers := make(map[*network.ServerIdentity]string)

	for i := 0; i < 4; i++ {
		servers[hosts[i].ServerIdentity] = services[i].(*Service).Name
		log.LLvl1("Signers 0 : ", hosts[i].ServerIdentity)
	}

	var wg sync.WaitGroup

	for _, s := range services {
		wg.Add(1)
		go func(serv *Service) {
			serv.SetGenesisSigners(servers)
			serv.StartClock()
			wg.Done()
		}(s.(*Service))
	}
	wg.Wait()

	for e := Epoch(1); e < nbrEpoch; e++ {
		log.LLvl1("\033[48;5;42mStart of Epoch ", e, "\033[0m ")

		for i := 0; i < joiningPerEpoch*(int(e)+1); i++ {
			log.LLvl1("Service : ", services[i].(*Service).ServerIdentity())

		}

		log.LLvl1("\033[48;5;43mRegistration : ", e, " for ", joiningPerEpoch*(int(e)+1), " nodes\033[0m ")
		// Registration
		for i := 0; i < joiningPerEpoch*(int(e)+1); i++ {
			wg.Add(1)
			go func(idx int) {
				assert.NoError(t, services[idx].(*Service).CreateProofForEpoch(e))
				wg.Done()
			}(i)
		}
		wg.Wait()

		time.Sleep(REGISTRATION_DUR)
		log.LLvl1("\033[48;5;44mSharing :", e, "\033[0m ")
		// Sharing
		for i := 0; i < joiningPerEpoch*(int(e)+1); i++ {
			wg.Add(1)
			go func(idx int) {
				assert.NoError(t, services[idx].(*Service).ShareProof())
				wg.Done()
			}(i)
		}
		wg.Wait()

		time.Sleep(SHARE_DUR)
		log.LLvl1("\033[48;5;45mStarting :", e, "\033[0m ")

		for i := 0; i < joiningPerEpoch*(int(e)+1); i++ {
			wg.Add(1)
			go func(idx int) {
				assert.NoError(t, services[idx].(*Service).StartNewEpoch())
				wg.Done()
			}(i)
		}
		wg.Wait()

		// WHAT TO DO DURING THE EPOCH ?
		time.Sleep(EPOCH_DUR)

		/*

			// CHURN - With Deregistration
			for i := 0; i < joiningPerEpoch*(int(e)+1); i++ {
				// NOT IMPLEMENTED
				if rand.Float64() < cR/2 {
					err = services[i].(*Service).Deregistrate()
					if err != nil {
						return err
					}
				}

			}

			// CHRUN - Without Deregistration
			for i := 0; i < joiningPerEpoch*(int(e)+1); i++ {
				if rand.Float64() < cR/2 {
					hosts[i].Router.Pause()
				}
			}

			// CHANGE IN LATENCIES
			for i := 0; i < nbrNodes; i++ {
				// NOT IMPLEMENTED
				services[i].(*Service).ChangeLatencies(ic)
			}

		*/

	}
}
