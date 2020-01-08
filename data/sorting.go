package data

import (
	"sort"
)

type sortInsertInterface interface {
	sort.Interface
	// Insert moves the last element to the ith place and shifts all other
	// elements back.
	Insert(i int)
}

// takes the last element and inserts it into the list of other elements that
// is assumed to be sorted, so that the whole data is sorted.
func insertSorted(ssi sortInsertInterface) {
	min := 0
	toInsert := ssi.Len() - 1
	max := toInsert
	for {
		if max-min <= 1 {
			ssi.Insert(min)
			return
		}
		cur := min + (max-min)/2
		if ssi.Less(cur, toInsert) {
			min = cur + 1
		} else {
			max = cur
		}
	}
}

type groupSortInterface struct {
	data []*group
}

func (gsi groupSortInterface) Len() int {
	return len(gsi.data)
}

func (gsi groupSortInterface) Less(i int, j int) bool {
	return gsi.data[i].name < gsi.data[j].name
}

func (gsi groupSortInterface) Swap(i int, j int) {
	gsi.data[i], gsi.data[j] = gsi.data[j], gsi.data[i]
}

func (gsi groupSortInterface) Insert(i int) {
	elm := gsi.data[len(gsi.data)-1]
	copy(gsi.data[i+1:], gsi.data[i:])
	gsi.data[i] = elm
}

type systemSortInterface struct {
	data  []*system
	start int
}

func (ssi systemSortInterface) Len() int {
	return len(ssi.data) - ssi.start
}

func (ssi systemSortInterface) Less(i int, j int) bool {
	return ssi.data[ssi.start+i].name < ssi.data[ssi.start+j].name
}

func (ssi systemSortInterface) Swap(i int, j int) {
	ssi.data[ssi.start+i], ssi.data[ssi.start+j] =
		ssi.data[ssi.start+j], ssi.data[ssi.start+i]
}

func (ssi systemSortInterface) Insert(i int) {
	elm := ssi.data[len(ssi.data)-1]
	copy(ssi.data[ssi.start+i+1:], ssi.data[ssi.start+i:])
	ssi.data[ssi.start+i] = elm
}
