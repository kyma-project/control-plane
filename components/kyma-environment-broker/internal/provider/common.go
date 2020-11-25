package provider

import (
	"math/rand"
	"strconv"
	"time"
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
	rand.Seed(time.Now().UnixNano())

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
