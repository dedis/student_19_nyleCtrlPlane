package membershipchainservice

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/kyber/v3/suites"
	"go.dedis.ch/onet"
	"go.dedis.ch/onet/v3/log"
)

var tSuite = suites.MustFind("Ed25519")

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func TestService_Gossiping(t *testing.T) {
	local := onet.NewTCPTest(tSuite)
	// generate 5 hosts, they don't connect, they process messages, and they
	// don't register the tree or entitylist
	nbrNodes := 5
	hosts, roster, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()

	services := local.GetServices(hosts, templateID)

	for _, s := range services {
		reg, err := s.(*Service).GetRegistrations()
		require.Nil(t, err)
		// registrations should be empty before gossiping
		require.Equal(t, 0, len(reg.List), "Non-empty registrations on node", s)

		log.Lvl2("Sending request to", s)
		_, err = s.(*Service).GossipRegistration(
			&GossipArgs{Roster: roster},
		)
		require.Nil(t, err)

		reg, err = s.(*Service).GetRegistrations()
		require.Nil(t, err)
		// registrations should be full after gossiping
		require.Equal(t, nbrNodes, len(reg.List), "Non-empty registrations on node", s)
		log.LLvl2(reg.List)
	}
}
