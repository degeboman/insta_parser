package vk

import "testing"

func TestDownloadVideo(t *testing.T) {
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
			name: "download video",
			args: args{
				url: "https://vkvd674.okcdn.ru/?expires=1775053931627&srcIp=37.215.8.188&pr=41&srcAg=UNKNOWN&ms=178.237.23.153&type=5&sig=DerEZ30IOOo&ct=0&urls=176.112.172.130&clientType=13&appId=512000384397&zs=25&id=8947271141983",
			},
			want:    "ff",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DownloadVideo(tt.args.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("DownloadVideo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DownloadVideo() got = %v, want %v", got, tt.want)
			}
		})
	}
}
