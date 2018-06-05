package cfg

import "testing"

func Test_ExampleApp_IsValid(t *testing.T) {
	a := ExampleApp("shop")
	if err := a.Validate(); err != nil {
		t.Error("example app conf fails validation: ", err)
	}
}
