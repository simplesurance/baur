package cfg

// InputIncludes is a list of InputInclude.
type InputIncludes []*InputInclude

// validate checks that the stored information is valid.
func (incl InputIncludes) validate() error {
	for _, in := range incl {
		if err := in.validate(); err != nil {
			if in.IncludeID != "" {
				return fieldErrorWrap(err, in.IncludeID)
			}

			return err
		}
	}

	return nil
}
