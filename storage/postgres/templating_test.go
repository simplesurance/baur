package postgres

import (
	"fmt"
	"testing"
)

func Test_stringHasPlaceholder(t *testing.T) {
	type args struct {
		string string
		key    string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "positive",
			args: args{
				string: "select * from cars where {{foo}}",
				key:    "foo",
			},
			want: true,
		},
		{
			name: "negative",
			args: args{
				string: "select * from cars where {{foo}}",
				key:    "boo",
			},
			want: false,
		},
		{
			name: "spaces",
			args: args{
				string: "select * from cars where {{foo bar}}",
				key:    "foo bar",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stringHasPlaceholder(tt.args.string, tt.args.key); got != tt.want {
				t.Errorf("stringHasPlaceholder() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_wrapKey(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "simple",
			args: args{"sup"},
			want: fmt.Sprintf(PlaceholderTemplate, "sup"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WrapKey(tt.args.key); got != tt.want {
				t.Errorf("WrapKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_replacePlaceholder(t *testing.T) {
	type args struct {
		key    string
		string string
		value  string
		n      int
	}
	tests := []struct {
		name    string
		input   args
		want    string
		wantErr bool
	}{
		{
			name:    "positive",
			input:   args{"foo", "bar {{foo}} baz {{foo}}", "wee", 1},
			want:    "bar wee baz {{foo}}",
			wantErr: false,
		},
		{
			name:    "err",
			input:   args{"boo", "bar {{woop}} baz {{foo}}", "wee", 1},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := setPlaceholderValue(tt.input.key, tt.input.string, tt.input.value, tt.input.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("replacePlaceholder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("replacePlaceholder() = %v, want %v", got, tt.want)
			}
		})
	}
}
