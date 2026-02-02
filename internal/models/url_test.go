package models

import "testing"

func Test_parseSocialAccountURL(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name            string
		args            args
		wantParsingType ParsingType
		wantAccount     string
		wantErr         bool
	}{
		{
			name: "case 1",
			args: args{
				url: "https://www.instagram.com/ollaserebro",
			},
			wantParsingType: InstagramParsingType,
			wantAccount:     "ollaserebro",
			wantErr:         false,
		},
		{
			name: "case 2",
			args: args{
				url: "https://vk.com/smotri_video_tovarov",
			},
			wantParsingType: VKParsingType,
			wantAccount:     "smotri_video_tovarov",
			wantErr:         false,
		},
		{
			name: "case 3",
			args: args{
				url: "https://vk.ru/smotri_video_tovarov",
			},
			wantParsingType: VKParsingType,
			wantAccount:     "smotri_video_tovarov",
			wantErr:         false,
		},
		{
			name: "case 4",
			args: args{
				url: "https://www.instagram.com/makovkaaaaaa/reels/",
			},
			wantParsingType: InstagramParsingType,
			wantAccount:     "makovkaaaaaa",
			wantErr:         false,
		},
		{
			name: "case 5",
			args: args{
				url: "https://www.instagram.com/mzhelskaya.zhenya?igsh=MTQxaXI5OTBsN2VmZw==",
			},
			wantParsingType: InstagramParsingType,
			wantAccount:     "mzhelskaya.zhenya",
			wantErr:         false,
		},
		{
			name: "case 6",
			args: args{
				url: "https://www.instagram.com/filatova__l?igsh=OHRpMXJnYzJjN2Rt",
			},
			wantParsingType: InstagramParsingType,
			wantAccount:     "filatova__l",
			wantErr:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAccount, gotParsingType, err := ParseSocialAccountURL(tt.args.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSocialAccountURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotParsingType != tt.wantParsingType {
				t.Errorf("ParseSocialAccountURL() gotParsingType = %v, want %v", gotParsingType, tt.wantParsingType)
			}
			if gotAccount != tt.wantAccount {
				t.Errorf("ParseSocialAccountURL() gotAccount = %v, want %v", gotAccount, tt.wantAccount)
			}
		})
	}
}

func TestParseVkClipURL(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name    string
		args    args
		want    int
		want1   int
		wantErr bool
	}{
		{
			name: "case 1",
			args: args{
				url: "https://vk.com/clips-73430300?z=clip-73430300_456240003",
			},
			want:    -73430300,
			want1:   456240003,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := ParseVkClipURL(tt.args.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVkClipURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseVkClipURL() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("ParseVkClipURL() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
