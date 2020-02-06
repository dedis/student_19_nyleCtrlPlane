package membershipchainservice

import (
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/dedis/student_19_nyleCtrlPlane/gentree"
	"github.com/stretchr/testify/assert"

	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/network"
)

func TestIfGetInteractionDistanceIsUpdatingPingDistances(t *testing.T) {
	local := onet.NewTCPTest(tSuite)

	// Generate 10 nodes, the first 2 are the first signers
	nbrNodes := 10
	hosts, _, _ := local.GenTree(nbrNodes, true)
	defer local.CloseAll()
	services := local.GetServices(hosts, MembershipID)
	for i, s := range services {
		s.(*Service).Name = "node_" + strconv.Itoa(i)
		s.(*Service).Nodes.All = make([]*gentree.LocalityNode, nbrNodes)
		s.(*Service).Nodes.ServerIdentityToName = make(map[network.ServerIdentityID]string)

	}

	for _, s := range services {

		k := 0
		for _, v := range services {

			s.(*Service).Nodes.All[k] = &gentree.LocalityNode{}
			s.(*Service).Nodes.All[k].Name = v.(*Service).Name
			s.(*Service).Nodes.ServerIdentityToName[v.(*Service).ServerIdentity().ID] = v.(*Service).Name
			s.(*Service).Nodes.All[k].ServerIdentity = v.(*Service).ServerIdentity()
			k++
		}
	}

	servers := make(map[*network.ServerIdentity]string)
	compareSet := make(SignersSet)

	for i := 0; i < 2; i++ {
		servers[hosts[i].ServerIdentity] = services[i].(*Service).Name
		compareSet[hosts[i].ServerIdentity.ID] = emptySign
	}

	for _, s := range services {
		s.(*Service).SetGenesisSigners(servers)
	}

	for i := 0; i < nbrNodes; i++ {
		services[i].(*Service).CreateProofForEpoch(1)
	}

	// Running consensus - pick a random leader in the previous committee
	leaderID := rand.Intn(2)
	services[leaderID].(*Service).GetConsencusOnNewSigners()

	time.Sleep(1500 * time.Millisecond)
	for _, s := range services {
		s.(*Service).e = 1
		s.(*Service).InteractionMtx.Lock()
		s.(*Service).CountInteractions = append(s.(*Service).CountInteractions, make(map[string]int))
		s.(*Service).InteractionMtx.Unlock()
	}

	var wg sync.WaitGroup
	for i := 0; i < nbrNodes; i++ {
		wg.Add(1)
		go func(idx int) {
			services[idx].(*Service).GetInteractionDistances()
			wg.Done()
		}(i)
	}
	wg.Wait()

	pingDist := services[0].(*Service).GetPingDistances()

	assert.NotNil(t, pingDist)

}
