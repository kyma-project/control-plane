package provider

import (
	"math/rand"
	"strconv"
)

func updateString(toUpdate *string, value *string) {
	if value != nil {
		*toUpdate = *value
	}
}

func updateSlice(toUpdate *[]string, value []string) {
	if value != nil {
		*toUpdate = value
	}
}

func generateDefaultAzureZones() []string {
	return []string{generateRandomAzureZone()}
}

func generateRandomAzureZone() string {
	const (
		min = 1
		max = 3
	)

	// generates random number from 1-3 range
	getRandomNumber := func() int {
		return rand.Intn(max-min+1) + min
	}

	return strconv.Itoa(getRandomNumber())
}

func generateMultipleAzureZones(zoneCount int) []string {
	zones := []string{"1", "2", "3"}

	rand.Shuffle(len(zones), func(i, j int) { zones[i], zones[j] = zones[j], zones[i] })
	return zones[:zoneCount]
}
