package membershipchainservice

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"sync"

	"github.com/dedis/paper_crux/dsn_exp/sqlconn"
	"github.com/dedis/student_19_nyleCtrlPlane/gentree"
	"go.dedis.ch/onet"
	"go.dedis.ch/onet/log"
	"go.dedis.ch/onet/simul/monitor"
	"go.dedis.ch/onet/v3/network"
)

// Setup is a method that will initialize the Crux Protocol
// it is copy-pasted from : https://github.com/dedis/paper_crux/blob/master/dsn_exp/service/service.go
func (s *Service) Setup(req *InitRequest) /*([]GraphTree, []*onet.Tree, map[network.ServerIdentityID]string, map[*gentree.LocalityNode]map[*gentree.LocalityNode]float64)*/ {

	s.Nodes.All = req.Nodes
	s.Nodes.ServerIdentityToName = make(map[network.ServerIdentityID]string)
	for k, v := range req.ServerIdentityToName {
		s.Nodes.ServerIdentityToName[k.ID] = v
	}
	for _, myNode := range s.Nodes.All {

		myNode.ADist = make([]float64, 0)
		myNode.PDist = make([]string, 0)
		myNode.OptimalCluster = make(map[string]bool)
		myNode.OptimalBunch = make(map[string]bool)
		myNode.Cluster = make(map[string]bool)
		myNode.Bunch = make(map[string]bool)
		myNode.Rings = make([]string, 0)

	}
	// order nodesin s.Nodes in the order of index
	nodes := make([]*gentree.LocalityNode, len(s.Nodes.All))
	for _, n := range s.Nodes.All {
		nodes[gentree.NodeNameToInt(n.Name)] = n
		log.Info(s.ServerIdentity(), fmt.Sprintf("%+v", nodes[gentree.NodeNameToInt(n.Name)]))
	}
	s.Nodes.All = nodes
	s.Nodes.ClusterBunchDistances = make(map[*gentree.LocalityNode]map[*gentree.LocalityNode]float64)
	s.Nodes.Links = make(map[*gentree.LocalityNode]map[*gentree.LocalityNode]map[*gentree.LocalityNode]bool)
	s.GraphTree = make(map[string][]GraphTree)
	s.BinaryTree = make(map[string][]*onet.Tree)
	s.RedisClusterPorts = make(map[int]bool)
	s.OpReply = make(map[int]*redisOpResult)
	s.PairOpReply = make(map[int]chan bool)
	s.LockedInstances = make(map[int][]LockInfo)
	s.SmallestCluster = 100000
	s.BiggestCluster = 0
	s.EtcdClients = make(map[string]clientv3.KV)
	s.cockroachClients = make(map[string]*sqlconn.Conn)
	s.cockroachHttpPorts = make(map[int]bool)
	s.cockroachPortsToRingMap = make(map[int]string)

	for reqId := req.OpIdxStart; reqId < req.NrOps; reqId++ {
		s.OpReply[reqId] = &redisOpResult{ResChan: make(chan string, 10), ResultUpdated: 0}
		s.PairOpReply[reqId] = make(chan bool)
		s.LockedInstances[reqId] = make([]LockInfo, 0)
	}

	// allocate distances
	for _, node := range s.Nodes.All {
		s.Nodes.ClusterBunchDistances[node] = make(map[*gentree.LocalityNode]float64)
		s.Nodes.Links[node] = make(map[*gentree.LocalityNode]map[*gentree.LocalityNode]bool)
		for _, node2 := range s.Nodes.All {
			s.Nodes.ClusterBunchDistances[node][node2] = math.MaxFloat64
			s.Nodes.Links[node][node2] = make(map[*gentree.LocalityNode]bool)

			if node == node2 {
				s.Nodes.ClusterBunchDistances[node][node2] = 0
			}

			//log.LLvl1("init map", node.Name, node2.Name)
		}
	}

	s.CosiWg = make(map[int]*sync.WaitGroup)
	s.metrics = make(map[string]*monitor.TimeMeasure)
	s.RedisConns = make(map[string]redis.Conn)
	s.OwnPings = make(map[string]float64)
	s.PingDistances = make(map[string]map[string]float64)
	s.NrPingAnswers = 0

	s.RedisPortsToRingsMap = make(map[int]string)
	s.RedisPortsToStandAloneMap = make(map[int]string)

	//ROOT_NAME := s.Nodes.GetServerIdentityToName(s.ServerIdentity())
	//filename := "fullcall-" + ROOT_NAME + ".txt"

	/*
		var file *os.File
		if _, err := os.Stat(filename); !os.IsNotExist(err) {

			//file, err = os.OpenFile(filename, os.O_APPEND | os.O_WRONLY, 0600)
			file, err = os.OpenFile(filename, os.O_WRONLY, 0600)
			if err != nil {
				panic(err)
			}
		} else {
			file, err = os.Create(filename)
		}

		s.File = file
		s.W = bufio.NewWriter(file)
	*/

	/*
		for _, node := range s.Nodes.All {
			log.Lvl1(node.Name)
		}
	*/
	//log.Lvl1(s.Nodes.ServerIdentityToName)

	log.Lvl1("called init service on", s.Nodes.GetServerIdentityToName(s.ServerIdentity()))

	s.getPings(true)

	s.genTrees(RND_NODES, NR_LEVELS, OPTIMIZED, MIN_BUNCH_SIZE, OPTTYPE, s.PingDistances)

	file7, _ := os.Create("compact-pings" + s.Nodes.GetServerIdentityToName(s.ServerIdentity()))
	w7 := bufio.NewWriter(file7)

	for i := 0; i < len(s.Nodes.All); i++ {
		rootName := "node_" + strconv.Itoa(i)
		for _, n := range s.GraphTree[rootName] {

			rosterNames := make([]string, 0)
			for _, si := range n.Tree.Roster.List {
				rosterNames = append(rosterNames, s.Nodes.GetServerIdentityToName(si))
			}

			sort.Strings(rosterNames)

			rosterList := ""
			for _, n := range rosterNames {
				rosterList += n + " "
			}

			w7.WriteString("rootName " + rootName + "creates binary with roster " + rosterList + "\n")

		}

	}

	w7.Flush()
	file7.Close()

	s.ShortestDistances = s.floydWarshall()
	/*
		for i := 0 ; i < len(s.Nodes.All); i++ {
			namei := "node_" + strconv.Itoa(i)
			for j := 0 ; j < len(s.Nodes.All); j++ {
				namej := "node_" + strconv.Itoa(j)

				if math.Abs(s.PingDistances[namei][namej] - s.ShortestDistances[namei][namej]) > 0.5 {
					log.LLvl1("still not cool", namei, namej, "ping=", s.PingDistances[namei][namej], "straight=", s.ShortestDistances[namei][namej])
				}
			}
		}


		s.genTrees(RND_NODES, NR_LEVELS, OPTIMIZED, MIN_BUNCH_SIZE, OPTTYPE, s.ShortestDistances)

		file8, _ := os.Create("compact-warshall" + s.Nodes.GetServerIdentityToName(s.ServerIdentity()))
		w8 := bufio.NewWriter(file8)

		for i := 0 ; i < len(s.Nodes.All); i++ {
			rootName := "node_" + strconv.Itoa(i)
			for _, n := range s.GraphTree[rootName] {

				rosterNames := make([]string, 0)
				for _, si := range n.Tree.Roster.List {
					rosterNames = append(rosterNames, s.Nodes.GetServerIdentityToName(si))
				}

				sort.Strings(rosterNames)

				rosterList := ""
				for _, n := range rosterNames {
					rosterList += n + " "
				}

				w8.WriteString("rootName " + rootName + "creates binary with roster " + rosterList + "\n")

			}

		}

		w8.Flush()
		file8.Close()

		////dist := make(map[string]map[string]float64)
		////s.genTrees(RND_NODES, NR_LEVELS, OPTIMIZED, MIN_BUNCH_SIZE, OPTTYPE, dist)
	*/

	// create all rings you're part of
}
