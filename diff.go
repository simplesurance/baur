package baur

import (
	"fmt"
	"sort"
)

type DiffType int

const (
	DigestMismatch DiffType = iota
	Removed
	Added
)

var diffTypeStrings = [...]string{"D", "-", "+"}

func (d DiffType) String() string {
	return diffTypeStrings[d]
}

type InputDiff struct {
	State   DiffType
	Path    string
	Digest1 string
	Digest2 string
}

// DiffInputs returns the differences between two sets of Inputs.
// The Input.String() is used as the key to identify each Input.
func DiffInputs(a, b *Inputs) ([]*InputDiff, error) {
	aMap, err := inputsToStrMap(a.Inputs())
	if err != nil {
		return nil, err
	}

	bMap, err := inputsToStrMap(b.Inputs())
	if err != nil {
		return nil, err
	}

	var diffs []*InputDiff

	for aPath, aDigest := range aMap {
		bDigest, exists := bMap[aPath]
		if !exists {
			diffs = append(diffs, &InputDiff{State: Removed, Path: aPath, Digest1: aDigest})
			continue
		}

		if aDigest != bDigest {
			diffs = append(diffs, &InputDiff{State: DigestMismatch, Path: aPath, Digest1: aDigest, Digest2: bDigest})
		}
	}

	for bPath, bDigest := range bMap {
		if _, exists := aMap[bPath]; !exists {
			diffs = append(diffs, &InputDiff{State: Added, Path: bPath, Digest2: bDigest})
		}
	}

	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].Path < diffs[j].Path
	})

	return diffs, nil
}

func inputsToStrMap(inputs []Input) (map[string]string, error) {
	inputsMap := make(map[string]string, len(inputs))

	for _, input := range inputs {
		digest, err := input.Digest()
		if err != nil {
			return nil, fmt.Errorf("%s: calculating digest failed: %w", input, err)
		}

		inputsMap[input.String()] = digest.String()
	}

	return inputsMap, nil
}
