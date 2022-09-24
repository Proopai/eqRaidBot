package util

import (
	"eqRaidBot/bot/eq"
	"eqRaidBot/db/model"
	"math/rand"
	"strconv"
	"time"
)

var (
	min = 1
	max = 14

	minL = 52
	maxL = 60
)

func GenerateDBObjects(size int) ([]model.Character, map[string]int) {
	rand.Seed(time.Now().UnixNano())
	var toons []model.Character

	for i := 0; i < 10; i++ {
		toons = append(toons, botPair()...)
	}

	for len(toons) < size {
		class := int64(rand.Intn(max-min+1) + min)
		level := int64(rand.Intn(maxL-minL+1) + minL)

		toons = append(toons, model.Character{
			Id:            int64(rand.Intn(10000)),
			Name:          eq.ClassChoiceMap[class] + strconv.Itoa(rand.Intn(10_000)),
			Level:         level,
			Class:         class,
			CharacterType: 2,
			AA:            0,
			CreatedBy:     "god" + randStringRunes(10),
		})
	}

	return toons, eq.GenSpread(toons)
}

func botPair() []model.Character {
	c1, c2, c3 := int64(rand.Intn(max-min+1)+min), int64(rand.Intn(max-min+1)+min), int64(rand.Intn(max-min+1)+min)
	name := randStringRunes(10)
	return []model.Character{
		{
			Id:            int64(rand.Intn(10000)),
			Name:          eq.ClassChoiceMap[c1] + strconv.Itoa(rand.Intn(10_000)),
			Class:         c1,
			Level:         int64(rand.Intn(maxL-minL+1) + minL),
			CharacterType: 2,
			AA:            0,
			CreatedBy:     name,
		},
		{
			Id:            int64(rand.Intn(10000)),
			Name:          eq.ClassChoiceMap[c2] + strconv.Itoa(rand.Intn(10_000)),
			Class:         c2,
			Level:         int64(rand.Intn(maxL-minL+1) + minL),
			CharacterType: 1,
			AA:            0,
			CreatedBy:     name,
		},
		{
			Id:            int64(rand.Intn(10000)),
			Name:          eq.ClassChoiceMap[c3] + strconv.Itoa(rand.Intn(10_000)),
			Class:         c3,
			Level:         int64(rand.Intn(maxL-minL+1) + minL),
			CharacterType: 3,
			AA:            0,
			CreatedBy:     name,
		},
	}
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
