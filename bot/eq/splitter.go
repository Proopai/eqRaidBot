package eq

import (
	"eqRaidBot/db/model"
	"fmt"
	"math"
	"sort"
)

type Splitter struct {
	characters []model.Character
	usedMap    map[int64]bool
	debug      bool
}

func NewSplitter(c []model.Character, debug bool) *Splitter {
	// we only take mains and box's so filter out any alts here just in case they are passed to us
	var characters []model.Character
	for _, v := range c {
		if v.CharacterType == model.TypeAlt {
			continue
		}
		characters = append(characters, v)
	}

	return &Splitter{
		characters: characters,
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
	charMap := make(map[string][]model.Character)
	for _, k := range r.characters {
		if _, ok := charMap[k.CreatedBy]; !ok {
			charMap[k.CreatedBy] = []model.Character{k}
		} else {
			charMap[k.CreatedBy] = append(charMap[k.CreatedBy], k)
		}
	}

	for k, v := range charMap {
		if len(v) == 1 {
			delete(charMap, k)
		}
	}

	classGroups := raidWideClassGroups(r.characters)
	splits := r.getSplits(groupN, classGroups, charMap)
	//for _, s := range splits {
	//	i := 0
	//	for _, v := range s {
	//		if v.CharacterType == model.TypeBox {
	//			fmt.Println(v)
	//		}
	//	}
	//	for c, v := range GenSpread(s) {
	//		i += v
	//		fmt.Printf("%s - %d\n", c, v)
	//	}
	//	fmt.Println("total", i)
	//	fmt.Println("==================================")
	//}
	//
	//os.Exit(0)

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
		list = r.buildChoiceList([]string{
			classTypeBard,
			classTypeMelee,
			classTypeCaster,
			classTypeHealer,
			classTypeTank,
		}, classGroups)
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
			choices := r.buildChoiceList([]string{
				classTypeHealer,
				classTypeMelee,
				classTypeCaster,
				classTypeTank,
			}, classGroups)

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

func (r *Splitter) buildChoiceList(types []string, classGroups map[string][]model.Character) []model.Character {
	var choices []model.Character
	for _, t := range types {
		choices = append(choices, classGroups[t]...)
	}
	return choices
}

func (r *Splitter) bardsAvailable(classGroups map[string][]model.Character) bool {
	for _, k := range classGroups[classTypeBard] {
		if _, ok := r.usedMap[k.Id]; !ok {
			return true
		}
	}

	return false
}

func (r *Splitter) getSplits(groupN int, classGroups map[int64][]model.Character, charMap map[string][]model.Character) [][]model.Character {
	splits := make([][]model.Character, groupN)
	var (
		top       model.Character
		classN    int64 = 1
		currSplit       = 0
	)

	// handle any bots that exist
	for _, chars := range charMap {
		splits[currSplit] = append(splits[currSplit], chars...)
		for _, c := range chars {
			for i, v := range classGroups[c.Class] {
				if c.Id == v.Id {
					classGroups[c.Class] = append(classGroups[c.Class][:i], classGroups[c.Class][i+1:]...)
					break
				}
			}
		}

		if currSplit == len(splits)-1 {
			currSplit = 0
		} else {
			currSplit++
		}
	}

	sort.Slice(splits, func(i int, j int) bool {
		return len(splits[i]) < len(splits[j])
	})

	currSplit = 0

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
			// has a bot
			if _, ok := charMap[top.CreatedBy]; ok {
				splits[currSplit] = append(splits[currSplit], charMap[top.CreatedBy]...)
			} else {
				splits[currSplit] = append(splits[currSplit], top)
			}

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
