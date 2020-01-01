package membershipchainservice

import (
	"strconv"
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
	nodes := make([]*gentree.LocalityNode, 1000)
	for i := range nodes {
		nodes[i] = &gentree.LocalityNode{}
	}

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
	nodes := make([]*gentree.LocalityNode, 1000)
	for i := range nodes {
		nodes[i] = &gentree.LocalityNode{}
	}

	for i := 0; i < 10; i++ {
		nodes[i].LotteryResult = float64(i) + 1.1
	}

	SetLevels(nodes)
	assert.Equal(t, 3, nodes[9].Level)
	for i := 0; i < 9; i++ {
		assert.Equal(t, 2, nodes[i].Level)
	}

}

func TestSetLevelsContinuity(t *testing.T) {
	nodes := make([]*gentree.LocalityNode, 6)
	for i := range nodes {
		nodes[i] = &gentree.LocalityNode{}
		nodes[i].Name = "node_" + strconv.Itoa(i)
	}

	SetLevels(nodes)

	maxLevel := 0
	levelsMap := make(map[int]int)
	for _, n := range nodes {
		levelsMap[n.Level]++
		if n.Level > maxLevel {
			maxLevel = n.Level
		}
	}

	for i := 0; i < maxLevel; i++ {
		assert.NotEqual(t, 0, levelsMap[i])
	}

}

func TestSetLevelsSmall(t *testing.T) {
	nodes := make([]*gentree.LocalityNode, 6)
	for i := range nodes {
		nodes[i] = &gentree.LocalityNode{}
		nodes[i].Name = "node_" + strconv.Itoa(i)
	}

	nodes[0].LotteryResult = 0.5660920659323543
	nodes[1].LotteryResult = 0.41765200380165207
	nodes[2].LotteryResult = 0.925128845219594
	nodes[3].LotteryResult = 0.42157058562840155

	SetLevels(nodes)

	maxLevel := 0
	levelsMap := make(map[int]int)
	for _, n := range nodes {
		levelsMap[n.Level]++
		if n.Level > maxLevel {
			maxLevel = n.Level
		}
	}

	for i := 0; i < maxLevel; i++ {
		assert.NotEqual(t, 0, levelsMap[i])
	}

}
