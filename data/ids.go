package data

import (
	"strconv"
	"strings"

	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func checkRemove(r rune) bool {
	return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') &&
		(r < '0' || r > '9') && r != '-' && r != '_'
}

var normalization = transform.Chain(norm.NFD, transform.RemoveFunc(checkRemove))

// generate a normalized string from the input, containing only alphanumerical
// characters, '-' and '_'
func normalize(input string) string {
	result, _, _ := transform.String(normalization, input)
	return result
}

// IDCollection describes a data structure with indexable items that, where each
// item has a unique ID
type idCollection interface {
	id(index int) string
	length() int
}

// genID generates an ID from the given name, ensuring that it is unique within
// the collection. If the normalized name is empty, baseName will be used
// instead as a base.
func genID(name string, baseName string, collection idCollection) string {
	base := strings.ToLower(normalize(name))
	id := base
	num := 0
	if base == "" {
		base = baseName
		id = baseName + "1"
		num++
	}
idCheckLoop:
	for {
		for i := 0; i < collection.length(); i++ {
			if collection.id(i) == id {
				num++
				id = base + strconv.Itoa(num)
				continue idCheckLoop
			}
		}
		break
	}
	return id
}

type systemIDs struct {
	data []*system
}

func (s systemIDs) id(index int) string {
	return s.data[index].id
}

func (s systemIDs) length() int {
	return len(s.data)
}

type groupIDs struct {
	data []*group
}

func (g groupIDs) id(index int) string {
	return g.data[index].id
}

func (g groupIDs) length() int {
	return len(g.data)
}

type sceneIDs struct {
	data []scene
}

func (s sceneIDs) id(index int) string {
	return s.data[index].id
}

func (s sceneIDs) length() int {
	return len(s.data)
}

type heroIDs struct {
	data []hero
}

func (h heroIDs) id(index int) string {
	return h.data[index].id
}

func (h heroIDs) length() int {
	return len(h.data)
}
