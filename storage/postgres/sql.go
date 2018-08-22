package postgres

type SqlStringer interface {
	GetFieldsMap() SqlFields
	GetOperatorsMap() SqlFilterOperators
}

type SqlMap struct {
	Fields    SqlFields
	Operators SqlFilterOperators
}

func (sm SqlMap) GetFieldsMap() SqlFields {
	return sm.Fields
}

func (sm SqlMap) GetOperatorsMap() SqlFilterOperators {
	return sm.Operators
}
