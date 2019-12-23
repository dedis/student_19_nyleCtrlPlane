package main

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"go.dedis.ch/onet/v3/log"
)

type NodeFloat struct {
	Name    string
	X       float64
	Y       float64
	Level   int
	Links   map[string]bool
	IP      map[string]bool
	cluster map[string]bool
	ADist   []float64
	pDist   []string
}

type DeterNode struct {
	Name  string
	Level int
	Links map[string]bool
	Dist  map[string]int
	IP    []string
}

type deterNodes []NodeFloat

var MAX_IFACES = 4

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

func euclidianDistNew(a NodeFloat, b NodeFloat) float64 {
	return math.Sqrt(math.Pow(float64(a.X-b.X), 2.0) + math.Pow(float64(a.Y-b.Y), 2.0))
}

func computeADistNew(nodes []NodeFloat, K int) {
	for lvl := K - 1; lvl >= 0; lvl-- {
		N := len(nodes)
		for i := 0; i < N; i++ {
			crtNode := &nodes[i]

			for j := 0; j < N; j++ {
				dist := euclidianDistNew(nodes[i], nodes[j])
				if nodes[j].Level >= lvl && dist < crtNode.ADist[lvl] {
					crtNode.ADist[lvl] = dist
					crtNode.pDist[lvl] = nodes[j].Name
				}
			}
			crtNode.ADist[0] = 0
			crtNode.pDist[0] = crtNode.Name

			if crtNode.ADist[lvl] == crtNode.ADist[lvl+1] {
				crtNode.pDist[lvl] = crtNode.pDist[lvl+1]
			}

		}

		// iterate all nodes at level i but not i + 1
		for i := 0; i < N; i++ {
			crtNode := &nodes[i]
			if crtNode.Level == lvl {
				for j := 0; j < N; j++ {
					dist := euclidianDistNew(nodes[i], nodes[j])
					if dist < crtNode.ADist[lvl+1] {
						crtNode.cluster[nodes[j].Name] = true
					}
				}
			}
		}

	}
}

func genAndPrintNodes(N int, S int, SpaceMax int, K int, zeroY bool, trueLatency bool) {

	var RndSrc *rand.Rand
	RndSrc = rand.New(rand.NewSource(time.Now().UnixNano()))
	nodes := make([]NodeFloat, S)

	for {
		// generate coordinates for the physical nodes
		for i := 0; i < N; i++ {
			nodes[i].Name = "node_" + strconv.Itoa(i)
			/*nodes[i].X = rand.Float64() * float64(SpaceMax)

			nodes[i].X = float64(rand.Intn(5))+2
			if i > 0 {
				nodes[i].X += nodes[i-1].X
			}
			*/
			nodes[i].X = 0
			if i > 0 {
				nodes[i].X = nodes[i-1].X + 2
			}

			if !zeroY {
				nodes[i].Y = rand.Float64() * float64(SpaceMax)
			} else {
				nodes[i].Y = 0
			}
			nodes[i].Links = make(map[string]bool)
			nodes[i].IP = make(map[string]bool)
			nodes[i].cluster = make(map[string]bool)
			nodes[i].ADist = make([]float64, K+1)
			nodes[i].Level = 0
			for j := 0; j <= K; j++ {
				nodes[i].ADist[j] = math.MaxInt64
			}
			nodes[i].pDist = make([]string, K+1)
		}

		// copy these over as many times as needed for the other collocated nodes within each site
		for i := N; i < S; i++ {
			nodes[i].Name = "node_" + strconv.Itoa(i)

			// dummy values so that they don't interfere with the sorting
			nodes[i].X = nodes[i%N].X
			//nodes[i].X = rand.Float64() * float64(SpaceMax)
			nodes[i].Y = nodes[i%N].Y
			nodes[i].Level = 0

			nodes[i].Links = make(map[string]bool)
			nodes[i].IP = make(map[string]bool)
			nodes[i].cluster = make(map[string]bool)
			nodes[i].ADist = make([]float64, K+1)
			for j := 0; j <= K; j++ {
				nodes[i].ADist[j] = math.MaxInt64
			}
			nodes[i].pDist = make([]string, K+1)
		}

		if nodes[S-1].X > float64(SpaceMax) {
			log.Fatal("Space max!")
		}

		prob := 1.0 / math.Pow(float64(S), 1.0/float64(K))

		for lvl := 0; lvl < K; lvl++ {
			for i := 0; i < S; i++ {
				if nodes[i].Level == lvl-1 {
					rnd := RndSrc.Float64()
					if rnd < prob {
						nodes[i].Level = lvl
					}
				}
			}
		}

		computeADistNew(nodes, K)
		cluster3 := 0
		cluster1 := 0
		cluster2 := 0

		for i := 0; i < S; i++ {
			//log.Lvl1(nodes[i].Name, nodes[i].cluster)
			if len(nodes[i].cluster) == 1 || len(nodes[i].cluster) == 2 {
				cluster3++
				if len(nodes[i].cluster) == 1 {
					cluster1++
				}
				if len(nodes[i].cluster) == 2 {
					cluster2++
				}
			}

		}

		log.LLvl1("clusters3=", cluster3, cluster1, cluster2)

		//if cluster3 == 0 {
		break
		//}

		/*
			isBreak := false

			if cluster3 == 0 {
				isBreak = true

				sort.Sort(deterNodes(nodes))
				for i := 0; i < S; i++ {
					ireal := i % N
					iplusreal := (i + 1) % N

					dist := int(math.Ceil(math.Abs(nodes[ireal].X - nodes[iplusreal].X)))
					if dist <= 2 {
						isBreak = false
						log.LLvl1(nodes[ireal].X, nodes[iplusreal].X)
					}
				}
			}

			if isBreak {
				break
			}
		*/
	}

	// check that cluster size is either 0 or bigger than 3

	// generateNSFILE

	//for i := N; i < S; i++ {

	// dummy values so that they don't interfere with the sorting
	//nodes[i].X = math.MaxFloat64
	//nodes[i].Y = math.MaxFloat64
	//}

	// arrange nodes by x coordinate
	//sort.Sort(deterNodes(nodes))

	for i := 0; i < S; i++ {
		log.LLvl1(nodes[i].Name, nodes[i].cluster)
	}

	// rename for ease of stuff
	for i := 0; i < N; i++ {
		nodes[i].Name = "node_" + strconv.Itoa(i)
	}

	for i := N; i < S; i++ {
		nodes[i].Name = "node_" + strconv.Itoa(i)
	}

	file2, _ := os.Create("delay.ns")
	defer file2.Close()
	w2 := bufio.NewWriter(file2)

	w2.WriteString("set ns [new Simulator]\n")
	w2.WriteString("source tb_compat.tcl\n")
	w2.WriteString("\n")
	w2.WriteString("tb-use-endnodeshaping 1\n")
	w2.WriteString("set n_nodes " + strconv.Itoa(N) + "\n")
	w2.WriteString("\n")
	w2.WriteString("for {set i 0} {$i < $n_nodes} {incr i} {\n")
	w2.WriteString("\tset site($i) [$ns node]\n")
	w2.WriteString("\ttb-set-hardware $site($i) {MicroCloud}\n")
	w2.WriteString("\ttb-set-node-os $site($i) Ubuntu1404-64-STD\n")
	w2.WriteString("}\n\n")

	// in the NS file connect just the first N nodes, because these are the physical ones

	// we first connect nodes that are next to each other
	//// -> for i := 0 ; i < N-1 ; i++ {
	for i := 0; i < 2*N; i++ {
		ireal := i % N
		iplusreal := (i + 1) % N

		idx1 := strings.Split(nodes[ireal].Name, "_")[1]
		idx2 := strings.Split(nodes[iplusreal].Name, "_")[1]

		dist := 0

		connectToNextNext := false

		if trueLatency {
			dist = int(math.Ceil(math.Abs(nodes[ireal].X - nodes[iplusreal].X)))
			if dist < 2 {

				connectToNextNext = true
				// TODO
				// distance < 2, connect to next next node

				log.Fatal("Shouldn't happen!")
			}
			/*
					// put the nodes farther apart
					// generate a random offset from previous node between 1 and 2
					rndVal := rand.Float64() * float64(1) + 1
					nodes[iplusreal].X = nodes[ireal].X + rndVal

					// move all subsequent nodes!
					for j := iplusreal + 1 ; j < N ; j++ {
						nodes[j].X += rndVal
					}

					dist = int(math.Ceil(math.Abs(nodes[ireal].X - nodes[iplusreal].X)))

				}
			*/
		}

		if !connectToNextNext {

			w2.WriteString("set link" + strconv.Itoa(i) + " [$ns duplex-link $site(" + idx1 + ") $site(" + idx2 + ") 100Mb " + strconv.Itoa(dist) + "ms DropTail]\n")
			w2.WriteString("tb-set-ip-link $site(" + idx1 + ") $link" + strconv.Itoa(i) + " 10.1." + strconv.Itoa(i+1) + ".2\n")
			w2.WriteString("tb-set-ip-link $site(" + idx2 + ") $link" + strconv.Itoa(i) + " 10.1." + strconv.Itoa(i+1) + ".3\n")
			nodes[i].Links[nodes[(i+1)%S].Name] = true
			nodes[(i+1)%(2*N)].Links[nodes[i].Name] = true

			nodes[i].IP["10.1."+strconv.Itoa(i+1)+".2"] = true
			nodes[(i+1)%(2*N)].IP["10.1."+strconv.Itoa(i+1)+".3"] = true
		} else {

			iplusreal = (i + 2) % N
			idx2 = strings.Split(nodes[iplusreal].Name, "_")[1]

			log.LLvl1("LHALIHLIHSLIH", idx1, idx2)

			w2.WriteString("set link" + strconv.Itoa(i) + " [$ns duplex-link $site(" + idx1 + ") $site(" + idx2 + ") 100Mb " + strconv.Itoa(dist) + "ms DropTail]\n")
			w2.WriteString("tb-set-ip-link $site(" + idx1 + ") $link" + strconv.Itoa(i) + " 10.1." + strconv.Itoa(i+1) + ".2\n")
			w2.WriteString("tb-set-ip-link $site(" + idx2 + ") $link" + strconv.Itoa(i) + " 10.1." + strconv.Itoa(i+1) + ".4\n")
			nodes[i].Links[nodes[(i+2)%S].Name] = true
			nodes[(i+2)%S].Links[nodes[i].Name] = true

			nodes[i].IP["10.1."+strconv.Itoa(i+1)+".2"] = true
			nodes[(i+2)%S].IP["10.1."+strconv.Itoa(i+1)+".4"] = true
		}

		/*
			if nodes[i].IP == "" {
				nodes[i].IP = "10.1." + strconv.Itoa(i+1) + ".2"
			}
			if nodes[i+1].IP == "" {
				nodes[i+1].IP = "10.1." + strconv.Itoa(i+1) + ".3"
			}
		*/
	}

	nextLinkNr := N - 1
	// for each node, generate 3 random neighbors

	maxNeighbors := 4
	minNeighbors := S / N
	if S%N != 0 {
		minNeighbors += 1
	}

	for i := 0; i < N; i++ {
		// generate random node
		log.LLvl1("i=", i)
		attempts := 0
		for {
			attempts++

			if len(nodes[i].Links) >= minNeighbors || attempts > 1000 {
				log.LLvl1("going to next one")
				break
			}
			neighIdx := rand.Intn(N)
			if neighIdx == i {
				continue
			}

			//log.LLvl1("neighIdx=",neighIdx )

			if len(nodes[neighIdx].Links) == maxNeighbors {
				continue
			}
			if nodes[i].Links[nodes[neighIdx].Name] == true {
				continue
			}

			//log.LLvl1("nextLinkNr=", nextLinkNr )

			// add
			idx1 := strings.Split(nodes[i].Name, "_")[1]
			idx2 := strings.Split(nodes[neighIdx].Name, "_")[1]
			dist := 0

			if trueLatency {
				dist = int(math.Ceil(math.Abs(nodes[i].X - nodes[neighIdx].X)))
			}

			log.LLvl1("HERE")
			w2.WriteString("set link" + strconv.Itoa(nextLinkNr) + " [$ns duplex-link $site(" + idx1 + ") $site(" + idx2 + ") 100Mb " + strconv.Itoa(dist) + "ms DropTail]\n")
			w2.WriteString("tb-set-ip-link $site(" + idx1 + ") $link" + strconv.Itoa(nextLinkNr) + " 10.1." + strconv.Itoa(nextLinkNr+1) + ".2\n")
			w2.WriteString("tb-set-ip-link $site(" + idx2 + ") $link" + strconv.Itoa(nextLinkNr) + " 10.1." + strconv.Itoa(nextLinkNr+1) + ".3\n")
			nodes[i].Links[nodes[neighIdx].Name] = true
			nodes[neighIdx].Links[nodes[i].Name] = true

			nodes[i].IP["10.1."+strconv.Itoa(nextLinkNr+1)+".2"] = true
			nodes[neighIdx].IP["10.1."+strconv.Itoa(nextLinkNr+1)+".3"] = true

			//if nodes[i].IP == "" {
			//	nodes[i].IP = "10.1." + strconv.Itoa(nextLinkNr+1) + ".2"
			//}
			//if nodes[neighIdx].IP == "" {
			//	nodes[neighIdx].IP = "10.1." + strconv.Itoa(nextLinkNr+1) + ".3"
			//}

			nextLinkNr++

		}
	}

	w2.WriteString("\n")
	// the LAN stuff adds one other IP addr to each machine that we cannot easily predict, for real world experiments we don't need that one
	// w2.WriteString("set lan0 [$ns make-lan \"$site(0) $site(1) $site(2) $site(3) $site(4) $site(5) $site(6) $site(7) $site(8) $site(9) $site(10) $site(11) $site(12) $site(13) $site(14) $site(15) $site(16) $site(17) $site(18) $site(19) $site(20) $site(21) $site(22) $site(23) $site(24) $site(25) $site(26) $site(27) $site(28) $site(29) $site(30) $site(31) $site(32) $site(33) $site(34) $site(35) $site(36) $site(37) $site(38) $site(39) $site(40) $site(41) $site(42) $site(43) $site(44) \" 100Mb 0ms]\n")
	//w2.WriteString("\n")
	w2.WriteString("$ns rtproto Static\n")
	w2.WriteString("\n")
	w2.WriteString("$ns run")

	w2.Flush()

	file, _ := os.Create("nodes" + strconv.Itoa(N) + ".txt")
	defer file.Close()
	w := bufio.NewWriter(file)

	// copy IP addresses from physical nodes to logical nodes

	for i := 0; i < N; i++ {
		for k, v := range nodes[i].IP {
			j := i
			for j < S {
				nodes[j].IP[k] = v
				nodes[j].X = nodes[i].X
				nodes[j].Y = nodes[i].Y
				j += N
			}
		}
	}

	for i := N; i < 2*N; i++ {
		for k, v := range nodes[i].IP {
			j := i
			for j < S {
				nodes[j].IP[k] = v
				nodes[j].X = nodes[i].X
				nodes[j].Y = nodes[i].Y
				j += N
			}
		}
	}

	// print nodes in the out experiment file
	for i := 0; i < S; i++ {
		xFloat := fmt.Sprintf("%f", nodes[i].X)
		yFloat := fmt.Sprintf("%f", nodes[i].Y)
		//w.WriteString(nodes[i].Name + " " + xFloat + "," + yFloat + " 127.0.0.1 " + strconv.Itoa(nodes[i].Level) + "\n")

		ips := ""

		if i >= N {
			for ip, exists := range nodes[i%N].IP {
				if exists {
					ips += ip + ","
				}
			}

			for ip, exists := range nodes[i].IP {
				if exists && !nodes[i%N].IP[ip] {
					ips += ip + ","
				}
			}

		} else {

			for ip, exists := range nodes[i].IP {
				if exists {
					ips += ip + ","
				}
			}
		}

		w.WriteString(nodes[i].Name + " " + xFloat + "," + yFloat + " " + ips + " " + strconv.Itoa(nodes[i].Level) + "\n")
	}

	w.Flush()
}

func (s deterNodes) Len() int {
	return len(s)
}
func (s deterNodes) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s deterNodes) Less(i, j int) bool {
	return s[i].X < s[j].X
}

func genNsFile(infile string, ourfile string) {

	readLine, _ := ReadFileLineByLine(infile)

	for true {
		line := readLine()
		//fmt.Println(line)
		if line == "" {
			//fmt.Println("end")
			break
		}

		if strings.HasPrefix(line, "#") {
			continue
		}

	}
}

func genAndPrintRndNodes(N int, S int, SpaceMax int, K int, zeroY bool, trueLatency bool) {

	var RndSrc *rand.Rand
	RndSrc = rand.New(rand.NewSource(time.Now().UnixNano()))
	nodes := make([]DeterNode, S)

	for i := 0; i < N; i++ {
		nodes[i].Name = "node_" + strconv.Itoa(i)
		nodes[i].Links = make(map[string]bool)
		nodes[i].Dist = make(map[string]int)
		nodes[i].IP = make([]string, 0)
		nodes[i].Level = 0
	}

	prob := 1.0 / math.Pow(float64(S), 1.0/float64(K))
	for lvl := 0; lvl < K; lvl++ {
		for i := 0; i < N; i++ {
			if nodes[i].Level == lvl-1 {
				rnd := RndSrc.Float64()
				if rnd < prob {
					nodes[i].Level = lvl
				}
			}
		}
	}

	// copy these over as many times as needed for the other collocated nodes within each site
	for i := N; i < S; i++ {
		nodes[i].Name = "node_" + strconv.Itoa(i)
		nodes[i].Level = nodes[i%N].Level
		nodes[i].Links = make(map[string]bool)
		nodes[i].IP = make([]string, 0)
	}

	// generateNSFILE

	//for i := N; i < S; i++ {

	// dummy values so that they don't interfere with the sorting
	//nodes[i].X = math.MaxFloat64
	//nodes[i].Y = math.MaxFloat64
	//}

	// arrange nodes by x coordinate
	//sort.Sort(deterNodes(nodes))

	file2, _ := os.Create("delay.ns")
	defer file2.Close()
	w2 := bufio.NewWriter(file2)

	w2.WriteString("set ns [new Simulator]\n")
	w2.WriteString("source tb_compat.tcl\n")
	w2.WriteString("\n")
	w2.WriteString("tb-use-endnodeshaping 1\n")
	w2.WriteString("set n_nodes " + strconv.Itoa(N) + "\n")
	w2.WriteString("\n")
	w2.WriteString("for {set i 0} {$i < $n_nodes} {incr i} {\n")
	w2.WriteString("\tset site($i) [$ns node]\n")
	w2.WriteString("\ttb-set-hardware $site($i) {MicroCloud}\n")
	w2.WriteString("\ttb-set-node-os $site($i) Ubuntu1404-64-STD\n")
	w2.WriteString("}\n\n")

	// in the NS file connect just the first N nodes, because these are the physical ones

	// make sure the network is connected

	linkNr := 0

	for i := 0; i < N; i++ {
		peerIdx := (i + 1) % N
		peerName := nodes[peerIdx].Name

		idx1 := strconv.Itoa(i)
		idx2 := strconv.Itoa(peerIdx)

		// generate random latency in 2-10 ms
		min := 0
		dist := RndSrc.Intn(9) + 2

		l := ""

		// check if triangle inequality stuff
		for j := 0; j < N; j++ {
			// do they connect through j? then add that as constraint
			nameJ := nodes[j].Name
			if nodes[i].Links[nameJ] && nodes[peerIdx].Links[nameJ] {
				min = nodes[i].Dist[nameJ] + nodes[peerIdx].Dist[nameJ]
				l += nameJ
			}
		}

		if min != 0 {
			log.LLvl1(i, peerIdx, "have min", min, "through node", l)
			dist = min
		}

		w2.WriteString("set link" + strconv.Itoa(linkNr) + " [$ns duplex-link $site(" + idx1 + ") $site(" + idx2 + ") 100Mb " + strconv.Itoa(dist) + "ms DropTail]\n")
		w2.WriteString("tb-set-ip-link $site(" + idx1 + ") $link" + strconv.Itoa(linkNr) + " 10.1." + strconv.Itoa(linkNr+1) + ".2\n")
		w2.WriteString("tb-set-ip-link $site(" + idx2 + ") $link" + strconv.Itoa(linkNr) + " 10.1." + strconv.Itoa(linkNr+1) + ".3\n")
		nodes[i].Links[peerName] = true
		nodes[peerIdx].Links[nodes[i].Name] = true
		nodes[i].Dist[peerName] = dist
		nodes[peerIdx].Dist[nodes[i].Name] = dist

		nodes[i].IP = append(nodes[i].IP, "10.1."+strconv.Itoa(linkNr+1)+".2")
		nodes[peerIdx].IP = append(nodes[peerIdx].IP, "10.1."+strconv.Itoa(linkNr+1)+".3")

		log.LLvl1(i, peerIdx, dist)

		linkNr++
	}

	/// generate rest of neighbors randomly
	for i := 0; i < N; i++ {
		attempts := 0
		for {

			log.LLvl1("node", i, "attempts", attempts)
			if len(nodes[i].IP) == MAX_IFACES || attempts == 100 {
				break
			}
			// generate a random node to connect to
			peerIdx := RndSrc.Intn(N)
			peerName := nodes[peerIdx].Name

			for {
				if (peerIdx != i && !nodes[i].Links[peerName] && len(nodes[peerIdx].IP) < MAX_IFACES) || attempts == 100 {
					break
				}
				peerIdx = RndSrc.Intn(N)
				peerName = nodes[peerIdx].Name
				attempts++
			}

			if attempts == 100 {
				continue
			}

			// connect the two nodes

			idx1 := strconv.Itoa(i)
			idx2 := strconv.Itoa(peerIdx)

			// generate random latency in 2-10 ms
			min := 0
			dist := RndSrc.Intn(9) + 2

			l := ""

			// check if triangle inequality stuff
			for j := 0; j < N; j++ {
				// do they connect through j? then add that as constraint
				nameJ := nodes[j].Name
				if nodes[i].Links[nameJ] && nodes[peerIdx].Links[nameJ] {
					min = nodes[i].Dist[nameJ] + nodes[peerIdx].Dist[nameJ]
					l += nameJ
				}
			}

			if min != 0 {
				log.LLvl1(i, peerIdx, "have min", min, "through node", l)
				dist = min
			}

			w2.WriteString("set link" + strconv.Itoa(linkNr) + " [$ns duplex-link $site(" + idx1 + ") $site(" + idx2 + ") 100Mb " + strconv.Itoa(dist) + "ms DropTail]\n")
			w2.WriteString("tb-set-ip-link $site(" + idx1 + ") $link" + strconv.Itoa(linkNr) + " 10.1." + strconv.Itoa(linkNr+1) + ".2\n")
			w2.WriteString("tb-set-ip-link $site(" + idx2 + ") $link" + strconv.Itoa(linkNr) + " 10.1." + strconv.Itoa(linkNr+1) + ".3\n")
			nodes[i].Links[peerName] = true
			nodes[peerIdx].Links[nodes[i].Name] = true
			nodes[i].Dist[peerName] = dist
			nodes[peerIdx].Dist[nodes[i].Name] = dist

			nodes[i].IP = append(nodes[i].IP, "10.1."+strconv.Itoa(linkNr+1)+".2")
			nodes[peerIdx].IP = append(nodes[peerIdx].IP, "10.1."+strconv.Itoa(linkNr+1)+".3")

			log.LLvl1(i, peerIdx, dist)

			linkNr++

		}
	}

	w2.WriteString("\n")
	// the LAN stuff adds one other IP addr to each machine that we cannot easily predict, for real world experiments we don't need that one
	// w2.WriteString("set lan0 [$ns make-lan \"$site(0) $site(1) $site(2) $site(3) $site(4) $site(5) $site(6) $site(7) $site(8) $site(9) $site(10) $site(11) $site(12) $site(13) $site(14) $site(15) $site(16) $site(17) $site(18) $site(19) $site(20) $site(21) $site(22) $site(23) $site(24) $site(25) $site(26) $site(27) $site(28) $site(29) $site(30) $site(31) $site(32) $site(33) $site(34) $site(35) $site(36) $site(37) $site(38) $site(39) $site(40) $site(41) $site(42) $site(43) $site(44) \" 100Mb 0ms]\n")
	//w2.WriteString("\n")
	w2.WriteString("$ns rtproto Static\n")
	w2.WriteString("\n")
	w2.WriteString("$ns run")

	w2.Flush()

	file, _ := os.Create("node.txt")
	defer file.Close()
	w := bufio.NewWriter(file)

	// peprend IP addresses from physical nodes to logical nodes

	for i := 0; i < N; i++ {
		j := i + N
		for j < S {
			nodes[j].IP = append(nodes[i].IP, nodes[j].IP...)
			j += N
		}
	}

	// print nodes in the out experiment file
	for i := 0; i < S; i++ {
		ips := ""
		for _, ip := range nodes[i].IP {
			ips += ip + ","
		}

		w.WriteString(nodes[i].Name + " " + ips + " " + strconv.Itoa(nodes[i].Level) + "\n")
	}

	w.Flush()
}

func main() {

	const numLevel = 3
	const spaceMax = 100
	const S = 135

	for n := 2; n < 50; n++ {
		genAndPrintRndNodes(n, S, spaceMax, numLevel, true, true)
	}

}
