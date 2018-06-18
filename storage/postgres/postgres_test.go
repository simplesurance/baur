package postgres

import (
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/simplesurance/baur/storage"
)

const sqlConStr = "postgresql://baur:@jenkins.sisu.sh:5432/baur?sslmode=disable"

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

func TestInsertSourceIfNotExist(t *testing.T) {
	c, err := New(sqlConStr)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := c.db.Begin()
	if err != nil {
		t.Fatal("starting transaction failed:", err)
	}

	defer tx.Rollback()

	src := storage.Source{
		Hash:         xid.New().String(),
		RelativePath: "/tmp/hello",
	}

	srcID, err := insertSourceIfNotExist(tx, &src)
	if err != nil {
		t.Fatal("insertSourceIfNotExist failed:", err)
	}

	if srcID == -1 {
		t.Fatal("insertSourceIfNotExist returned id -1")
	}

	srcIDSecond, err := insertSourceIfNotExist(tx, &src)
	if err != nil {
		t.Fatal("insertSourceIfNotExist fails when record exist:", err)
	}

	if srcID != srcIDSecond {
		t.Fatalf("insertAppIfNotExist() returned differend id on 2. call, %q vs %q",
			srcID, srcIDSecond)
	}
}

func TestInsertBuildAndArtifact(t *testing.T) {
	appName := "TestInsertBuildAndArtifact " + xid.New().String()

	c, err := New(sqlConStr)
	if err != nil {
		t.Fatal(err)
	}

	defer c.Close()

	tx, err := c.db.Begin()
	if err != nil {
		t.Fatal("starting transaction failed:", err)
	}
	defer tx.Rollback()

	start := time.Date(2018, 1, 1, 1, 1, 1, 1, time.UTC)
	end := time.Date(2018, 1, 1, 2, 1, 1, 1, time.UTC)

	appID, err := insertAppIfNotExist(tx, appName)
	if err != nil {
		t.Fatal("insertAppIfNotExist() failed:", err)
	}

	id, err := insertBuild(tx, appID, &storage.Build{
		StopTimeStamp:  start,
		StartTimeStamp: end,
		TotalSrcHash:   "123",
	})

	if err != nil {
		t.Fatal("insertBuild() failed:", err)
	}

	if id == -1 {
		t.Error("inserBuid() returned id -1")
	}

	artID, err := insertArtifact(tx, id, &storage.Artifact{
		Type:      storage.S3Artifact,
		URL:       "http://yo",
		Hash:      "567",
		SizeBytes: 64,
	})
	if err != nil {
		t.Fatal("insertArtifact() failed:", err)
	}

	if artID == -1 {
		t.Error("artifact id is -1")
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
				URL:            "http://test.de",
				Hash:           "5678",
				SizeBytes:      64,
				UploadDuration: time.Duration(5 * time.Second),
			},
		},
		Sources: []*storage.Source{
			&storage.Source{
				Hash:         "890",
				RelativePath: "baur-unittest/file1.xyz",
			},

			&storage.Source{
				Hash:         "a3",
				RelativePath: "baur-unittest/file2.xyz",
			},
		},
		TotalSrcHash: "123",
	}

	err = c.Save(&b)
	if err != nil {
		t.Error("Saving build failed:", err)
	}
}
