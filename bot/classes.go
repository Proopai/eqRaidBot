package bot

import "fmt"

const maxLevel = 60

var classChoiceMap = map[int64]string{
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

var classChoiceString = func() string {
	str := ""
	var i int64
	for i = 1; i < int64(len(classChoiceMap)); i++ {
		str += fmt.Sprintf("%d. %s\n", i, classChoiceMap[i])
	}
	return str
}
