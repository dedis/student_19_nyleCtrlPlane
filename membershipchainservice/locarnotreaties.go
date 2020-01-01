package membershipchainservice

import (
	"fmt"
	"math"
	"math/rand"
	"sort"

	"github.com/dedis/student_19_nyleCtrlPlane/gentree"
	"go.dedis.ch/onet/v3/log"
)

const SHARED_SOURCE_OF_RANDOMNESS = 10

// SetLevels set the levels according to the algorithm defined in the part Locarno Treaties
// of the Report.
func SetLevels(nodes []*gentree.LocalityNode, epochOfEntryMap map[string]Epoch) {
	log.LLvl1(epochOfEntryMap)
	str := "\n"
	for i, n := range nodes {
		str += fmt.Sprintf("%v, %v {L : %v, Entry : %v  - %v,%v}", i, n.Name, n.Level, epochOfEntryMap[n.Name], n.X, n.Y) + "\n"
	}
	log.LLvl1(str)
	nbNodes := len(nodes)
	// TODO : verify Other method
	probability := 1.0 / math.Pow(float64(nbNodes), 1.0/float64(NR_LEVELS-1))

	var lotteryResults []float64
	for _, n := range nodes {
		randSrc := rand.New(getSource(n.Name, epochOfEntryMap[n.Name], SHARED_SOURCE_OF_RANDOMNESS))
		lotteryResults = append(lotteryResults, randSrc.Float64())
	}

	indexes := indexesOfSortedValues(lotteryResults)
	k := 0
	for l := 0; l < NR_LEVELS; l++ {
		reduceFactor := 1.0
		if l != NR_LEVELS {
			reduceFactor = 1 - probability
		}

		nSelected := int(math.Round(reduceFactor * math.Pow(probability, float64(l)) * float64(nbNodes)))
		for i := 0; i < nSelected; i++ {
			nodes[indexes[k]].Level = l
			k++
		}
	}

}

func indexesOfSortedValues(list []float64) []int {
	indexes := make([]int, len(list))
	copyList := make([]float64, len(list))
	copy(copyList, list)
	for i := range indexes {
		indexes[i] = i
	}
	sort.Sort(IndexValue{Indexes: indexes, Values: copyList})
	return indexes
}

func getSource(name string, e Epoch, sharedSource int64) rand.Source {
	return rand.NewSource(int64(gentree.NodeNameToInt(name)) + int64(e) + sharedSource)
}

// IndexValue sort a list and keep track of the swap of indicies
type IndexValue struct {
	Indexes []int
	Values  []float64
}

func (iv IndexValue) Len() int           { return len(iv.Values) }
func (iv IndexValue) Less(i, j int) bool { return iv.Values[i] < iv.Values[j] }
func (iv IndexValue) Swap(i, j int) {
	iv.Indexes[i], iv.Indexes[j] = iv.Indexes[j], iv.Indexes[i]
	iv.Values[i], iv.Values[j] = iv.Values[j], iv.Values[i]
}
