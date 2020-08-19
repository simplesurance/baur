package cfg

// OutputIncludes is a list of OutputInclude
type OutputIncludes []*OutputInclude

// Validate checks if the stored information is valid.
func (incl *OutputIncludes) Validate() error {
	for _, out := range *incl {
		if err := out.Validate(); err != nil {
			if out.IncludeID != "" {
				return FieldErrorWrap(err, out.IncludeID)
			}

			return err
		}
	}

	return nil
}
