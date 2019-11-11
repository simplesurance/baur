package postgres

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/rs/xid"

	"github.com/simplesurance/baur/storage"
)

var sqlConStr string

var build = storage.Build{
	Application:    storage.Application{Name: "baur-unittest"},
	StartTimeStamp: time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC),
	StopTimeStamp:  time.Date(2018, 1, 1, 2, 1, 1, 1, time.UTC),
	Outputs: []*storage.Output{
		{
			Name: "baur-unittest/dist/artifact.tar.xz",
			Type: storage.FileArtifact,
			Upload: storage.Upload{
				URI:            "http://test.de",
				UploadDuration: 5 * time.Second,
			},
			Digest:    "sha384:c825bb06739ba6b41f6cc0c123a5956bd65be9e22d51640a0460e0b16eb4523af4d68a1b56d63fd67dab484a0796fc69",
			SizeBytes: 64,
		},
	},
	Inputs: []*storage.Input{
		{
			Digest: "890",
			URI:    "baur-unittest/file1.xyz",
		},

		{
			Digest: "a3",
			URI:    "baur-unittest/file2.xyz",
		},
	},
	TotalInputDigest: "123",
	VCSState: storage.VCSState{
		CommitID: "123",
		IsDirty:  true,
	},
}

func TestMain(m *testing.M) {
	sqlConStr = os.Getenv("BAUR_POSTGRESQL_URL")
	if sqlConStr == "" {
		fmt.Println("BAUR_POSTGRESQL_URL environment variable not set")
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestInsertAppIfNotExist(t *testing.T) {
	app := storage.Application{
		ID:   -1,
		Name: "TestInsertAppIfNotExist " + xid.New().String(),
	}

	c, err := New(sqlConStr)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := c.Db.Begin()
	if err != nil {
		t.Fatal("starting transaction failed:", err)
	}

	// nolint: errcheck
	defer tx.Rollback()

	err = insertAppIfNotExist(tx, &app)
	if err != nil {
		t.Fatal("insertAppIfNotExist() failed:", err)
	}

	if app.ID == -1 {
		t.Fatal("insertAppIfNotExist returned -1 as id")
	}
	prevID := app.ID

	err = insertAppIfNotExist(tx, &app)
	if err != nil {
		t.Fatal("insertAppIfNotExist() failed when record already exists", err)
	}

	if app.ID != prevID {
		t.Fatalf("insertAppIfNotExist returned a different id when on 2. insert, %q vs %q",
			app.ID, prevID)
	}

}

func TestSave(t *testing.T) {
	c, err := New(sqlConStr)
	if err != nil {
		t.Fatal(err)
	}

	err = c.Save(&build)
	if err != nil {
		t.Error("Saving build failed:", err)
	}
}

func TestGetSameTotalInputDigestsForAppBuilds(t *testing.T) {
	c, err := New(sqlConStr)
	if err != nil {
		t.Fatal(err)
	}

	b1 := build
	b1.Application.Name = xid.New().String()
	b1.TotalInputDigest = xid.New().String()

	b2 := b1
	b2.TotalInputDigest = xid.New().String()

	err = c.Save(&b1)
	if err != nil {
		t.Fatal("Saving b1 failed:", err)
	}

	digests, err := c.GetSameTotalInputDigestsForAppBuilds(b1.Application.Name, build.StartTimeStamp)
	if err != nil {
		t.Errorf("returned error %q  when no builds with same input digest exist, expected no error", err)
	}
	if len(digests) != 0 {
		t.Errorf("returned %d digests, expected 0, if none exist with same input digest", len(digests))
	}

	err = c.Save(&b2)
	if err != nil {
		t.Fatal("Saving b2 failed:", err)
	}

	digests, err = c.GetSameTotalInputDigestsForAppBuilds(b1.Application.Name, build.StartTimeStamp)
	if err != nil {
		t.Errorf("returned error %q expected no error when no builds with same input digest exist, expected no error", err)
	}
	if len(digests) != 0 {
		t.Errorf("returned %d digests, expected 0, if only builds with different input digest exist", len(digests))
	}

	err = c.Save(&b1)
	if err != nil {
		t.Fatal("Saving b1 a second time failed:", err)
	}

	digests, err = c.GetSameTotalInputDigestsForAppBuilds(b1.Application.Name, build.StartTimeStamp)
	if err != nil {
		t.Error("returned an error instead of 1 digests:", err)
	}

	if len(digests) != 1 {
		t.Errorf("returned %d digests, expected 1", len(digests))
	}
}
