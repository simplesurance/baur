package storage

import (
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// findMostCommonOutputsByDigests finds builds that prododuces the same number
// of outputs with the same digests and returns a build that has the most common
// produced outputs.
//
// If the build  passed builds does not contain any outputs, the function
// returns an empty slice
func findMostCommonOutputsByDigests(builds []*BuildWithDuration) *BuildWithDuration {
	type entry struct {
		cnt   int
		build *BuildWithDuration
	}

	var highestCnt int
	var entryWithHighestCnt *entry
	entries := map[string]*entry{}

	for _, b := range builds {
		var digests []string

		for _, o := range b.Outputs {
			digests = append(digests, o.Digest)
		}

		sort.Strings(digests)
		key := strings.Join(digests, " ")

		if e, exist := entries[key]; exist {
			e.cnt++

			continue
		}

		entries[key] = &entry{
			build: b,
			cnt:   1,
		}
	}

	for _, e := range entries {
		if e.cnt > highestCnt {
			highestCnt = e.cnt
			entryWithHighestCnt = e
		}
	}

	if entryWithHighestCnt == nil {
		return nil
	}

	return entryWithHighestCnt.build
}

func digestIsInOutputSlice(outputs []*Output, digest string) bool {
	for _, o := range outputs {
		if o.Digest == digest {
			return true
		}
	}

	return false
}

// Issue describes an issue in the build database
type Issue string

const (
	// IssueOutputDigestMissing describes that a build didn't produces an
	// output that other builds with the same input digest produced
	IssueOutputDigestMissing Issue = "output with digest missing"
	// IssueUnmatchedDigest describes that a build produced an output with
	// a different digest then other builds with the same inputs
	IssueUnmatchedDigest Issue = "output with different digest"
)

// VerifyIssue describes a found issue during verification
type VerifyIssue struct {
	Build          *BuildWithDuration
	Output         *Output
	ReferenceBuild *BuildWithDuration
	Issue          Issue
}

func findOutputsWithDifferentDigest(refBuild *BuildWithDuration, builds []*BuildWithDuration) *VerifyIssue {
	for _, b := range builds {
		for _, o := range b.Outputs {
			if !digestIsInOutputSlice(refBuild.Outputs, o.Digest) {
				return &VerifyIssue{
					Build:          b,
					Output:         o,
					ReferenceBuild: refBuild,
					Issue:          IssueUnmatchedDigest,
				}
			}
		}

		for _, o := range refBuild.Outputs {
			if !digestIsInOutputSlice(b.Outputs, o.Digest) {
				return &VerifyIssue{
					Build:          b,
					Output:         o,
					ReferenceBuild: refBuild,
					Issue:          IssueOutputDigestMissing,
				}

			}
		}
	}

	return nil
}

// VerifySameInputDigestSameOutputs if the application has multiple builds with
// the same total input digest, it finds the most common outputs by digest from
// those builds and checks if the other builds have the outputs with the same
// digest.
// For builds that don't match an Issue description is returned
func VerifySameInputDigestSameOutputs(clt Storer, appName string, startTs time.Time) ([]*VerifyIssue, error) {
	var issues []*VerifyIssue

	builds, err := clt.GetSameTotalInputDigestsForAppBuilds(appName, startTs)
	if err != nil {
		if err == ErrNotExist {
			return nil, err
		}

		return nil, errors.Wrap(err, "retrieving builds with same total input digest failed")
	}

	for totalInputDigest, buildIDs := range builds {
		builds, err := clt.GetBuildsWithoutInputs([]*Filter{
			&Filter{
				Field:    FieldBuildID,
				Operator: OpIN,
				Value:    buildIDs,
			}}, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "retrieving builds for %s with TotalInputDigest %q failed", appName, totalInputDigest)
		}

		refBuild := findMostCommonOutputsByDigests(builds)

		issue := findOutputsWithDifferentDigest(refBuild, builds)
		if issue != nil {
			issues = append(issues, issue)
		}

	}

	return issues, nil
}
