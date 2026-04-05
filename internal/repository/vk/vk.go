package vk

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"inst_parser/internal/models"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/api/params"
	"github.com/SevereCloud/vksdk/v2/object"
)

type Repository struct {
	logger *slog.Logger
	vkApi  *api.VK
}

func NewRepository(logger *slog.Logger, accessToken string) *Repository {
	vk := api.NewVK(accessToken)
	return &Repository{
		logger: logger,
		vkApi:  vk,
	}
}

func (r *Repository) GroupID(groupName string) (string, error) {
	const vkApiMethod = "groups.getById"
	var groupInfo []struct {
		ID int `json:"id"`
	}

	params := api.Params{
		"group_id": groupName,
	}

	if err := r.vkApi.RequestUnmarshal(vkApiMethod, &groupInfo, params); err != nil {
		return "", fmt.Errorf("failed to get group info: group_id = %s, err = %w", groupName, err)
	}

	if len(groupInfo) == 0 {
		return "", fmt.Errorf("group not found: group_id = %s", groupName)
	}

	return strconv.Itoa(-groupInfo[0].ID), nil // Для групп ID отрицательный
}

func (r *Repository) PostInfo(postID string) (*models.VKClipInfo, error) {
	builder := params.NewWallGetByIDBuilder()
	builder.Posts([]string{postID})

	//var response models.VKWallResponse
	response, err := r.vkApi.WallGetByID(builder.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to get clip info: %w", err)
	}

	if len(response) == 0 {
		return nil, fmt.Errorf("post not found")
	}

	var (
		description     string
		comments, views int
	)

	//if len(response[0].Attachments) > 0 {
	//	if response[0].Attachments[0].Type == "video" {
	//
	//	}
	//}

	//todo добавить условие && response[0].Attachments[0].Type == "video"
	if len(response[0].Attachments) > 0 {
		description = response[0].Attachments[0].Video.Description
		views = response[0].Attachments[0].Video.Views
		comments = response[0].Attachments[0].Video.Comments

		if views == 0 {
			views = response[0].Views.Count
		}

		if description == "" {
			description = response[0].Text
		}

		if comments == 0 {
			comments = response[0].Comments.Count
		}

		return &models.VKClipInfo{
			Description: description,
			OwnerID:     response[0].OwnerID,
			ClipID:      response[0].ID,
			Views:       views,
			Likes:       response[0].Likes.Count,
			Comments:    comments,
			Shares:      response[0].Reposts.Count,
			Date:        time.Unix(int64(response[0].Date), 0),
		}, nil
	}

	postInfo := &models.VKClipInfo{
		Description: response[0].Text,
		OwnerID:     response[0].OwnerID,
		ClipID:      response[0].ID,
		Views:       response[0].Views.Count,
		Likes:       response[0].Likes.Count,
		Comments:    response[0].Comments.Count,
		Shares:      response[0].Reposts.Count,
		Date:        time.Unix(int64(response[0].Date), 0),
	}

	return postInfo, nil
}

func (r *Repository) ClipInfo(ownerID, clipID int) (*models.VKClipInfo, error) {
	const vkApiMethod = "video.get"
	params := api.Params{
		"videos":   fmt.Sprintf("%d_%d", ownerID, clipID),
		"extended": 1,
	}

	var response api.VideoGetResponse
	if err := r.vkApi.RequestUnmarshal(vkApiMethod, &response, params); err != nil {
		return nil, fmt.Errorf("failed to get clip info: %w", err)
	}

	if response.Count == 0 {
		return nil, fmt.Errorf("clip not found")
	}

	item := response.Items[0]

	clipInfo := &models.VKClipInfo{
		Description: item.Description,
		OwnerID:     item.OwnerID,
		ClipID:      item.ID,
		Views:       item.Views,
		Likes:       item.Likes.Count,
		Comments:    item.Comments,
		Shares:      item.Reposts.Count,
		Date:        time.Unix(int64(item.Date), 0),
		DownloadURL: findHighQualityLink(item),
	}

	return clipInfo, nil
}

func findHighQualityLink(item object.VideoVideo) string {
	if item.Files.Mp4_2160 != "" {
		return item.Files.Mp4_2160
	}

	if item.Files.Mp4_1440 != "" {
		return item.Files.Mp4_1440
	}

	if item.Files.Mp4_1080 != "" {
		return item.Files.Mp4_1080
	}

	if item.Files.Mp4_720 != "" {
		return item.Files.Mp4_720
	}

	if item.Files.Mp4_480 != "" {
		return item.Files.Mp4_480
	}

	if item.Files.Mp4_360 != "" {
		return item.Files.Mp4_360
	}

	if item.Files.Mp4_240 != "" {
		return item.Files.Mp4_240
	}

	return ""
}
