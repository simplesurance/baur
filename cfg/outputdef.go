package cfg

type OutputDef interface {
	DockerImageOutputs() *[]*DockerImageOutput
	FileOutputs() *[]*FileOutput
}

// OutputRemoveEmptySections removes elements from the file and
// DockerImageOutputs slices that only contain empty elements.
// If the config file does not contain a definition of section and it is a
// concrete type in the config struct instead of a pointer, the toml lib
// creates an element in the slice where no field is set.
func OutputRemoveEmptySections(o OutputDef) {
	var i int

	fileOutputs := o.FileOutputs()
	for _, f := range *fileOutputs {
		if !f.IsEmpty() {
			(*fileOutputs)[i] = f
			i++
		}
	}
	(*fileOutputs) = (*fileOutputs)[:i]

	i = 0
	dockerOutputs := o.DockerImageOutputs()
	for _, d := range *dockerOutputs {
		if !d.IsEmpty() {
			(*dockerOutputs)[i] = d
			i++
		}
	}
	(*dockerOutputs) = (*dockerOutputs)[:i]
}
