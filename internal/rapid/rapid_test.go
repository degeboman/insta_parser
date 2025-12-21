package rapid

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestService_fetchInstagramDataSafe(t *testing.T) {
	type fields struct {
		rapidAPIKey string
		httpClient  *http.Client
	}
	type args struct {
		reelURL string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "case 1",
			fields: fields{
				rapidAPIKey: "",
				httpClient: &http.Client{
					Timeout: 5 * time.Second,
				},
			},
			args: args{
				reelURL: "https://www.instagram.com/reel/DRhUYUTDIvu/?igsh=MW4zcWY2cmkxeG1wdg==",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				rapidAPIKey: tt.fields.rapidAPIKey,
				httpClient:  tt.fields.httpClient,
			}
			got, err := s.fetchInstagramDataSafe(tt.args.reelURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetchInstagramDataSafe() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Println(got)
		})
	}
}

func TestService_ParseUrl(t *testing.T) {
	type fields struct {
		rapidAPIKey string
		httpClient  *http.Client
	}
	type args struct {
		reelURL string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "case 1",
			fields: fields{
				rapidAPIKey: "",
				httpClient: &http.Client{
					Timeout: 5 * time.Second,
				},
			},
			args: args{
				reelURL: "https://www.instagram.com/reel/DRhUYUTDIvu/?igsh=MW4zcWY2cmkxeG1wdg==",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				rapidAPIKey: tt.fields.rapidAPIKey,
				httpClient:  tt.fields.httpClient,
			}
			got := s.ParseUrl([]string{tt.args.reelURL})

			fmt.Println(got)
		})
	}
}

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
			if got := getER(tt.args.likes, tt.args.shares, tt.args.comments, tt.args.views); got != tt.want {
				t.Errorf("getER() = %v, want %v", got, tt.want)
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
			if got := getVirality(tt.args.shares, tt.args.views); got != tt.want {
				t.Errorf("getVirality() = %v, want %v", got, tt.want)
			}
		})
	}
}
