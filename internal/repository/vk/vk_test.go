package vk

import "testing"

func TestDownloadVideo(t *testing.T) {
	type args struct {
		name string
		url  string
		dir  string
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
				name: "test",
				url:  "https://vkvd674.okcdn.ru/?expires=1775053931627&srcIp=37.215.8.188&pr=41&srcAg=UNKNOWN&ms=178.237.23.153&type=5&sig=DerEZ30IOOo&ct=0&urls=176.112.172.130&clientType=13&appId=512000384397&zs=25&id=8947271141983",
				dir:  "",
			},
			want:    "ff",
			wantErr: false,
		},
		{
			name: "download video",
			args: args{
				name: "test",
				url:  "https://scontent-iad3-2.xx.fbcdn.net/o1/v/t2/f2/m69/AQNIOWG7bW4kNp_zSUFH6LSRGdORKOiTdA5TPEflcwPtSTeXaUl5IAmh9Of9C-kUZVze7Wf4fRyvVJoKfTz_YJDu.mp4?strext=1&_nc_cat=106&_nc_sid=8bf8fe&_nc_ht=scontent-iad3-2.xx.fbcdn.net&_nc_ohc=6iA95JANfIoQ7kNvgF-AVLV&efg=eyJ2ZW5jb2RlX3RhZyI6Inhwdl9wcm9ncmVzc2l2ZS5BVURJT19PTkxZLi5DMy4wLnByb2dyZXNzaXZlX2F1ZGlvIiwieHB2X2Fzc2V0X2lkIjo1NTM5NTY5MjM5MTE4NzksInVybGdlbl9zb3VyY2UiOiJ3d3cifQ%3D%3D&ccb=9-4&_nc_zt=28&oh=00_AYCEgsChCWjeects3UdCTdR-p-ZLjaQCA7yy5ccsD7XJuw&oe=67637AF1",
				dir:  "",
			},
			want:    "ff",
			wantErr: false,
		},
	}
	repo := NewRepository(nil, "")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.DownloadVideo(tt.args.name, tt.args.url, tt.args.dir)
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
