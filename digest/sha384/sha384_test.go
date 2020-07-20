package sha384_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/simplesurance/baur/v1/digest"
	"github.com/simplesurance/baur/v1/digest/sha384"
)

func TestDigestOnEmptyHashErrors(t *testing.T) {
	const emptySHA384Digest = "38b060a751ac96384cd9327eb1b1e36a21fdb71114be07434c0cc7bf63f6e1da274edebfe76f65fbd51ad2f14898b95b"
	sha := sha384.New()
	d := sha.Digest()

	if d.Sum.Text(16) != emptySHA384Digest {
		t.Errorf("hash of nothing is %q expected %q", d.Sum.Text(16), emptySHA384Digest)
	}

	if d.Algorithm != digest.SHA384 {
		t.Errorf("Algorithm of Digest is set to %q expected %q", d.Algorithm, digest.SHA384)
	}
}

func TestAddBytes(t *testing.T) {
	const (
		helloSha384    = "59e1748777448c69de6b800d7a33bbfb9ff1b463e44354c3553bcdb9c666fa90125a3c79f90397bdf5f6a13de828684f"
		hellobyeSha384 = "f9904746d036ce9df915c4d2cae83acdd12aa9ef046648c4bc415cce6b86e64870b9c369d1a9b675b302d557b0a49ba5"
		helloStr       = "hello"
		byeStr         = "bye"
	)

	sha := sha384.New()
	err := sha.AddBytes([]byte(helloStr))
	if err != nil {
		t.Fatalf("AddBytes(%q) failed: %s", helloStr, err.Error())
	}

	d1 := sha.Digest()
	if d1.Algorithm != digest.SHA384 {
		t.Errorf("Algorithm of Digest is set to %q expected %q", d1.Algorithm, digest.SHA384)
	}

	if d1.Sum.Text(16) != helloSha384 {
		t.Errorf("calculated hash of %q is %q, expected %q", helloStr, d1.Sum.Text(16), helloSha384)
	}

	expectedStrRepr := "sha384:" + helloSha384
	if d1.String() != expectedStrRepr {
		t.Errorf("string representation of digest is %q, expected %q", d1.String(), expectedStrRepr)
	}

	err = sha.AddBytes([]byte(byeStr))
	if err != nil {
		t.Fatalf("AddBytes(%q) failed: %s", byeStr, err)
	}

	d2 := sha.Digest()
	if d1.Sum.Cmp(&d2.Sum) == 0 {
		t.Fatalf("adding %q to hash didn't change digest", byeStr)
	}

	if d2.Sum.Text(16) != hellobyeSha384 {
		t.Errorf("calculated hash of 'hellobye' is %q, expected %q", d1.Sum.Text(16), hellobyeSha384)
	}
}

func TestAddFile(t *testing.T) {
	const (
		testStr       = "this is a baur sha384 test file"
		testStrSHA384 = "63e291131dbf905a7fea3ffa4dbd8a49bee10055242e6ff1eea3c3862aefc33a4eb9580dd0c706d48b9ee861abfdacdf"
	)

	file, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal("creating tempfile failed:", err.Error())
	}
	defer os.Remove(file.Name())

	_, err = file.Write([]byte(testStr))
	if err != nil {
		file.Close()
		t.Fatal("writing to file failed:", err.Error())
	}

	if err := file.Close(); err != nil {
		t.Fatal("closing file failed:", err.Error())
	}

	sha := sha384.New()

	err = sha.AddFile(file.Name())
	if err != nil {
		t.Fatal("hashing file failed:", err.Error())
	}
	d := sha.Digest()

	if d.Sum.Text(16) != testStrSHA384 {
		t.Errorf("hash of file is %q expeted %q", d.Sum.Text(16), testStrSHA384)
	}
}

func TestHashingNonExistingFileFails(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal("creating tempfile failed:", err.Error())
	}
	os.Remove(file.Name())

	sha := sha384.New()
	err = sha.AddFile(file.Name())
	if err == nil {
		t.Errorf("hashing non existing file was successful")
	}
}
