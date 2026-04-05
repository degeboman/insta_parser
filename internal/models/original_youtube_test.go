package models

import "testing"

func TestExtractYouTubeShortsID(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 bool
	}{
		{
			name: "case 1",
			args: args{
				url: "https://www.youtube.com",
			},
			want:  "",
			want1: false,
		},
		{
			name: "case 2",
			args: args{
				url: "https://youtube.com/shorts/5CHd6h1-Zps?si=SJx8dwcBw88NrL9j",
			},
			want:  "5CHd6h1-Zps",
			want1: true,
		},
		{
			name: "case 3",
			args: args{
				url: "https://www.youtube.com/shorts/BJ162nWYZ_8",
			},
			want:  "BJ162nWYZ_8",
			want1: true,
		},
		{
			name: "case 4",
			args: args{
				url: "https://youtube.com/shorts/oKWtN_lNnUQ?si=AJu04Se6TuMOOMeP",
			},
			want:  "oKWtN_lNnUQ",
			want1: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := ExtractYouTubeShortsID(tt.args.url)
			if got != tt.want {
				t.Errorf("ExtractYouTubeShortsID() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("ExtractYouTubeShortsID() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
