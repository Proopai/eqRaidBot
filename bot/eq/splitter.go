package eq

import (
	"eqRaidBot/db/model"
	"fmt"
	"math"
)

type Splitter struct {
	characters []model.Character
	usedMap    map[int64]bool
	debug      bool
}

func NewSplitter(c []model.Character, debug bool) *Splitter {
	return &Splitter{
		characters: c,
		usedMap:    make(map[int64]bool),
		debug:      debug,
	}
}

// sort the classes into groups
// sort each group by level / aa / dkp
// round robin class buckets off into raid groups

// once raids are balacned with respect to class composition form groups

// raid should have 1 main tank group - and DPS / healing groups
// bards should be spread - enchanters paired with healers, and melee clustered together
func (r *Splitter) Split(groupN int) ([][][]model.Character, []map[int64]int) {
	classGroups := raidWideClassGroups(r.characters)
	splits := r.getSplits(groupN, classGroups)

	splitGroups := make([][][]model.Character, len(splits))

	if r.debug {
		fmt.Printf("Splitting %d by %d, split size: %d\n", len(r.characters), groupN, len(splits[0]))
	}

	for i, c := range splits {
		groups := r.buildGroups(c)
		splitGroups[i] = groups
		if r.debug {
			fmt.Printf("\nSplit %d\n", i+1)
			for _, g := range groups {
				var gString string
				for _, c := range g {
					gString += fmt.Sprintf("%s ", ClassChoiceMap[c.Class])
				}

				fmt.Printf(gString + "\n")
			}
		}
	}

	var stats []map[int64]int
	for _, s := range splits {
		stats = append(stats, RaidWideClassCounts(s))
	}

	return splitGroups, stats
}

func (r *Splitter) buildGroups(raidList []model.Character) [][]model.Character {
	groups := make([][]model.Character, int(math.Ceil(float64(len(raidList))/6)))
	classGroups := selectionClassGroups(raidList)

	if r.debug {
		for k, c := range classGroups {
			fmt.Printf("%s - %d\n", k, len(c))
		}
	}

	for i := 0; i < len(groups)-1; i++ {
		groups[i] = r.buildGroup(i, classGroups)
	}

	var leftOver []model.Character
	for _, k := range raidList {
		if _, ok := r.usedMap[k.Id]; !ok {
			leftOver = append(leftOver, k)
		}
	}

	if len(leftOver) > 0 {
		groups[len(groups)-1] = leftOver
	}

	return groups
}

// Build a group given the group n and list of possible members
// gNum represents the group number
// group 1 will be the tank group
// group 2 will be the cleric group
// group 3 will be the primary melee dps group
// all other groups will be DPS or mixed
// split bards in all groups
func (r *Splitter) buildGroup(gNum int, classGroups map[string][]model.Character) []model.Character {
	var group []model.Character
	var t string
	switch gNum {
	case 0: // tank group
		t = classTypeTank
	case 1: // healer group
		t = classTypeMelee
	case 2: // melee group
		t = classTypeHealer
	case 3: // caster group
		t = classTypeCaster
	default:
		t = "any"
	}

	r.addMembersToGroup(classGroups, t, &group)

	return group
}

func (r *Splitter) addMembersToGroup(classGroups map[string][]model.Character, indicator string, group *[]model.Character) {
	var list []model.Character
	if indicator == "any" {
		list = append(list, classGroups[classTypeBard]...)
		list = append(list, classGroups[classTypeMelee]...)
		list = append(list, classGroups[classTypeCaster]...)
		list = append(list, classGroups[classTypeHealer]...)
		list = append(list, classGroups[classTypeEnchanter]...)
		list = append(list, classGroups[classTypeTank]...)
	} else {
		list = classGroups[indicator]
	}

	for _, raider := range list {
		if _, ok := r.usedMap[raider.Id]; ok {
			continue
		}

		if len(*group) == 5 {
			break
		}

		*group = append(*group, raider)
		r.usedMap[raider.Id] = true
	}

	hasBard := false
	for len(*group) != 6 {
		// see if we can add a bard
		if !hasBard && r.bardsAvailable(classGroups) {
			for _, k := range classGroups[classTypeBard] {
				if _, ok := r.usedMap[k.Id]; !ok {
					*group = append(*group, k)
					hasBard = true
					r.usedMap[k.Id] = true
					break
				}
			}
		} else {
			var choices []model.Character
			choices = append(choices, classGroups[classTypeHealer]...)
			choices = append(choices, classGroups[classTypeMelee]...)
			choices = append(choices, classGroups[classTypeCaster]...)
			choices = append(choices, classGroups[classTypeEnchanter]...)
			choices = append(choices, classGroups[classTypeTank]...)

			for _, k := range choices {
				if _, ok := r.usedMap[k.Id]; !ok {
					*group = append(*group, k)
					r.usedMap[k.Id] = true
					break
				}
			}
		}
	}
}

func (r *Splitter) bardsAvailable(classGroups map[string][]model.Character) bool {
	for _, k := range classGroups[classTypeBard] {
		if _, ok := r.usedMap[k.Id]; !ok {
			return true
		}
	}

	return false
}

func (r *Splitter) getSplits(groupN int, classGroups map[int64][]model.Character) [][]model.Character {
	splits := make([][]model.Character, groupN)
	var (
		top       model.Character
		classN    int64 = 1
		currSplit       = 0
	)

	for {
		if classN > 14 {
			break
		}

		if _, ok := classGroups[classN]; !ok {
			classN++
			continue
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
