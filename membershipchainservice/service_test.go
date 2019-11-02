package membershipchainservice

import (
	"testing"

	"go.dedis.ch/kyber/v3/suites"
	"go.dedis.ch/onet"
	"go.dedis.ch/onet/v3/log"
)

var tSuite = suites.MustFind("Ed25519")

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func TestService_Registration(t *testing.T) {
	local := onet.NewTCPTest(tSuite)

	// Generate 10 nodes, the first 2 are the first signers
	// Then the next 2 will try to register for the 1 epoch
	nbrNodes := 10
	hosts, _, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()

	_ = local.GetServices(hosts, templateID)

}
