package service

import (
	_ "crypto/sha256"
	"sync"
	"testing"

	"go.dedis.ch/onet/v3/network"

	"github.com/dedis/student_19_nyleCtrlPlane/gentree"
	"github.com/dedis/student_19_nyleCtrlPlane/transaction"
	"go.dedis.ch/kyber/v3/util/random"
	"go.dedis.ch/protobuf"

	"go.dedis.ch/kyber/v3/sign/bls"

	"go.dedis.ch/kyber/v3/pairing"

	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
)

var testSuite = pairing.NewSuiteBn256()

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func TestTreesBLSCoSi(t *testing.T) {
	filename := "../nodeGen/nodes.txt"
	local := onet.NewTCPTest(testSuite)
	servers, roster, _ := local.GenTree(50, true)
	mapOfServers := make(map[string]*onet.Server)
	lc := gentree.LocalityContext{}
	lc.Setup(roster, filename)
	defer local.CloseAll()

	// Translating the trees into sets

	var fullTreeSlice []*onet.Tree
	var serverIDs []*network.ServerIdentity
	for _, server := range servers {
		mapOfServers[server.ServerIdentity.String()] = server
		serverIDs = append(serverIDs, server.ServerIdentity)
	}

	for _, trees := range lc.LocalityTrees {
		for _, tree := range trees[1:] {
			fullTreeSlice = append(fullTreeSlice, tree)
		}
	}

	translations := TreesToSetsOfNodes(fullTreeSlice, roster.List)
	distances := CreateMatrixOfDistances(serverIDs, lc.Nodes)
	for _, server := range servers {
		service := server.Service(ServiceName).(*Service)
		service.Setup(&SetupArgs{
			Roster:       roster,
			Translations: translations,
			Distances:    distances,
			Filename:     filename,
		})
	}

	PvK0, PbK0 := bls.NewKeyPair(testSuite, random.New())
	_, PbK1 := bls.NewKeyPair(testSuite, random.New())
	_, PbK2 := bls.NewKeyPair(testSuite, random.New())

	PubK0, _ := PbK0.MarshalBinary()
	PubK1, _ := PbK1.MarshalBinary()
	PubK2, _ := PbK2.MarshalBinary()
	iD0 := []byte("Genesis0")
	iD1 := []byte("Genesis1")
	coinID := []byte("0")
	coinID1 := []byte("1")

	// First transaction
	inner := transaction.InnerTx{
		CoinID:     coinID,
		PreviousTx: iD0,
		SenderPK:   PubK0,
		ReceiverPK: PubK1,
	}
	innerEncoded, _ := protobuf.Encode(&inner)
	signature, _ := bls.Sign(testSuite, PvK0, innerEncoded)
	tx := transaction.Tx{
		Inner:     inner,
		Signature: signature,
	}
	txEncoded, _ := protobuf.Encode(&tx)
	innerAlt := transaction.InnerTx{
		CoinID:     coinID,
		PreviousTx: iD0,
		SenderPK:   PubK0,
		ReceiverPK: PubK2,
	}
	innerEncodedAlt, _ := protobuf.Encode(&innerAlt)
	signatureAlt, _ := bls.Sign(testSuite, PvK0, innerEncodedAlt)
	txAlt := transaction.Tx{
		Inner:     innerAlt,
		Signature: signatureAlt,
	}
	txEncodedAlt, _ := protobuf.Encode(&txAlt)

	for _, server := range servers {
		service := server.Service(ServiceName).(*Service)
		service.GenesisTx(&GenesisArgs{
			ID:         iD0,
			CoinID:     coinID,
			ReceiverPK: PubK0,
		})
		service.GenesisTx(&GenesisArgs{
			ID:         iD1,
			CoinID:     coinID1,
			ReceiverPK: PubK0,
		})
	}

	var wg sync.WaitGroup
	n := len(servers[:1])
	wg.Add(n)
	for _, server := range servers[:1] {
		go func(server *onet.Server) {
			// I exclude the first tree of every slice since it only contains one node
			trees := lc.LocalityTrees[lc.Nodes.GetServerIdentityToName(server.ServerIdentity)][1:]
			var treeIDs []onet.TreeID
			for _, tree := range trees {
				treeIDs = append(treeIDs, tree.ID)
				log.LLvl1(tree.Roster.List)
			}
			if len(trees) > 0 {
				// First valid Tx
				service := server.Service(ServiceName).(*Service)
				var w sync.WaitGroup
				w.Add(1)
				var err0 error
				go func() {
					_, err0 = service.TreesBLSCoSi(&CoSiTrees{
						Message: txEncoded,
					})

					w.Done()
				}()

				// Double spending attempt

				_, err := service.TreesBLSCoSi(&CoSiTrees{
					Message: txEncodedAlt,
				})
				w.Wait()
				if err0 != nil {
					log.LLvl1("Error for txn1", err0)
				} else {
					log.LLvl1("txn1  accepted")
				}
				if err != nil {
					log.LLvl1("Error for txn2", err)
				} else {
					log.LLvl1("txn2  accepted")
				}

				if err0 == nil && err == nil {
					log.Fatal("Double spending accepted")
				}
			}
			wg.Done()
		}(server)
	}
	wg.Wait()
}
