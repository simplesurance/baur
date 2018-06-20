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
		Artifacts: []*storage.Artifact{
			&storage.Artifact{
				Name:           "baur-unittest/dist/artifact.tar.xz",
				Type:           storage.S3Artifact,
				URI:            "http://test.de",
				Digest:         "5678",
				SizeBytes:      64,
				UploadDuration: time.Duration(5 * time.Second),
			},
		},
		Sources: []*storage.Source{
			&storage.Source{
				Digest:       "890",
				RelativePath: "baur-unittest/file1.xyz",
			},

			&storage.Source{
				Digest:       "a3",
				RelativePath: "baur-unittest/file2.xyz",
			},
		},
		TotalSrcDigest: "123",
	}

	err = c.Save(&b)
	if err != nil {
		t.Error("Saving build failed:", err)
	}
}
