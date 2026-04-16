package vk

import (
	"reflect"
	"testing"

	"inst_parser/internal/models"
)

func TestRepository_getAdvertiserInfo(t *testing.T) {
	type args struct {
		eridURL string
	}
	tests := []struct {
		name    string
		args    args
		want    *models.AdvertiserInfoFromUrl
		wantErr bool
	}{
		{
			name: "case 1",
			args: args{eridURL: "https://ord.vk.com/adv_info?erid=2Vtzqv5EPsC&s=nszUyj9ZZ_2FrjxQgwSGIOLQ5dHVmW965RdC8baZb74&t=1776330436"},
			want: &models.AdvertiserInfoFromUrl{
				INN:  "503018954840",
				Name: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Repository{}
			got, err := r.getAdvertiserInfo(tt.args.eridURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("getAdvertiserInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAdvertiserInfo() got = %v, want %v", got, tt.want)
			}
		})
	}
}
