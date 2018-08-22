package storage

import "fmt"

// Field is the field type
type Field int

const (
	// FieldNull is the null value
	FieldNull Field = iota

	// FieldOne is used for the default 1=1 condition
	FieldOne

	// FieldApplicationName app name
	FieldApplicationName

	// FieldBuildID build id
	FieldBuildID
	// FieldBuildStartDatetime start time
	FieldBuildStartDatetime
	// FieldBuildTotalInputDigest total input digest
	FieldBuildTotalInputDigest

	// FieldDuration duration
	FieldDuration
)

// GetName returns the name of a field
func (f Field) GetName(fields map[Field]string) (string, error) {
	if name, ok := fields[f]; ok {
		return name, nil
	}

	return "", fmt.Errorf("field %d missing from collection", f)
}
