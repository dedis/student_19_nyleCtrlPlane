package membershipchainservice

import (
	"testing"

	"github.com/dedis/student_19_nyleCtrlPlane/gentree"
	"github.com/stretchr/testify/assert"
)

func TestIndexesOfSortedValues(t *testing.T) {
	list := []float64{1.0, 2.0, 3.0}

	indexes := indexesOfSortedValues(list)
	expected := []int{2, 1, 0}
	assert.Equal(t, expected, indexes)
}

func TestSetLevels(t *testing.T) {
	nodes := make([]gentree.LocalityNode, 1000)

	SetLevels(nodes)

	levelsMap := make(map[int]int)
	for _, n := range nodes {
		levelsMap[n.Level]++
	}

	assert.Equal(t, 900, levelsMap[0])
	assert.Equal(t, 90, levelsMap[1])
	assert.Equal(t, 9, levelsMap[2])
	assert.Equal(t, 1, levelsMap[3])

}

func TestSetLevelsWithResults(t *testing.T) {
	nodes := make([]gentree.LocalityNode, 1000)

	for i := 0; i < 10; i++ {
		nodes[i].LotteryResult = float64(i) + 1.1
	}

	SetLevels(nodes)
	assert.Equal(t, 3, nodes[9].Level)
	for i := 0; i < 9; i++ {
		assert.Equal(t, 2, nodes[i].Level)
	}

}
