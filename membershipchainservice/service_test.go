package membershipchainservice

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.dedis.ch/kyber/v3/suites"
	"go.dedis.ch/onet/network"
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
	hosts, roster, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()
	services := local.GetServices(hosts, membershipID)

	signers := map[*network.ServerIdentity]bool{
		hosts[0].ServerIdentity: true,
		hosts[1].ServerIdentity: true,
	}

	for _, s := range services {
		service := s.(*Service)

		reply := service.GetSigners()
		assert.Equal(t, len(reply.Set), 0)

		service.SetGenesisSigners(signers)

		reply = service.GetSigners()
		assert.Equal(t, reply.Set, signers)

	}
}

func TestRegisterNewSigners(t *testing.T) {
	local := onet.NewTCPTest(tSuite)

	// Generate 10 nodes, the first 2 are the first signers
	nbrNodes := 10
	hosts, _, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()
	services := local.GetServices(hosts, membershipID)

	signers := map[*onet.Server]bool{
		hosts[0]: true,
		hosts[1]: true,
	}

	for _, s := range services {
		service := s.(*Service)
		service.SetGenesisSigners(signers)
	}

	assert.NoError(t, services[2].(*Service).Registrate())
	for _, s := range services {
		service := s.(*Service)
		reply := service.GetSigners()
		assert.True(t, reply.Set[hosts[2]])

	}

}
