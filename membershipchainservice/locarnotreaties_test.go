package membershipchainservice

import (
	"strconv"
	"testing"

	"github.com/dedis/student_19_nyleCtrlPlane/gentree"
	"github.com/stretchr/testify/assert"
)

func TestIndexesOfSortedValues(t *testing.T) {
	list := []float64{3.0, 2.0, 1.0}

	indexes := indexesOfSortedValues(list)
	expected := []int{2, 1, 0}
	assert.Equal(t, expected, indexes)
}

func TestSetLevels(t *testing.T) {
	nodes := make([]*gentree.LocalityNode, 1000)
	epochOfEntryMap := make(map[string]Epoch)
	for i := range nodes {
		nodes[i] = &gentree.LocalityNode{}
		name := "node_" + strconv.Itoa(i)
		nodes[i].Name = name
		epochOfEntryMap[name] = Epoch(0)
	}

	SetLevels(nodes, epochOfEntryMap)

	levelsMap := make(map[int]int)
	for _, n := range nodes {
		levelsMap[n.Level]++
	}

	assert.Equal(t, 968, levelsMap[0])
	assert.Equal(t, 31, levelsMap[1])
	assert.Equal(t, 1, levelsMap[2])
	assert.Equal(t, 0, levelsMap[3])

}

func TestSetLevelsContinuity(t *testing.T) {
	nodes := make([]*gentree.LocalityNode, 6)
	epochOfEntryMap := make(map[string]Epoch)
	for i := range nodes {
		nodes[i] = &gentree.LocalityNode{}
		name := "node_" + strconv.Itoa(i)
		nodes[i].Name = name
		epochOfEntryMap[name] = Epoch(0)
	}

	SetLevels(nodes, epochOfEntryMap)

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

func TestSameLevelsForSameEntryTime(t *testing.T) {
	nodes := make([]*gentree.LocalityNode, 1000)
	epochOfEntryMap := make(map[string]Epoch)
	for i := range nodes {
		nodes[i] = &gentree.LocalityNode{}
		name := "node_" + strconv.Itoa(i)
		nodes[i].Name = name
		epochOfEntryMap[name] = Epoch(i)
	}

	SetLevels(nodes, epochOfEntryMap)

	levelsList1 := make([]int, 1000)
	for _, n := range nodes {
		levelsList1 = append(levelsList1, n.Level)
	}

	SetLevels(nodes, epochOfEntryMap)

	levelsList2 := make([]int, 1000)
	for _, n := range nodes {
		levelsList2 = append(levelsList2, n.Level)
	}

	assert.Equal(t, levelsList1, levelsList2)

	for i := range nodes {
		epochOfEntryMap["node_"+strconv.Itoa(i)] = Epoch(0)
	}

	SetLevels(nodes, epochOfEntryMap)

	levelsList2 = make([]int, 1000)
	for _, n := range nodes {
		levelsList2 = append(levelsList2, n.Level)
	}

	assert.NotEqual(t, levelsList1, levelsList2)

}
