package storage

import "fmt"

type Field int

const (
	FieldNull Field = iota

	FieldOne

	FieldApplicationID
	FieldApplicationName

	FieldVCSStateID
	FieldVCSStateCommitID
	FieldVCSStateIsDirty

	FieldBuildID
	FieldBuildStartDatetime
	FieldBuildStopTimestamp
	FieldBuildTotalInputDigest

	FieldOutputID
	FieldOutputName
	FieldOutputType
	FieldOutputSizeBytes

	FieldUploadID
	FieldUploadDuration

	FieldDuration
)

func (f Field) GetName(fields map[Field]string) (string, error) {
	if name, ok := fields[f]; ok {
		return name, nil
	}

	return "", fmt.Errorf("field %d missing from collection", f)
}
