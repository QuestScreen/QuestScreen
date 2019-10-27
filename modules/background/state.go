package background

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/flyx/rpscreen/data"
)

type state struct {
	owner     *Background
	curIndex  int
	resources []data.Resource
}

// LoadFrom loads the stored selection, defaults to no item being selected.
func (s *state) LoadFrom(yamlSubtree interface{}, store *data.Store) error {
	s.resources = store.ListFiles(s.owner, "")
	s.curIndex = -1
	if yamlSubtree != nil {
		scalar, ok := yamlSubtree.(string)
		if !ok {
			return errors.New("unexpected value for Background state: not a string")
		}
		for i := range s.resources {
			if s.resources[i].Name == scalar {
				s.curIndex = i
				break
			}
		}
		if s.curIndex == -1 {
			log.Println("Didn't find resource \"" + scalar + "\"")
		}
	}
EmptyBuffer:
	for {
		select {
		case <-s.owner.reqTextureIndex:
		default:
			break EmptyBuffer
		}
	}
	s.owner.reqTextureIndex <- s.curIndex
	return nil
}

// ToYAML returns the name of the currently selected resource, or nil if none
func (s *state) ToYAML(store *data.Store) interface{} {
	if s.curIndex == -1 {
		return nil
	}
	return s.resources[s.curIndex].Name
}

type jsonState struct {
	CurIndex int      `json:"curIndex"`
	Items    []string `json:"items"`
}

// ToJSON returns the index of the current item (-1 if none)
func (s *state) ToJSON() interface{} {
	return jsonState{CurIndex: s.curIndex, Items: data.ResourceNames(s.resources)}
}

func (s *state) Actions() []string {
	return []string{"set"}
}

func (s *state) HandleAction(index int, payload []byte, store *data.Store) error {
	if index != 0 {
		panic("Index out of bounds!")
	}
	var value int
	if err := json.Unmarshal(payload, &value); err != nil {
		return err
	}
	if value < -1 || value >= len(s.resources) {
		return fmt.Errorf("Value %d out of bounds -1..%d",
			value, len(s.resources)-1)
	}
	select {
	case s.owner.reqTextureIndex <- value:
		s.curIndex = value
	default:
		return errors.New("Too many requests")
	}
	return nil
}
