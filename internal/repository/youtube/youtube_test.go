package youtube

import (
	"context"
	"reflect"
	"testing"
)

func TestGetYouTubeVideoStats(t *testing.T) {
	type args struct {
		ctx     context.Context
		videoID string
		apiKey  string
	}
	tests := []struct {
		name    string
		args    args
		want    *Statistics
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				ctx:     context.Background(),
				apiKey:  "AIzaSyDRZTXVLHbHYahrzIgSuDHUjqq0ThHyR7s",
				videoID: "-glmnleoNkw",
			},
			want:    &Statistics{},
			wantErr: false,
		},
		{
			name: "ok",
			args: args{
				ctx:     context.Background(),
				apiKey:  "",
				videoID: "2EMmfcZ_UuY",
			},
			want:    &Statistics{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetYouTubeVideoStats(tt.args.ctx, tt.args.videoID, tt.args.apiKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetYouTubeVideoStats() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetYouTubeVideoStats() got = %v, want %v", got, tt.want)
			}
		})
	}
}
