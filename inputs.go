package baur

import (
	"context"
	"fmt"

	"github.com/simplesurance/baur/v1/internal/digest"
	"github.com/simplesurance/baur/v1/internal/digest/sha384"
	"github.com/simplesurance/baur/v1/storage"
)

// Inputs are resolved Inputs of a task.
type Inputs struct {
	Files                       []*Inputfile
	AdditionalStr               *InputString
	LookupAdditionalStrFallback *InputString
	digest                      *digest.Digest
	store                       storage.Storer
}

// NewInputs returns a new Inputs
func NewInputs(files []*Inputfile, additionalStr *InputString, lookupAdditionalStrFallback *InputString, store storage.Storer) *Inputs {
	return &Inputs{
		Files:                       files,
		AdditionalStr:               additionalStr,
		LookupAdditionalStrFallback: lookupAdditionalStrFallback,
		store:                       store}
}

// Digest returns a summarized digest over all Inputs.
// On the first call the digest is calculated, on subsequent calls the stored digest is returned.
func (in *Inputs) Digest() (*digest.Digest, error) {
	if in.digest != nil {
		return in.digest, nil
	}

	digests := make([]*digest.Digest, len(in.Files))

	for i, file := range in.Files {
		fdigest, err := file.Digest()
		if err != nil {
			return nil, fmt.Errorf("calculating digest for %q failed: %w", file.Path(), err)
		}

		digests[i] = fdigest
	}

	if in.AdditionalStr.Exists() {
		idigest, err := in.AdditionalStr.Digest()
		if err != nil {
			return nil, fmt.Errorf("calculating digest for additional string %q failed: %w", in.AdditionalStr.Value(), err)
		}
		digests = append(digests, idigest)
	}

	totalDigest, err := sha384.Sum(digests)
	if err != nil {
		return nil, err
	}

	in.digest = totalDigest

	return in.digest, nil
}

// TaskStatusDigest returns a summarized digest over all Inputs for looking up task statuses.
// The return value can differ depending on the state of the additional input string and lookup fallback string values.
func (in *Inputs) TaskStatusDigest(ctx context.Context, task *Task) (*digest.Digest, error) {
	digests := make([]*digest.Digest, len(in.Files))

	for i, file := range in.Files {
		fdigest, err := file.Digest()
		if err != nil {
			return nil, fmt.Errorf("calculating digest for %q failed: %w", file.Path(), err)
		}

		digests[i] = fdigest
	}

	if in.AdditionalStr.Exists() {
		idigest, err := in.AdditionalStr.Digest()
		if err != nil {
			return nil, fmt.Errorf("calculating digest for additional string %q failed: %w", in.AdditionalStr.Value(), err)
		}

		additionalStrExistsInDb, err := in.store.InputExistsByDigest(ctx, task.AppName, task.Name, idigest.String())
		if err != nil {
			return nil, fmt.Errorf("calculating digest for additional string %q failed: %w", in.AdditionalStr.Value(), err)
		}

		if !additionalStrExistsInDb && in.LookupAdditionalStrFallback.Exists() {
			idigest, err = in.LookupAdditionalStrFallback.Digest()
			if err != nil {
				return nil, fmt.Errorf("calculating digest for lookup additional string fallback %q failed: %w", in.LookupAdditionalStrFallback.Value(), err)
			}
		}

		digests = append(digests, idigest)
	}

	totalDigest, err := sha384.Sum(digests)
	if err != nil {
		return nil, err
	}

	return totalDigest, nil
}
