package util

import (
	"eqRaidBot/bot/eq"
	"eqRaidBot/db/model"
	"math/rand"
	"strconv"
	"time"
)

func GenerateDBObjects(size int) ([]model.Character, map[string]int) {
	rand.Seed(time.Now().UnixNano())
	var toons []model.Character

	min := 1
	max := 14

	minL := 52
	maxL := 60

	for len(toons) < size {
		class := int64(rand.Intn(max-min+1) + min)
		level := int64(rand.Intn(maxL-minL+1) + minL)

		toons = append(toons, model.Character{
			Id:        int64(rand.Intn(10000)),
			Name:      eq.ClassChoiceMap[class] + strconv.Itoa(rand.Intn(10_000)),
			Level:     level,
			Class:     class,
			IsBot:     false,
			AA:        0,
			CreatedBy: "god",
		})
	}

	return toons, eq.GenSpread(toons)
}
