package main

import (
	"testing"
)

func TestFindElementInArray(t *testing.T) {

	tags := []string{"one", "two", "tHree", "four", "one", "One"}
	text := "one"

	wasFound, location := FindElementInArray(tags, text)

	// fmt.Println(wasFound, location)
	if !wasFound || location != 2 {
		t.Errorf("Expected wasFound to be true and location = 2")
	}
}
