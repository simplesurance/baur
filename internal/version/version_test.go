package version

import (
	"reflect"
	"testing"
)

func TestSemVerFromString(t *testing.T) {
	tests := []struct {
		arg     string
		want    *SemVer
		wantErr bool
	}{

		{
			arg:     "",
			wantErr: true,
		},

		{
			arg:     "abcdas",
			wantErr: true,
		},

		{
			arg:     "a3.4.5",
			wantErr: true,
		},

		{
			arg:     "3.c.5",
			wantErr: true,
		},

		{
			arg:     "3.4-asd",
			wantErr: true,
		},

		{
			arg: "1",
			want: &SemVer{
				Major:    1,
				Minor:    0,
				Patch:    0,
				Appendix: "",
			},
			wantErr: false,
		},

		{
			arg: "0.11",
			want: &SemVer{
				Major:    0,
				Minor:    11,
				Patch:    0,
				Appendix: "",
			},
			wantErr: false,
		},

		{
			arg: "1.2",
			want: &SemVer{
				Major:    1,
				Minor:    2,
				Patch:    0,
				Appendix: "",
			},
			wantErr: false,
		},

		{
			arg: "1.2.3",
			want: &SemVer{
				Major:    1,
				Minor:    2,
				Patch:    3,
				Appendix: "",
			},
			wantErr: false,
		},

		{
			arg: "1.2.3-hello.yo",
			want: &SemVer{
				Major:    1,
				Minor:    2,
				Patch:    3,
				Appendix: "hello.yo",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			got, err := New(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("SemVerFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SemVerFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}
