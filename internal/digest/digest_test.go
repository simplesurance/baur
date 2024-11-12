package digest

import (
	"testing"
)

func TestFromString(t *testing.T) {
	const shash = "sha384:5cb48e5ee7ec1305b3b6b26325bde82cc734f17dca9ea58510948156e3c4c51df04a580604b7b4c3f183bdda47b93322"

	d, err := FromString(shash)
	if err != nil {
		t.Fatalf("parsing %q failed: %s", shash, err)
	}

	if d.Algorithm != SHA384 {
		t.Errorf("wrong algorithm %q parsed, expected SHA384", d.Algorithm)
	}

	if d.String() != shash {
		t.Errorf("String() returned %q expected %q", d.String(), shash)
	}
}
