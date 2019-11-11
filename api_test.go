package nylechain

import (
	"testing"

	"github.com/dedis/cothority/blscosi"
	"github.com/dedis/student_19_nyleCtrlPlane/gentree"
	mbrSer "github.com/dedis/student_19_nyleCtrlPlane/membershipchainservice"
	"github.com/stretchr/testify/assert"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet"
	"go.dedis.ch/onet/v3/log"
)

var testSuite = pairing.NewSuiteBn256()

func TestMain(m *testing.M) {
	log.MainTest(m)
}
func TestFewEpochs(t *testing.T) {
	_ = NewClient()
	local := onet.NewTCPTest(testSuite)

	nbrNodes := 10
	hosts, roster, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()

	services := local.GetServices(hosts, mbrSer.MembershipID)

	blsServ := local.GetServices(hosts, blscosi.ServiceID)

	signers := mbrSer.SignersSet{
		hosts[0].ServerIdentity.ID: true,
		hosts[1].ServerIdentity.ID: true,
	}

	var listServices []*mbrSer.Service
	for _, s := range services {
		service := s.(*mbrSer.Service)
		listServices = append(listServices, s.(*mbrSer.Service))
		service.SetGenesisSigners(signers)
	}

	for i := 0; i < nbrNodes; i++ {
		services[i].(*mbrSer.Service).Registrate(blsServ[i].(*blscosi.Service), roster, 1)
	}

	lc := gentree.LocalityContext{}
	lc.Setup(roster, "gentree/nodes_small.txt")

	for i := 0; i < nbrNodes; i++ {
		assert.NoError(t, services[i].(*mbrSer.Service).StartNewEpoch(roster, lc.Nodes.All))
	}

}
