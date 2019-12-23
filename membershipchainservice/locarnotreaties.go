package membershipchainservice

import (
	"math"
	"math/rand"
	"sort"

	"github.com/dedis/student_19_nyleCtrlPlane/gentree"
	"go.dedis.ch/onet/v3/log"
)

const SHARED_RANDOM_SEED = 10

// SetLevels set the levels according to the algorithm defined in the part Locarno Treaties
// of the Report.
func SetLevels(nodes []gentree.LocalityNode) {
	nbNodes := len(nodes)
	rand.Seed(SHARED_RANDOM_SEED)
	probability := 1.0 / math.Pow(float64(nbNodes), 1.0/float64(NR_LEVELS))

	var lotteryResults []float64
	for i := 0; i < nbNodes; i++ {
		lotteryResults = append(lotteryResults, rand.Float64())
	}

	indexes := indexesOfSortedValues(lotteryResults)

	log.LLvl1(probability, lotteryResults, indexes)
	k := 0
	for l := NR_LEVELS - 1; l >= 0; l-- {
		nSelected := int(math.Pow(probability, float64(l)))
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
	log.LLvl1(indexes)
	log.LLvl1(copyList)
	return indexes
}

// IndexValue sort a list and keep track of the swap of indicies
type IndexValue struct {
	Indexes []int
	Values  []float64
}

func (iv IndexValue) Len() int           { return len(iv.Values) }
func (iv IndexValue) Less(i, j int) bool { return iv.Values[i] > iv.Values[j] }
func (iv IndexValue) Swap(i, j int) {
	iv.Indexes[i], iv.Indexes[j] = iv.Indexes[j], iv.Indexes[i]
	iv.Values[i], iv.Values[j] = iv.Values[j], iv.Values[i]
}
