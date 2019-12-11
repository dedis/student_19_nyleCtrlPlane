package nylechain

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	gpr "github.com/dedis/student_19_nyleCtrlPlane/gossipregistrationprotocol"
	mbrSer "github.com/dedis/student_19_nyleCtrlPlane/membershipchainservice"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

var testSuite = pairing.NewSuiteBn256()

func TestMain(m *testing.M) {
	log.MainTest(m)
}
func TestFewEpochs(t *testing.T) {
	c := NewClient()
	local := onet.NewTCPTest(testSuite)

	nbrNodes := 10
	hosts, _, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()

	services := local.GetServices(hosts, mbrSer.MembershipID)
	for i, s := range services {
		s.(*mbrSer.Service).Name = "node_" + strconv.Itoa(i)
		s.(*mbrSer.Service).PrefixForReadingFile = "."
	}

	servers := make(map[*network.ServerIdentity]string)
	compareSet := make(mbrSer.SignersSet)

	for i := 0; i < 2; i++ {
		servers[hosts[i].ServerIdentity] = services[i].(*mbrSer.Service).Name
		compareSet[hosts[i].ServerIdentity.ID] = gpr.SignatureResponse{}
	}

	for _, h := range hosts {
		_, err := c.SetGenesisSignersRequest(h.ServerIdentity, servers)
		assert.Nil(t, err)
	}
}
