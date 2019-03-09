package main

import (
	"testing"
)

func TestImageSearch(t *testing.T) {
	readConfig()
	image, err := googleImageSearch("kittens", ImageType_All, true, RandUint32(100)+1, 10)
	if err != nil {
		t.Log(err)
		t.Fail()
		return
	}
	t.Log(image)
}
