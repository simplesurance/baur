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

func (incl *OutputIncludes) RemoveEmptyElements() {
	var i int

	for _, output := range *incl {
		OutputRemoveEmptySections(output)

		if len(output.DockerImage) != 0 || len(output.File) != 0 {
			(*incl)[i] = output
			i++
		}
	}

	*incl = (*incl)[:i]
}
