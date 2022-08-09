package eq

import (
	"eqRaidBot/db/model"
	"fmt"
)

type Splitter struct {
	characters []model.Character
}

func NewSplitter(c []model.Character) *Splitter {
	return &Splitter{
		characters: c,
	}
}

func (r *Splitter) Split(groupN int) [][]model.Character {
	fmt.Println(r.characters, groupN)
	return nil
}
