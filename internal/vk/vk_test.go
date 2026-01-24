package vk

import "testing"

func Test_parseGroupURL(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "case 1",
			args: args{
				url: "https://vk.com/36hockey",
			},
			want:    "36hockey",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGroupURL(tt.args.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseGroupURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseGroupURL() got = %v, want %v", got, tt.want)
			}
		})
	}
}
