package eq

import (
	"eqRaidBot/db/model"
	"fmt"
	"sort"
)

type Splitter struct {
	characters []model.Character
}

func NewSplitter(c []model.Character) *Splitter {
	return &Splitter{
		characters: c,
	}
}

// sort the classes into groups
// sort each group by level / aa / dkp
// round robin class buckets off into raid groups

// once raids are balacned with respect to class composition form groups

// raid should have 1 main tank group - and DPS / healing groups
// bards should be spread - enchanters paired with healers, and melee clustered together
func (r *Splitter) Split(groupN int) [][]model.Character {
	classGroups := r.genClassGroups(groupN)
	splits := r.getSplits(groupN, classGroups)
	for i, c := range splits {
		fmt.Println(i, len(c))
		fmt.Println(GenSpread(c))
	}

	return splits
}

func (r *Splitter) getSplits(groupN int, classGroups map[int64][]model.Character) [][]model.Character {
	splits := make([][]model.Character, groupN)
	var (
		top       model.Character
		classN    int64 = 1
		currSplit       = 0
	)

	for {
		if classN > 13 {
			break
		}

		for len(classGroups[classN]) > 0 {
			top, classGroups[classN] = classGroups[classN][0], classGroups[classN][1:]
			splits[currSplit] = append(splits[currSplit], top)

			if currSplit == len(splits)-1 {
				currSplit = 0
			} else {
				currSplit++
			}
		}
		classN++
	}

	return splits
}

func (r *Splitter) genClassGroups(groupN int) map[int64][]model.Character {
	classGroups := make(map[int64][]model.Character)
	for _, c := range r.characters {
		if _, ok := classGroups[c.Class]; !ok {
			classGroups[c.Class] = []model.Character{c}
		} else {
			classGroups[c.Class] = append(classGroups[c.Class], c)
		}
	}

	for _, group := range classGroups {
		sort.Slice(group, func(i, j int) bool {
			if group[i].Level != group[j].Level {
				return group[i].Level > group[j].Level
			}

			return group[i].AA > group[j].AA
		})
	}

	return classGroups
}

func GenSpread(toons []model.Character) map[string]int {
	spread := make(map[string]int)

	for _, t := range toons {
		c := ClassChoiceMap[t.Class]
		if _, ok := spread[c]; !ok {
			spread[c] = 1
		} else {
			spread[c]++
		}
	}

	return spread

}
