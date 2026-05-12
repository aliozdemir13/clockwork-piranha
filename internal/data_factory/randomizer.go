// Package data_factory handles data preparation and randomization
package data_factory

import (
	"errors"
	"math/rand/v2"
)

// ShuffleOnInit aims to run a cheap shuffle for read operations
// TODO: copy the given array to keep original version intact
func ShuffleOnInit(memberDetails []*MemberDetails) ([]*MemberDetails, error) {
	if len(memberDetails) == 0 {
		return nil, errors.New("sample list is empty")
	}

	rand.Shuffle(len(memberDetails), func(i, j int) {
		memberDetails[i], memberDetails[j] = memberDetails[j], memberDetails[i]
	})

	return memberDetails, nil
}
