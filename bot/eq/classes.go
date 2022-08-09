package eq

import "fmt"

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
)

type classRangeMap map[int]int

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

var ClassChoiceMap = map[int64]string{
	1:  "Warrior",
	2:  "Monk",
	3:  "Rogue",
	4:  "Paladin",
	5:  "Shadowknight",
	6:  "Ranger",
	7:  "Enchanter",
	8:  "Wizard",
	9:  "Magician",
	10: "Necromancer",
	11: "Shaman",
	12: "Druid",
	13: "Cleric",
}

var ClassChoiceString = func() string {
	str := ""
	var i int64
	for i = 1; i < int64(len(ClassChoiceMap)); i++ {
		str += fmt.Sprintf("%d. %s\n", i, ClassChoiceMap[i])
	}
	return str
}
