package utils

import "testing"

func TestFormatParsingDate(t *testing.T) {
	type args struct {
		date string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "case 1",
			args: args{
				date: "2025-10-01T18:01:02Z",
			},
			want: "01.10.2025 18:01",
		},
		{
			name: "case 2",
			args: args{
				date: "14.04.2026 11:10",
			},
			want: "14.04.2026 11:10",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatParsingDate(tt.args.date); got != tt.want {
				t.Errorf("FormatParsingDate() = %v, want %v", got, tt.want)
			}
		})
	}
}
