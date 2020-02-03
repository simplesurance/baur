package docker

import (
	"testing"
)

func Test_parseRepositoryURI(t *testing.T) {
	tests := []struct {
		name           string
		destArg        string
		wantRepository string
		wantTag        string
		wantErr        bool
	}{
		{
			name:           "shortPath",
			destArg:        "simplesurance/calculator:latest",
			wantRepository: "simplesurance/calculator",
			wantTag:        "latest",
		},
		{
			name:           "longRepoPath",
			destArg:        "simplesurance/app/math/calculator:latest",
			wantRepository: "simplesurance/app/math/calculator",
			wantTag:        "latest",
		},
		{
			name:    "longRepoPathMissingTag",
			destArg: "simplesurance/app/math/calculator",
			wantErr: true,
		},
		{
			name:    "OnlyTag",
			destArg: "latest",
			wantErr: true,
		},
		{
			name:    "emptyArg",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRepository, gotTag, err := parseRepositoryURI(tt.destArg)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRepositoryURI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotRepository != tt.wantRepository {
				t.Errorf("parseRepositoryURI() gotRepository = %v, want %v", gotRepository, tt.wantRepository)
			}
			if gotTag != tt.wantTag {
				t.Errorf("parseRepositoryURI() gotTag = %v, want %v", gotTag, tt.wantTag)
			}
		})
	}
}
