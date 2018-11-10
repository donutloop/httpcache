package roundtripper

import (
	"net/http"
	"testing"
)

func TestMakeHashFromRequest(t *testing.T) {

	req, err := http.NewRequest(http.MethodGet, "http://test.de", nil)
	if err != nil {
		t.Fatal(err)
	}

	hash1, err := makeHashFromRequest(req)
	if err != nil {
		t.Fatal(err)
	}

	hash2, err := makeHashFromRequest(req)
	if err != nil {
		t.Fatal(err)
	}

	if hash1 != hash2 {
		t.Log("hash 1: " + hash1)
		t.Log("hash 2: " + hash1)
		t.Error("hash are not equal")
	}

	t.Log("hash 1: " + hash1)
	t.Log("hash 2: " + hash1)
}
