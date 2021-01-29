package cfg

// OutputIncludes is a list of OutputInclude
type OutputIncludes []*OutputInclude

// validate checks if the stored information is valid.
func (incl *OutputIncludes) validate() error {
	for _, out := range *incl {
		if err := out.validate(); err != nil {
			if out.IncludeID != "" {
				return fieldErrorWrap(err, out.IncludeID)
			}

			return err
		}
	}

	return nil
}
