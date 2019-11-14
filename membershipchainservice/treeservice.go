package membershipchainservice

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dedis/student_19_nyleCtrlPlane/gentree"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

const RND_NODES = false
const NR_LEVELS = 3
const OPTIMIZED = false
const OPTTYPE = 1
const MIN_BUNCH_SIZE = 39
const TREE_ID = 8

var execReqPingsMsgID network.MessageTypeID
var execReplyPingsMsgID network.MessageTypeID

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
		//log.Info(s.ServerIdentity(), fmt.Sprintf("%+v", nodes[gentree.NodeNameToInt(n.Name)]))
	}
	s.Nodes.All = nodes
	s.Nodes.ClusterBunchDistances = make(map[*gentree.LocalityNode]map[*gentree.LocalityNode]float64)
	s.Nodes.Links = make(map[*gentree.LocalityNode]map[*gentree.LocalityNode]map[*gentree.LocalityNode]bool)
	s.GraphTree = make(map[string][]GraphTree)
	s.BinaryTree = make(map[string][]*onet.Tree)
	s.OwnPings = make(map[string]float64)
	s.PingDistances = make(map[string]map[string]float64)

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

	file7, _ := os.Create("Specs/compact-pings" + s.Nodes.GetServerIdentityToName(s.ServerIdentity()))
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

func (s *Service) getPings(readFromFile bool) {
	if !readFromFile {
		// measure pings to other nodes
		s.measureOwnPings()
		s.DonePing = true

		s.PingMapMtx.Lock()
		for name, dist := range s.OwnPings {
			src := s.Nodes.GetServerIdentityToName(s.ServerIdentity())
			dst := name

			if _, ok := s.PingDistances[src]; !ok {
				s.PingDistances[src] = make(map[string]float64)
			}

			s.PingDistances[src][dst] = dist
			s.PingDistances[src][src] = 0.0
		}
		s.PingMapMtx.Unlock()

		log.LLvl1(s.Nodes.GetServerIdentityToName(s.ServerIdentity()), "finished ping own meas with len", len(s.OwnPings))

		// ask for pings from others
		for _, node := range s.Nodes.All {
			if node.Name != s.Nodes.GetServerIdentityToName(s.ServerIdentity()) {
				e := s.SendRaw(node.ServerIdentity, &ReqPings{SenderName: s.Nodes.GetServerIdentityToName(s.ServerIdentity())})
				log.LLvl1("\033[94m Error ? : ", e, "\033[39m ")
				if e != nil {
					panic(e)
				}
			}
		}

		// wit for ping replies from everyone but myself
		for s.NrPingAnswers != len(s.Nodes.All)-1 {
			log.LLvl1(" \033[32m WAITING ------------------------------------------ ", s.NrPingAnswers, len(s.Nodes.All)-1, "\033[39m ")
			time.Sleep(5 * time.Second)
		}

		// prints
		observerNode := s.Nodes.GetServerIdentityToName(s.ServerIdentity())
		pingDistStr := observerNode + "pingDistStr--------> "

		// divide all pings by 2
		/*
			for i := 0 ; i < 45 ; i++ {
				name1 := "node_" + strconv.Itoa(i)
				for j := 0; j < 45; j++ {
					name2 := "node_" + strconv.Itoa(j)
					s.PingDistances[name1][name2] = s.PingDistances[name1][name2] / 2.0
				}
			}
		*/

		for i := 0; i < 45; i++ {
			name1 := "node_" + strconv.Itoa(i)
			for j := 0; j < 45; j++ {
				name2 := "node_" + strconv.Itoa(j)
				pingDistStr += name1 + "-" + name2 + "=" + fmt.Sprintf("%f", s.PingDistances[name1][name2])
			}
			pingDistStr += "\n"
		}

		log.LLvl1(pingDistStr)

		// check that there are enough pings
		if len(s.PingDistances) < len(s.Nodes.All) {
			log.Lvl1(s.Nodes.GetServerIdentityToName(s.ServerIdentity()), " too few pings 1")
		}
		for _, m := range s.PingDistances {
			if len(m) < len(s.Nodes.All) {
				log.Lvl1(s.Nodes.GetServerIdentityToName(s.ServerIdentity()), " too few pings 2")
				log.LLvl1(m)
			}
		}

		log.LLvl1(s.Nodes.GetServerIdentityToName(s.ServerIdentity()), "has all pings, starting tree gen")

		// check TIV just once
		// painful, but run floyd warshall and build the static routes
		// ifrst, let's solve the k means mistery

		// ping node_0 node_1 = 19.314
		if s.Nodes.GetServerIdentityToName(s.ServerIdentity()) == "node_0" {
			file8, _ := os.Create("Specs/mypings.txt")
			w8 := bufio.NewWriter(file8)

			for n1, m := range s.PingDistances {
				for n2, d := range m {
					w8.WriteString("ping " + n1 + " " + n2 + " = " + fmt.Sprintf("%.2f", d) + "\n")
				}
			}
			w8.Flush()
			file8.Close()
		}
	} else {
		// read from file lines of fomrm "ping node_19 node_7 = 32.317"
		readLine, _ := ReadFileLineByLine("../utils/pings.txt")
		//readLine,_ := ReadFileLineByLine("shortest.txt")

		for true {
			line := readLine()
			if line == "" {
				break
			}

			if strings.HasPrefix(line, "#") {
				continue
			}

			tokens := strings.Split(line, " ")
			src := tokens[1]
			dst := tokens[2]
			pingTime, err := strconv.ParseFloat(tokens[4], 64)
			if err != nil {
				log.Error("Problem when parsing pings")
			}

			s.PingMapMtx.Lock()
			if _, ok := s.PingDistances[src]; !ok {
				s.PingDistances[src] = make(map[string]float64)
			}

			s.PingDistances[src][dst] = math.Round(pingTime*100) / 100
			log.LLvl1("PASSING PINGS.")
			s.PingMapMtx.Unlock()
		}
	}
}

func (s *Service) genTrees(RandomCoordsLevels bool, Levels int, Optimized bool, OptimisationLevel int, OptType int, pingDist map[string]map[string]float64) {

	// genTrees placeholder code, ideally we'll generate trees from small to large

	file3, _ := os.Create("Specs/gentree-" + s.Nodes.GetServerIdentityToName(s.ServerIdentity()))
	w3 := bufio.NewWriter(file3)

	gentree.CreateLocalityGraph(s.Nodes, RandomCoordsLevels, RandomCoordsLevels, Levels, pingDist, w3)
	myname := s.Nodes.GetServerIdentityToName(s.ServerIdentity())

	if Optimized {
		gentree.OptimizeGraph(s.Nodes, myname, OptimisationLevel, OptType)
	}

	//tree, NodesList, Parents, Distances := gentree.CreateOnetLPTree(s.Nodes, myname, OptimisationLevel)

	// route request to the roots of all rings i'm part of, using the distance oracles thingie

	// then everyone runs consensus in their trees

	dist2 := gentree.AproximateDistanceOracle(s.Nodes)

	log.Lvl1("andnowcomingdist2", dist2)

	file5, _ := os.Create("Specs/check-" + s.Nodes.GetServerIdentityToName(s.ServerIdentity()))
	w5 := bufio.NewWriter(file5)

	file6, _ := os.Create("Specs/sanity-" + s.Nodes.GetServerIdentityToName(s.ServerIdentity()))
	w6 := bufio.NewWriter(file6)

	file7, _ := os.Create("Specs/tiv-" + s.Nodes.GetServerIdentityToName(s.ServerIdentity()))
	w7 := bufio.NewWriter(file7)

	w3.WriteString("approximate-start\n")

	// loop in order
	for i := 0; i < len(dist2); i++ {
		name1 := "node_" + strconv.Itoa(i)
		for j := 0; j < len(dist2); j++ {
			name2 := "node_" + strconv.Itoa(j)
			compact := dist2[s.Nodes.GetByName(name1)][s.Nodes.GetByName(name2)]
			checkDist := s.ShortestDistances[name1][compact.ViaNodeName] + s.ShortestDistances[compact.ViaNodeName][name2]
			w3.WriteString(fmt.Sprintf("%.2f", compact.Dist) + " " + compact.ViaNodeName + " " + fmt.Sprintf("%.2f", checkDist) + " ")

			if math.Abs(compact.Dist-checkDist) > 0.9 {
				w5.WriteString(name1 + " " + name2 + " via " + compact.ViaNodeName + " should be " + fmt.Sprintf("%.2f", checkDist) + " is " + fmt.Sprintf("%.2f", compact.Dist) + "\n")
			}

			bound := 5 * s.ShortestDistances[name1][name2]
			if compact.Dist > bound {
				w6.WriteString(name1 + " " + name2 + " TOO LARGE, should be at most " + fmt.Sprintf("%.2f", bound) + "= 5 * " + fmt.Sprintf("%.2f", s.ShortestDistances[name1][name2]) + " is " + fmt.Sprintf("%.2f", compact.Dist) + "\n")
			}

			if s.ShortestDistances[name1][name2]+3.0 < s.PingDistances[name1][name2] {
				w7.WriteString(name1 + " " + name2 + " TIV shortest " + fmt.Sprintf("%.2f", s.ShortestDistances[name1][name2]) + " ping " + fmt.Sprintf("%.2f", s.PingDistances[name1][name2]) + "\n")
			}

		}
		w3.WriteString("\n")
	}

	w3.WriteString("approximate-end\n")

	w3.Flush()
	file3.Close()

	w5.Flush()
	file5.Close()

	w6.Flush()
	file6.Close()

	w7.Flush()
	file7.Close()

	// TODO we generate trees for all nodes
	for _, crtRoot := range s.Nodes.All {
		crtRootName := crtRoot.Name

		// here we should give the REAL distance from root because all nodes have REAL distances to their cluster!!! perhaps this is the difference
		tree, NodesList, Parents, TreeRadiuses := gentree.CreateOnetRings(s.Nodes, crtRootName, dist2)

		// update distances only if i'm the root
		if crtRootName == myname {
			s.Distances = dist2

			// just one of them prints
			/*
				if crtRootName == "node_0" {
					log.Lvl1("CHECK that distances make sense")
					for src, m := range dist2 {
						for dst, dist := range m {
							log.Lvl1("comparing for", src.Name, "-", dst.Name, "physical dist", gentree.ComputeDist(src, dst, pingDist), "approx dist", dist)
							if dist > 5*gentree.ComputeDist(src, dst, pingDist) {
								log.Lvl1("comparing for", src.Name, "-", dst.Name, "physical dist", gentree.ComputeDist(src, dst, pingDist), "approx dist", dist, "5x dist", 5*gentree.ComputeDist(src, dst, pingDist))
								log.Lvl1("WOAAAAA way too long!!!")
							}

						}

					}
				}
			*/

		}

		//s.GraphTree[crtRootName] = make([]GraphTree, 0)
		for i, n := range tree {
			s.GraphTree[crtRootName] = append(s.GraphTree[crtRootName], GraphTree{
				n,
				NodesList[i],
				Parents[i],
				TreeRadiuses[i],
			})
		}
	}

	//panic("")

	// send the graph trees to all nodes part of them

	//s.SendGraphTrees()

	for rootName, graphTrees := range s.GraphTree {
		for _, n := range graphTrees {

			rosterNames := make([]string, 0)
			rosterList := ""
			for _, si := range n.Tree.Roster.List {
				rosterNames = append(rosterNames, s.Nodes.GetServerIdentityToName(si))
				rosterList += s.Nodes.GetServerIdentityToName(si) + " "
			}

			// log.LLvl1("generation node ", s.Nodes.GetServerIdentityToName(s.ServerIdentity()), "rootName x", rootName, "creates binary with roster", rosterNames)
			// w7.WriteString("rootName " + rootName + "creates binary with roster " + rosterList + "\n")

			s.BinaryTree[rootName] = append(s.BinaryTree[rootName], s.CreateBinaryTreeFromGraphTree(n))
		}
	}

}
func (s *Service) floydWarshall() map[string]map[string]float64 {
	shortest := make(map[string]map[string]float64)
	for i := 0; i < len(s.Nodes.All); i++ {
		name := "node_" + strconv.Itoa(i)
		shortest[name] = make(map[string]float64)
	}

	for x, m := range s.PingDistances {
		for y, d := range m {
			shortest[x][y] = d
		}
	}

	for k := 0; k < len(s.Nodes.All); k++ {
		namek := "node_" + strconv.Itoa(k)
		for i := 0; i < len(s.Nodes.All); i++ {
			namei := "node_" + strconv.Itoa(i)
			for j := 0; j < len(s.Nodes.All); j++ {
				namej := "node_" + strconv.Itoa(j)
				if shortest[namei][namej] > shortest[namei][namek]+shortest[namek][namej] {
					shortest[namei][namej] = shortest[namei][namek] + shortest[namek][namej]
				}

			}
		}
	}
	return shortest
}

//Coumputes A Binary Tree Based On A Graph
func (s *Service) CreateBinaryTreeFromGraphTree(GraphTree GraphTree) *onet.Tree {

	BinaryTreeRoster := GraphTree.Tree.Roster
	Tree := BinaryTreeRoster.GenerateBinaryTree()

	return Tree
}

func ReadFileLineByLine(configFilePath string) (func() string, error) {
	f, err := os.Open(configFilePath)
	//defer close(f)

	if err != nil {
		return func() string { return "" }, err
	}
	checkErr(err)
	reader := bufio.NewReader(f)
	//defer close(reader)
	var line string
	return func() string {
		if err == io.EOF {
			return ""
		}
		line, err = reader.ReadString('\n')
		checkErr(err)
		line = strings.Split(line, "\n")[0]
		return line
	}, nil
}
func checkErr(e error) {
	if e != nil && e != io.EOF {
		fmt.Print(e)
		panic(e)
	}
}

func (s *Service) measureOwnPings() {
	myName := s.Nodes.GetServerIdentityToName(s.ServerIdentity())
	for _, node := range s.Nodes.All {

		if node.ServerIdentity.String() != s.ServerIdentity().String() {

			log.LLvl2(myName, "meas ping to ", s.Nodes.GetServerIdentityToName(node.ServerIdentity))

			for {
				peerName := node.Name
				pingCmdStr := "ping -W 150 -q -c 10 -i 1 " + node.ServerIdentity.Address.Host() + " | tail -n 1"
				pingCmd := exec.Command("sh", "-c", pingCmdStr)
				pingOutput, err := pingCmd.Output()

				if err != nil {
					log.Fatal("couldn't measure ping")
				}

				if strings.Contains(string(pingOutput), "pipe") || len(strings.TrimSpace(string(pingOutput))) == 0 {
					log.LLvl1(s.Nodes.GetServerIdentityToName(s.ServerIdentity()), "retrying for", peerName, node.ServerIdentity.Address.Host(), node.ServerIdentity.String())
					log.LLvl1("retry")
					continue
				}

				processPingCmdStr := "echo " + string(pingOutput) + " | cut -d ' ' -f 4 | cut -d '/' -f 1-2 | tr '/' ' '"
				processPingCmd := exec.Command("sh", "-c", processPingCmdStr)
				processPingOutput, _ := processPingCmd.Output()
				if string(pingOutput) == "" {
					log.Lvl1("empty ping ", myName+" "+peerName)
				} else {
					log.Lvl1("%%%%%%%%%%%%% ping ", myName+" "+peerName, "output ", string(pingOutput), "processed output ", string(processPingOutput))
				}

				log.Lvl1("%%%%%%%%%%%%% ping ", s.Nodes.GetServerIdentityToName(s.ServerIdentity())+" "+peerName, "output ", string(pingOutput), "processed output ", string(processPingOutput))

				strPingOut := string(processPingOutput)

				pingRes := strings.Split(strPingOut, "/")
				log.LLvl1("pingRes", pingRes)

				avgPing, err := strconv.ParseFloat(pingRes[5], 64)
				if err != nil {
					log.Fatal("Problem when parsing pings")
				}

				s.OwnPings[peerName] = float64(avgPing)

				log.LLvl1("~~~~~~~~~~~~~~", myName, "to", peerName, "is", avgPing)

				break
			}

		}
	}
}

func (s *Service) ExecReqPings(env *network.Envelope) error {
	log.LLvl1("EXEC REQ : ")
	// Parse message
	req, ok := env.Msg.(*ReqPings)
	if !ok {
		log.Error(s.ServerIdentity(), "failed to cast to ReqPings")
		return errors.New("failed to cast to ReplyPings")
	}

	// wait for pings to be finished
	for !s.DonePing {
		time.Sleep(5 * time.Second)
	}

	reply := ""
	myName := s.Nodes.GetServerIdentityToName(s.ServerIdentity())
	// build reply
	for peerName, pingTime := range s.OwnPings {
		//if peerName == myName {
		//	reply += myName + " " + peerName + " " + "0.0"
		//} else {
		reply += myName + " " + peerName + " " + fmt.Sprintf("%f", pingTime) + "\n"
		//}
	}

	log.LLvl3("\033[941  SENDING", reply, "\033[39m ")
	requesterIdentity := s.Nodes.GetByName(req.SenderName).ServerIdentity

	e := s.SendRaw(requesterIdentity, &ReplyPings{Pings: reply, SenderName: myName})
	if e != nil {
		panic(e)
	}
	return e
}

func (s *Service) ExecReplyPings(env *network.Envelope) error {
	log.LLvl1("\033[94m REPLY ?????????????????? :  \033[39m ")

	// Parse message
	req, ok := env.Msg.(*ReplyPings)
	if !ok {
		log.Error(s.ServerIdentity(), "failed to cast to ReplyPings")
		return errors.New("failed to cast to ReplyPings")
	}

	// process ping output
	//log.LLvl1("resp=", req.Pings)
	s.PingMapMtx.Lock()
	lines := strings.Split(req.Pings, "\n")
	for _, line := range lines {
		if line != "" {
			//log.LLvl1("line=", line)
			words := strings.Split(line, " ")
			src := words[0]
			dst := words[1]
			pingRes, err := strconv.ParseFloat(words[2], 64)
			if err != nil {
				log.Error("Problem when parsing pings")
			}

			if _, ok := s.PingDistances[src]; !ok {
				s.PingDistances[src] = make(map[string]float64)
			}

			//if _, ok := s.PingDistances[dst]; !ok {
			//	s.PingDistances[dst] = make(map[string]float64)
			//}

			s.PingDistances[src][dst] += pingRes
			//s.PingDistances[dst][src] += pingRes
			s.PingDistances[src][src] = 0.0

		}
	}
	s.PingMapMtx.Unlock()

	s.PingAnswerMtx.Lock()
	s.NrPingAnswers++
	s.PingAnswerMtx.Unlock()

	return nil
}
