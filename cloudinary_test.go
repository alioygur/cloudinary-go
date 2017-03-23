package cloudinary

import (
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	type args struct {
		uri string
	}
	tests := []struct {
		name    string
		args    args
		want    *Cloudinary
		wantErr bool
	}{
		{"without scheme", args{"apikey:apisecret@cloudname"}, nil, true},
		{"with wrong scheme", args{"wrongscheme://apikey:apisecret@cloudname"}, nil, true},
		{"without cloudname", args{"cloudinary://apikey:apisecret@"}, nil, true},
		{"without apikey", args{"cloudinary://:apisecret@cloudname"}, nil, true},
		{"without secret", args{"cloudinary://apikey:@cloudname"}, nil, true},
		{"with good params", args{"cloudinary://apikey:apisecret@cloudname"}, &Cloudinary{"cloudname", "apikey", "apisecret"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}
