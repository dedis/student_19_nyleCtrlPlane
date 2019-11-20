package nylechain

import (
	"sync"
	"testing"

	gpr "github.com/dedis/student_19_nyleCtrlPlane/gossipregistrationprotocol"
	mbrSer "github.com/dedis/student_19_nyleCtrlPlane/membershipchainservice"
	"github.com/stretchr/testify/assert"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/v3"
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

	signers := mbrSer.SignersSet{
		hosts[0].ServerIdentity.ID: gpr.SignatureResponse{},
		hosts[1].ServerIdentity.ID: gpr.SignatureResponse{},
	}

	var wg sync.WaitGroup
	for _, s := range services {
		wg.Add(1)
		go func(serv *mbrSer.Service) {
			serv.SetGenesisSigners(signers)
			wg.Done()
		}(s.(*mbrSer.Service))
	}
	wg.Wait()

	for i := 0; i < nbrNodes; i++ {
		wg.Add(1)
		go func(idx int) {
			services[idx].(*mbrSer.Service).Registrate(roster, 1)
			wg.Done()
		}(i)
	}
	wg.Wait()

	for i := 0; i < nbrNodes; i++ {
		wg.Add(1)
		go func(idx int) {
			assert.NoError(t, services[idx].(*mbrSer.Service).StartNewEpoch(roster))
			wg.Done()
		}(i)

	}
	wg.Wait()

}
