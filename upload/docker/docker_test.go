package docker

import (
	"testing"
)

func Test_parseRepositoryURI(t *testing.T) {
	tests := []struct {
		name           string
		destArg        string
		wantServer     string
		wantRepository string
		wantTag        string
		wantErr        bool
	}{
		{
			name:           "noServer",
			destArg:        "simplesurance/calculator:latest",
			wantServer:     "",
			wantRepository: "simplesurance/calculator",
			wantTag:        "latest",
		},

		{
			name:           "server",
			destArg:        "localhost/simplesurance/calculator:latest",
			wantServer:     "localhost",
			wantRepository: "simplesurance/calculator",
			wantTag:        "latest",
		},

		{
			name:           "serverWithPort",
			destArg:        "localhost:1234/simplesurance/calculator:latest",
			wantServer:     "localhost:1234",
			wantRepository: "simplesurance/calculator",
			wantTag:        "latest",
		},
		{
			name:           "serverWithPortLongRepo",
			destArg:        "localhost/simplesurance/app/calculator:latest",
			wantServer:     "localhost",
			wantRepository: "simplesurance/app/calculator",
			wantTag:        "latest",
		},
		{
			name:    "emptyArg",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotServer, gotRepository, gotTag, err := parseRepositoryURI(tt.destArg)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRepositoryURI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotServer != tt.wantServer {
				t.Errorf("parseRepositoryURI() gotServer = %v, want %v", gotServer, tt.wantServer)
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
