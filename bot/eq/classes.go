package eq

import (
	"eqRaidBot/db/model"
	"fmt"
	"sort"
)

const MaxLevel = 60

const (
	classWarrior      = 1
	classMonk         = 2
	classRogue        = 3
	classPaladin      = 4
	classShadowknight = 5
	classRanger       = 6
	classEnchanter    = 7
	classWizard       = 8
	classMagician     = 9
	classNecromancer  = 10
	classShaman       = 11
	classDruid        = 12
	classCleric       = 13
	classBard         = 14

	classTypeTank   = "tank"
	classTypeMelee  = "melee"
	classTypeCaster = "caster"
	classTypeHealer = "healer"
	classTypeBard   = "bard"
)

type classRangeMap map[int64]int

var Tanks = classRangeMap{
	classWarrior:      1,
	classPaladin:      2,
	classShadowknight: 2,
}

var MeleeDps = classRangeMap{
	classMonk:   1,
	classRogue:  2,
	classRanger: 3,
}

var Healers = classRangeMap{
	classCleric: 1,
	classDruid:  2,
	classShaman: 3,
}

var CasterDps = classRangeMap{
	classNecromancer: 1,
	classMagician:    2,
	classWizard:      2,
	classEnchanter:   3,
}

var Bards = classRangeMap{
	classBard: 1,
}

var ClassChoiceMap = map[int64]string{
	classWarrior:      "Warrior",
	classMonk:         "Monk",
	classRogue:        "Rogue",
	classPaladin:      "Paladin",
	classShadowknight: "Shadowknight",
	classRanger:       "Ranger",
	classEnchanter:    "Enchanter",
	classWizard:       "Wizard",
	classMagician:     "Magician",
	classNecromancer:  "Necromancer",
	classShaman:       "Shaman",
	classDruid:        "Druid",
	classCleric:       "Cleric",
	classBard:         "Bard",
}

var ClassChoiceString = func() string {
	str := ""
	var i int64
	for i = 1; i < int64(len(ClassChoiceMap)); i++ {
		str += fmt.Sprintf("%d. %s\n", i, ClassChoiceMap[i])
	}
	return str
}

func raidWideClassGroups(characters []model.Character) map[int64][]model.Character {
	classGroups := make(map[int64][]model.Character)
	for _, c := range characters {
		if _, ok := classGroups[c.Class]; !ok {
			classGroups[c.Class] = []model.Character{c}
		} else {
			classGroups[c.Class] = append(classGroups[c.Class], c)
		}
	}

	for _, group := range classGroups {
		sort.Slice(group, sortToons(group))
	}

	return classGroups
}

func sortToons(group []model.Character) func(i, j int) bool {
	return func(i, j int) bool {
		if group[i].Level != group[j].Level {
			return group[i].Level > group[j].Level
		}

		if group[i].AA != group[j].AA {
			return group[i].AA > group[j].AA
		}

		return group[i].IsBot != group[j].IsBot
	}
}

func selectionClassGroups(raidList []model.Character) map[string][]model.Character {
	classes := make(map[string][]model.Character)

	for _, k := range raidList {
		if _, ok := Tanks[k.Class]; ok {
			if _, ok := classes[classTypeTank]; ok {
				classes[classTypeTank] = append(classes[classTypeTank], k)
			} else {
				classes[classTypeTank] = []model.Character{k}
			}
			continue
		}

		if _, ok := MeleeDps[k.Class]; ok {
			if _, ok := classes[classTypeMelee]; ok {
				classes[classTypeMelee] = append(classes[classTypeMelee], k)
			} else {
				classes[classTypeMelee] = []model.Character{k}
			}
			continue
		}

		if _, ok := Healers[k.Class]; ok {
			if _, ok := classes[classTypeHealer]; ok {
				classes[classTypeHealer] = append(classes[classTypeHealer], k)
			} else {
				classes[classTypeHealer] = []model.Character{k}
			}
			continue
		}

		if _, ok := CasterDps[k.Class]; ok {
			if _, ok := classes[classTypeCaster]; ok {
				classes[classTypeCaster] = append(classes[classTypeCaster], k)
			} else {
				classes[classTypeCaster] = []model.Character{k}
			}
			continue
		}

		if _, ok := Bards[k.Class]; ok {
			if _, ok := classes[classTypeBard]; ok {
				classes[classTypeBard] = append(classes[classTypeBard], k)
			} else {
				classes[classTypeBard] = []model.Character{k}
			}
		}
	}

	for class, group := range classes {
		// bards dont need sorting
		if class == classTypeBard {
			continue
		}

		switch class {
		case classTypeTank:
			sort.Slice(group, func(i, j int) bool {
				return Tanks[group[i].Class] < Tanks[group[j].Class]
			})
		case classTypeMelee:
			sort.Slice(group, func(i, j int) bool {
				return MeleeDps[group[i].Class] < MeleeDps[group[j].Class]
			})
		case classTypeCaster:
			sort.Slice(group, func(i, j int) bool {
				return CasterDps[group[i].Class] < CasterDps[group[j].Class]
			})
		case classTypeHealer:
			sort.Slice(group, func(i, j int) bool {
				return Healers[group[i].Class] < Healers[group[j].Class]
			})
		}
	}

	return classes
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
