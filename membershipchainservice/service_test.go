package membershipchainservice

import (
	"testing"

	"github.com/dedis/cothority/blscosi"
	"github.com/stretchr/testify/assert"
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
	services := local.GetServices(hosts, membershipID)

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
	services := local.GetServices(hosts, membershipID)

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

	// Generate 10 nodes, the first 2 are the first signers
	nbrNodes := 10
	hosts, roster, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()
	services := local.GetServices(hosts, membershipID)

	blsServ := local.GetServices(hosts, blscosi.ServiceID)

	signers := SignersSet{
		hosts[0].ServerIdentity.ID: true,
		hosts[1].ServerIdentity.ID: true,
	}

	var listServices []*Service
	for _, s := range services {
		service := s.(*Service)
		listServices = append(listServices, s.(*Service))
		service.SetGenesisSigners(signers)
	}

	for i := 0; i < nbrNodes; i++ {
		services[i].(*Service).Registrate(blsServ[i].(*blscosi.Service), roster, 1)
	}

	for i := 0; i < nbrNodes; i++ {
		services[i].(*Service).StartNewEpoch()

	}

}
