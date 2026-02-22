package models

import "testing"

func TestExtractTiktokVideoID(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{{
		name: "success",
		args: args{
			url: "https://www.tiktok.com/@alenochka_neapina/video/7575144767089069368",
		},
		want:    "7575144767089069368",
		wantErr: false,
	}, {
		//
		name: "success 2",
		args: args{
			url: "https://www.tiktok.com/@naz.insemeyy/video/7578847141221698827?_r=1&_t=ZM-91rCg4Aur9f",
		},
		want:    "7578847141221698827",
		wantErr: false,
	},
		{
			name: "success 3",
			args: args{
				url: "https://vm.tiktok.com/ZGdaodcMU/",
			},
			want:    "ZGdaodcMU",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractTiktokVideoID(tt.args.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractTiktokVideoID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractTiktokVideoID() got = %v, want %v", got, tt.want)
			}
		})
	}
}
