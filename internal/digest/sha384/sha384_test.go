package sha384_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/simplesurance/baur/v1/internal/digest"
	"github.com/simplesurance/baur/v1/internal/digest/sha384"
)

func TestDigestOnEmptyHashErrors(t *testing.T) {
	const emptySHA384Digest = "sha384:38b060a751ac96384cd9327eb1b1e36a21fdb71114be07434c0cc7bf63f6e1da274edebfe76f65fbd51ad2f14898b95b"
	sha := sha384.New()
	d := sha.Digest()

	if d.String() != emptySHA384Digest {
		t.Errorf("hash of nothing is %q expected %q", d.String(), emptySHA384Digest)
	}

	if d.Algorithm != digest.SHA384 {
		t.Errorf("Algorithm of Digest is set to %q expected %q", d.Algorithm, digest.SHA384)
	}
}

func TestDigestWithLeadingZero(t *testing.T) {
	const expectedDigest = "sha384:033b18e7688f2a7ea6cf8101210f84f18c848576ece7600e5794fef70360b445c83ccd5b42e54d490e823399406cb81d"

	s := sha384.New()
	err := s.AddBytes([]byte("thing"))
	if err != nil {
		t.Fatal("adding bytes to digest failed", err)
	}

	strDigest := s.Digest().String()
	if strDigest != expectedDigest {
		t.Fatalf("digest is %q expected %q", strDigest, expectedDigest)
	}

	digestFromStr, err := digest.FromString(strDigest)
	if err != nil {
		t.Fatal("converting digest from string failed", err)
	}

	if digestFromStr.String() != strDigest {
		t.Fatalf("digest converted from string is %q expected %q", digestFromStr.String(), strDigest)
	}
}

func TestAddBytes(t *testing.T) {
	const (
		helloSha384    = "sha384:59e1748777448c69de6b800d7a33bbfb9ff1b463e44354c3553bcdb9c666fa90125a3c79f90397bdf5f6a13de828684f"
		hellobyeSha384 = "sha384:f9904746d036ce9df915c4d2cae83acdd12aa9ef046648c4bc415cce6b86e64870b9c369d1a9b675b302d557b0a49ba5"
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

	if d1.String() != helloSha384 {
		t.Errorf("string representation of digest is %q, expected %q", d1, helloSha384)
	}

	err = sha.AddBytes([]byte(byeStr))
	if err != nil {
		t.Fatalf("AddBytes(%q) failed: %s", byeStr, err)
	}

	d2 := sha.Digest()
	if bytes.Equal(d1.Sum, d2.Sum) {
		t.Fatalf("adding %q to hash didn't change digest", byeStr)
	}

	if d2.String() != hellobyeSha384 {
		t.Errorf("calculated hash of 'hellobye' is %q, expected %q", d1, hellobyeSha384)
	}
}

func TestAddFile(t *testing.T) {
	const (
		testStr       = "this is a baur sha384 test file"
		testStrSHA384 = "sha384:63e291131dbf905a7fea3ffa4dbd8a49bee10055242e6ff1eea3c3862aefc33a4eb9580dd0c706d48b9ee861abfdacdf"
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

	if d.String() != testStrSHA384 {
		t.Errorf("hash of file is %q expected %q", d.String(), testStrSHA384)
	}
}

func TestHashingNonExistingFileFails(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal("creating tempfile failed:", err.Error())
	}
	// The file must be closed before it can be deleted on Windows
	file.Close()
	os.Remove(file.Name())

	sha := sha384.New()
	err = sha.AddFile(file.Name())
	if err == nil {
		t.Errorf("hashing non existing file was successful")
	}
}
