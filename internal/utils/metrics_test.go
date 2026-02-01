package utils

import "testing"

func Test_getER(t *testing.T) {
	type args struct {
		likes    int64
		shares   int64
		comments int64
		views    int64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "case 1",
			args: args{
				likes:    0,
				shares:   0,
				comments: 0,
				views:    0,
			},
			want: "0",
		},
		{
			name: "case 2",
			args: args{
				likes:    3891,
				shares:   7043,
				comments: 18,
				views:    173514,
			},
			want: "6.31%",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetER(tt.args.likes, tt.args.shares, tt.args.comments, tt.args.views); got != tt.want {
				t.Errorf("GetER() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getVirality(t *testing.T) {
	type args struct {
		shares int64
		views  int64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "case 1",
			args: args{
				shares: 7043,
				views:  173514,
			},
			want: "4.06%",
		},
		{
			name: "case 2",
			args: args{
				shares: 3891,
				views:  0,
			},
			want: "0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetVirality(tt.args.shares, tt.args.views); got != tt.want {
				t.Errorf("GetVirality() = %v, want %v", got, tt.want)
			}
		})
	}
}
