package membershipchainservice

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.dedis.ch/kyber/v3/suites"
	"go.dedis.ch/onet"
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

	signers := map[string]bool{
		"Test":  true,
		"Test2": true,
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
