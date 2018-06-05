package cfg

import "testing"

func Test_ExampleRepository_IsValid(t *testing.T) {
	r := ExampleRepository()
	if err := r.Validate(); err != nil {
		t.Error("example repository conf fails validation: ", err)
	}
}
