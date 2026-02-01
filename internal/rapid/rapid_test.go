package rapid

import (
	"fmt"
	"inst_parser/internal/models"
	"net/http"
	"sync"
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
		rapidAPIKey           string
		httpClient            *http.Client
		processingInstagramMu sync.Mutex
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
				processingInstagramMu: sync.Mutex{},
				rapidAPIKey:           "d0b61d8381msha85339d3ad2c820p1919e1jsn203a3291a43f",
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
			got := s.ParseUrl("1J-_Ka6O8EGWjwbsHxOxdve-H2CFPUXTIeV7phAOlK-8", []*models.UrlInfo{{URL: tt.args.reelURL}})

			fmt.Println(got)
		})
	}
}
