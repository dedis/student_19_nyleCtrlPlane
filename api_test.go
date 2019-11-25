package nylechain

import (
	"strconv"
	"sync"
	"testing"

	gpr "github.com/dedis/student_19_nyleCtrlPlane/gossipregistrationprotocol"
	mbrSer "github.com/dedis/student_19_nyleCtrlPlane/membershipchainservice"
	"github.com/stretchr/testify/assert"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/onet/network"
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

	var wg sync.WaitGroup
	wg.Add(len(services))
	for _, s := range services {
		go func(serv *mbrSer.Service) {
			serv.SetGenesisSigners(servers)
			wg.Done()
		}(s.(*mbrSer.Service))
	}
	wg.Wait()

	for i := 0; i < nbrNodes; i++ {
		wg.Add(1)
		go func(idx int) {
			services[idx].(*mbrSer.Service).CreateProofForEpoch(1)
			wg.Done()
		}(i)
	}
	wg.Wait()

	for i := 0; i < nbrNodes; i++ {
		wg.Add(1)
		go func(idx int) {
			services[idx].(*mbrSer.Service).ShareProof()
			wg.Done()
		}(i)
	}
	wg.Wait()

	for i := 0; i < nbrNodes; i++ {
		wg.Add(1)
		go func(idx int) {
			assert.NoError(t, services[idx].(*mbrSer.Service).StartNewEpoch())
			wg.Done()
		}(i)

	}
	wg.Wait()

}
