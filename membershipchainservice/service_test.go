package membershipchainservice

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/dedis/cothority/blscosi"
	"github.com/dedis/student_19_nyleCtrlPlane/gentree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v3/suites"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
)

var tSuite = suites.MustFind("Ed25519")

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

	signers := SignersSet{
		hosts[0].ServerIdentity.ID: true,
		hosts[1].ServerIdentity.ID: true,
	}

	for _, s := range services {
		service := s.(*Service)

		reply := service.GetSigners(0)
		assert.Equal(t, len(reply.Set), 0)

		service.SetGenesisSigners(signers)

		reply = service.GetSigners(0)
		assert.Equal(t, reply.Set, signers)

		reply = service.GetSigners(1)
		assert.Equal(t, len(reply.Set), 0)

	}
}

func TestRegisterNewSigners(t *testing.T) {
	local := onet.NewTCPTest(tSuite)

	// Generate 10 nodes, the first 2 are the first signers
	nbrNodes := 10
	hosts, roster, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()
	services := local.GetServices(hosts, MembershipID)

	blsServ := local.GetServices(hosts, blscosi.ServiceID)

	signers := SignersSet{
		hosts[0].ServerIdentity.ID: true,
		hosts[1].ServerIdentity.ID: true,
	}

	for _, s := range services {
		s.(*Service).SetGenesisSigners(signers)
	}

	for i := 0; i < nbrNodes; i++ {
		assert.NoError(t, services[i].(*Service).Registrate(blsServ[i].(*blscosi.Service), roster, 1))
		err := services[i].(*Service).Registrate(blsServ[i].(*blscosi.Service), roster, 2)
		assert.NotNil(t, err)
		err = services[i].(*Service).Registrate(blsServ[i].(*blscosi.Service), roster, 0)
		assert.NotNil(t, err)

		for _, s := range services {
			service := s.(*Service)
			reply := service.GetSigners(1)

			assert.Contains(t, reply.Set, services[i].(*Service).ServerIdentity().ID)

			// Does not change the signers of Epoch 0
			reply = service.GetSigners(0)
			assert.Equal(t, reply.Set, signers)

		}
	}
}

func TestNewEpoch(t *testing.T) {
	local := onet.NewTCPTest(tSuite)
	nbrNodes := 10
	hosts, roster, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()

	services := local.GetServices(hosts, MembershipID)

	blsServ := local.GetServices(hosts, blscosi.ServiceID)

	signers := SignersSet{
		hosts[0].ServerIdentity.ID: true,
		hosts[1].ServerIdentity.ID: true,
	}

	var wg sync.WaitGroup
	for _, s := range services {
		wg.Add(1)
		go func(serv *Service) {
			serv.SetGenesisSigners(signers)
			wg.Done()
		}(s.(*Service))
	}
	wg.Wait()

	for i := 0; i < nbrNodes; i++ {
		wg.Add(1)
		go func(idx int) {
			services[idx].(*Service).Registrate(blsServ[idx].(*blscosi.Service), roster, 1)
			wg.Done()
		}(i)
	}
	wg.Wait()

	lc := gentree.LocalityContext{}
	lc.Setup(roster, "../gentree/nodes_small.txt")

	for i := 0; i < nbrNodes; i++ {
		wg.Add(1)
		go func(idx int) {
			services[idx].(*Service).StartNewEpoch(roster, lc.Nodes.All)
			wg.Done()
		}(i)
		//assert.NoError(t,  )
	}
	wg.Wait()

}

func TestWholeSystemOverFewEpochs(t *testing.T) {
	t.Skip("A lot of function are not implemented for now")

	// Can be changed to slices to test the system in different cases.
	churnRate := 0.2
	epochRateInHours := 2.0
	interNodeLatencyChange := 0.2

	err := runSystemWithParameters(churnRate, epochRateInHours, interNodeLatencyChange, 10, 10)
	require.Nil(t, err)

}

func runSystemWithParameters(cR, eR, ic float64, nbrNodes int, nbrEpoch Epoch) error {
	var err error
	joiningPerEpoch := int(0.1 * float64(nbrNodes))

	local := onet.NewTCPTest(tSuite)

	hosts, roster, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()
	services := local.GetServices(hosts, MembershipID)

	blsServ := local.GetServices(hosts, blscosi.ServiceID)

	signers := SignersSet{}

	for i := 0; i < joiningPerEpoch; i++ {
		signers[hosts[i].ServerIdentity.ID] = true
	}

	for _, s := range services {
		s.(*Service).SetGenesisSigners(signers)
	}

	for e := Epoch(1); e < nbrEpoch; e++ {
		log.LLvl1("Start of Epoch ", e)

		// Registration
		for i := joiningPerEpoch * int(e); i < joiningPerEpoch*(int(e)+1); i++ {
			err = services[i].(*Service).Registrate(blsServ[i].(*blscosi.Service), roster, e)
			if err != nil {
				return err
			}
		}

		// Participating
		s0 := services[0].(*Service)
		cs := s0.GetSigners(e)
		SIs, err := s0.getServerIdentityFromSignersSet(cs.Set, roster)
		if err != nil {
			return err
		}
		roForEpoch := onet.NewRoster(SIs)

		lc := gentree.LocalityContext{}
		// THIS WILL FAIL FOR NOW
		lc.Setup(roForEpoch, "../gentree/nodes_small.txt")

		for i := 0; i < nbrNodes; i++ {
			// THIS MIGHT FAIL FOR NOW - SEEK WHAT WILL BE THE PROBLEM OF STARTING EPOCH WITHOUT ALL THE NODES
			err = services[i].(*Service).StartNewEpoch(roForEpoch, lc.Nodes.All)
			if err != nil {
				return err
			}
		}

		// WHAT TO DO DURING THE EPOCH ?
		time.Sleep(time.Duration(eR) * time.Second)

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

	}
	return nil
}
