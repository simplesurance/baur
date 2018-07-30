package postgres

import (
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/simplesurance/baur/storage"
)

const sqlConStr = "postgresql://baur@jenkins.sisu.sh:5432/baur?sslmode=disable"

func TestInsertAppIfNotExist(t *testing.T) {
	appName := "TestInsertAppIfNotExist " + xid.New().String()

	c, err := New(sqlConStr)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := c.db.Begin()
	if err != nil {
		t.Fatal("starting transaction failed:", err)
	}

	defer tx.Rollback()

	appID, err := insertAppIfNotExist(tx, appName)
	if err != nil {
		t.Fatal("insertAppIfNotExist() failed:", err)
	}

	if appID == -1 {
		t.Fatal("insertAppIfNotExist returned -1 as id")
	}

	appIDReinsert, err := insertAppIfNotExist(tx, appName)
	if err != nil {
		t.Fatal("insertAppIfNotExist() failed when record already exists", err)
	}

	if appID != appIDReinsert {
		t.Fatalf("insertAppIfNotExist returned a different id when on 2. insert, %q vs %q",
			appID, appIDReinsert)
	}

}

func TestSave(t *testing.T) {
	c, err := New(sqlConStr)
	if err != nil {
		t.Fatal(err)
	}

	start := time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC)
	end := time.Date(2018, 1, 1, 2, 1, 1, 1, time.UTC)

	b := storage.Build{
		AppName:        "baur-unittest",
		StartTimeStamp: start,
		StopTimeStamp:  end,
		Outputs: []*storage.Output{
			&storage.Output{
				Name:           "baur-unittest/dist/artifact.tar.xz",
				Type:           storage.S3Output,
				URI:            "http://test.de",
				Digest:         "sha384:c825bb06739ba6b41f6cc0c123a5956bd65be9e22d51640a0460e0b16eb4523af4d68a1b56d63fd67dab484a0796fc69",
				SizeBytes:      64,
				UploadDuration: time.Duration(5 * time.Second),
			},
		},
		Inputs: []*storage.Input{
			&storage.Input{
				Digest: "890",
				URL:    "file://baur-unittest/file1.xyz",
			},

			&storage.Input{
				Digest: "a3",
				URL:    "file://baur-unittest/file2.xyz",
			},
		},
		TotalInputDigest: "123",
		VCSState: storage.VCSState{
			CommitID: "123",
			IsDirty:  true,
		},
	}

	err = c.Save(&b)
	if err != nil {
		t.Error("Saving build failed:", err)
	}
}
