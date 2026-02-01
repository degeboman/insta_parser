package models

import "testing"

func TestIsAvailableByParsingType(t *testing.T) {
	type args struct {
		url          string
		parsingTypes []ParsingType
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "case 1",
			args: args{
				url:          "https://www.instagram.com/reel/DOEGKscjAWx/?igsh=MXZibWx0N2Y5aHB3eA==",
				parsingTypes: []ParsingType{InstagramParsingType},
			},
			want: true,
		},
		{
			name: "case 2",
			args: args{
				url:          "https://www.instagram.com/p/DPQkbQXDULV/",
				parsingTypes: []ParsingType{InstagramParsingType},
			},
			want: true,
		},
		{
			name: "case 3",
			args: args{
				url:          "https://youtube.com/shorts/zg67vNdmoAw?si=8udy79rgqO6c6xCT",
				parsingTypes: []ParsingType{InstagramParsingType},
			},
			want: false,
		},
		{
			name: "case 4",
			args: args{
				url:          "https://vk.com/clip-235319600_456239017",
				parsingTypes: []ParsingType{InstagramParsingType, VKParsingType},
			},
			want: true,
		},
		{
			name: "case 5",
			args: args{
				url:          "https://vk.com/clips-73430300?z=clip-73430300_456240003",
				parsingTypes: []ParsingType{InstagramParsingType, VKParsingType},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAvailableByParsingType(tt.args.url, tt.args.parsingTypes); got != tt.want {
				t.Errorf("IsAvailableByParsingType() = %v, want %v", got, tt.want)
			}
		})
	}
}
