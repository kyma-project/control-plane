package keb

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func getRuntimes(endpoint string) (*Runtimes, error) {
	resp, err := http.Get(endpoint)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	fmt.Println("Response status:", resp.Status)

	var runtimes Runtimes

	err = json.NewDecoder(resp.Body).Decode(&runtimes)
	if &runtimes != nil {
		fmt.Println(runtimes.Count)
		fmt.Println(runtimes.TotalCount)
		fmt.Println(runtimes.Data[0].Status.CreatedAt)
	}
	if err != nil {
		fmt.Printf("invalid JSON: %v", err)
		return nil, err
	}
	return &runtimes, err
}
