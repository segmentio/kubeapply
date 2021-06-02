package provider

import "testing"

func TestProvider(t *testing.T) {
	if err := Provider(nil).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}
