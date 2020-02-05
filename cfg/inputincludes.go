package cfg

// InputIncludes is a list of InputInclude.
type InputIncludes []*InputInclude

// Validate checks that the stored information is valid.
func (incl InputIncludes) Validate() error {
	for _, in := range incl {
		if err := in.Validate(); err != nil {
			if in.IncludeID != "" {
				return FieldErrorWrap(err, in.IncludeID)
			}

			return err
		}
	}

	return nil
}
