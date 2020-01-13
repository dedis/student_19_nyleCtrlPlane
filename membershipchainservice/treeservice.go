package membershipchainservice

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/dedis/student_19_nyleCtrlPlane/gentree"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

const USE_LOCARNO = false
const USE_SPACE_TIME = false
const NR_LEVELS = 3
const OPTIMIZED = false
const OPTTYPE = 1
const MIN_BUNCH_SIZE = 39
const TREE_ID = 8

var execReqPingsMsgID network.MessageTypeID
var execReplyPingsMsgID network.MessageTypeID

// Setup is a method that will initialize the Crux Protocol
// it is copy-pasted from : https://github.com/dedis/paper_crux/blob/master/dsn_exp/service/service.go
func (s *Service) Setup(req *InitRequest) {
	s.Nodes.All = make([]*gentree.LocalityNode, len(req.ServerIdentityToName))
	s.Nodes.ServerIdentityToName = make(map[network.ServerIdentityID]string)
	readNodePositionFromFile(s.Nodes.All, s.PrefixForReadingFile+"/utils/NodesFiles/nodes"+strconv.Itoa(len(s.Nodes.All))+".txt")

	for k, v := range req.ServerIdentityToName {
		s.Nodes.ServerIdentityToName[k.ID] = v
		s.Nodes.All[gentree.NodeNameToInt(v)].ServerIdentity = k
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
	}
	s.Nodes.All = nodes
	s.Nodes.ClusterBunchDistances = make(map[*gentree.LocalityNode]map[*gentree.LocalityNode]float64)
	s.Nodes.Links = make(map[*gentree.LocalityNode]map[*gentree.LocalityNode]map[*gentree.LocalityNode]bool)
	s.GraphTree = make(map[string][]GraphTree)
	s.BinaryTree = make(map[string][]*onet.Tree)
	s.OwnPings = make(map[string]float64)
	s.PingMapMtx.Lock()
	s.PingDistances = make(map[string]map[string]float64)
	s.PingMapMtx.Unlock()

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
		}
	}

	if USE_SPACE_TIME {
		s.GetInteractionDistances()
	} else {
		s.getPings(true)
	}
	if USE_LOCARNO {
		SetLevels(s.Nodes.All, s.getepochOfEntryMap())
	}

	str := s.Name + "\n"
	for _, n := range s.Nodes.All {
		str += n.Name + " -- " + strconv.Itoa(n.Level) + " -- " + fmt.Sprintf("%v,%v", n.X, n.Y) + " \n"
	}
	//log.LLvl1(str)

	// If it does not use Locarno Treaties for generating the levels it has to draw them randomly
	// If one wants to modify this code to read levels from a file one might have a look to Maxime Sierro Code : https://github.com/dedis/student_19_nylechain
	s.genTrees(!USE_LOCARNO, NR_LEVELS, OPTIMIZED, MIN_BUNCH_SIZE, OPTTYPE, s.PingDistances)

	s.ShortestDistances = s.floydWarshall()

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

		// wait for ping replies from everyone but myself
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

		if len(s.Nodes.All) > 100 {
			panic("This file was not generated")
		}
		// read from file lines of form "ping node_19 node_7 = 32.317"
		readLine, err := ReadFileLineByLine(s.PrefixForReadingFile + "/utils/PingsFiles/pings" + strconv.Itoa(len(s.Nodes.All)) + ".txt")
		if err != nil {
			panic(fmt.Sprintf("Cannot read file for ping /utils/PingsFiles/pings%v", len(s.Nodes.All)))
		}

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
			s.PingMapMtx.Unlock()
		}
	}
}

func (s *Service) genTrees(RandomLevels bool, Levels int, Optimized bool, OptimisationLevel int, OptType int, pingDist map[string]map[string]float64) {
	folderStr := "Data/"
	if RandomLevels {
		folderStr += "Random/"
	} else {
		folderStr += "Locarno/"
	}

	file3, _ := os.Create(folderStr + "gentree-" + s.Nodes.GetServerIdentityToName(s.ServerIdentity()) + "-epoch" + strconv.Itoa(int(s.e)))
	w3 := bufio.NewWriter(file3)
	w3.WriteString("Name,Level,X,Y,cluster,bunch\n")

	// To do a proper comparison, levels should be generated randomly at each epoch, but nodes keep their positions (read from file)
	gentree.CreateLocalityGraph(s.Nodes, false, RandomLevels, Levels, pingDist, w3)
	myname := s.Nodes.GetServerIdentityToName(s.ServerIdentity())

	if Optimized {
		gentree.OptimizeGraph(s.Nodes, myname, OptimisationLevel, OptType)
	}

	dist2 := gentree.AproximateDistanceOracle(s.Nodes)

	for _, crtRoot := range s.Nodes.All {
		crtRootName := crtRoot.Name

		// here we should give the REAL distance from root because all nodes have REAL distances to their cluster!!! perhaps this is the difference
		tree, NodesList, Parents, TreeRadiuses := gentree.CreateOnetRings(s.Nodes, crtRootName, dist2)

		// update distances only if i'm the root
		if crtRootName == myname {
			s.Distances = dist2
		}
		for i, n := range tree {
			s.GraphTree[crtRootName] = append(s.GraphTree[crtRootName], GraphTree{
				n,
				NodesList[i],
				Parents[i],
				TreeRadiuses[i],
			})
		}
	}
	for rootName, graphTrees := range s.GraphTree {
		for _, n := range graphTrees {

			rosterNames := make([]string, 0)
			rosterList := ""
			for _, si := range n.Tree.Roster.List {
				rosterNames = append(rosterNames, s.Nodes.GetServerIdentityToName(si))
				rosterList += s.Nodes.GetServerIdentityToName(si) + " "
			}
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
	requesterIdentity := s.Nodes.GetByName(req.SenderName).ServerIdentity

	e := s.SendRaw(requesterIdentity, &ReplyPings{Pings: reply, SenderName: myName})
	if e != nil {
		panic(e)
	}
	return e
}

func (s *Service) ExecReplyPings(env *network.Envelope) error {

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

func readNodePositionFromFile(Nodes []*gentree.LocalityNode, filename string) {

	// read from file lines of fomrm "nodes_x X Y"
	readLine, err := ReadFileLineByLine(filename)
	//readLine,_ := ReadFileLineByLine("shortest.txt")
	if err != nil {
		panic("Cannot read file for nodes " + filename)
	}

	i := 0
	for true {
		line := readLine()
		if line == "" {
			break
		}

		if strings.HasPrefix(line, "#") {
			continue
		}

		tokens := strings.Split(line, " ")
		name := tokens[0]
		X, err := strconv.ParseFloat(tokens[1], 64)
		if err != nil {
			log.Error("Problem when parsing positions")
		}
		Y, err := strconv.ParseFloat(tokens[2], 64)
		if err != nil {
			log.Error("Problem when parsing positions")
		}
		Nodes[i] = &gentree.LocalityNode{}
		Nodes[i].Name = name
		Nodes[i].X = X
		Nodes[i].Y = Y
		i++
	}
}
