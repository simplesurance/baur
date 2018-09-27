package postgres

// SQLStringer contains fields and operator maps for string conversion
type SQLStringer interface {
	GetFieldsMap() SQLFields
	GetOperatorsMap() SQLFilterOperators
}

// SQLMap implements SQLStringer
type SQLMap struct {
	Fields    SQLFields
	Operators SQLFilterOperators
}

// GetFieldsMap is the getter for the map of fields
func (sm SQLMap) GetFieldsMap() SQLFields {
	return sm.Fields
}

// GetOperatorsMap is the getter for the map of operators
func (sm SQLMap) GetOperatorsMap() SQLFilterOperators {
	return sm.Operators
}
