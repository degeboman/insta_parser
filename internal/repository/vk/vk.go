package vk

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"inst_parser/internal/models"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/api/params"
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

func (v *Repository) GroupID(groupName string) (string, error) {
	const vkApiMethod = "groups.getById"
	var groupInfo []struct {
		ID int `json:"id"`
	}

	params := api.Params{
		"group_id": groupName,
	}

	err := v.vkApi.RequestUnmarshal(vkApiMethod, &groupInfo, params)
	if err != nil {
		return "", fmt.Errorf("failed to get group info: group_id = %s, err = %w", groupName, err)
	}

	if len(groupInfo) == 0 {
		return "", fmt.Errorf("group not found: group_id = %s", groupName)
	}

	return strconv.Itoa(-groupInfo[0].ID), nil // Для групп ID отрицательный
}

func (v *Repository) PostInfo(postID string) (*models.VKClipInfo, error) {
	builder := params.NewWallGetByIDBuilder()
	builder.Posts([]string{postID})

	//var response models.VKWallResponse
	response, err := v.vkApi.WallGetByID(builder.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to get clip info: %w", err)
	}

	if len(response) == 0 {
		return nil, fmt.Errorf("post not found")
	}

	var description string
	if len(response[0].Attachments) > 0 {
		description = response[0].Attachments[0].Video.Description

		return &models.VKClipInfo{
			Description: description,
			OwnerID:     response[0].OwnerID,
			ClipID:      response[0].ID,
			Views:       response[0].Attachments[0].Video.Views,
			Likes:       response[0].Likes.Count,
			Comments:    response[0].Attachments[0].Video.Comments,
			Shares:      response[0].Reposts.Count,
			Date:        time.Unix(int64(response[0].Date), 0),
		}, nil
	}

	postInfo := &models.VKClipInfo{
		Description: description,
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

func (v *Repository) ClipInfo(ownerID, clipID int) (*models.VKClipInfo, error) {
	const vkApiMethod = "video.get"
	params := api.Params{
		"videos": fmt.Sprintf("%d_%d", ownerID, clipID),
	}

	var response struct {
		Count int `json:"count"`
		Items []struct {
			ID          int    `json:"id"`
			OwnerID     int    `json:"owner_id"`
			Title       string `json:"title"`
			Description string `json:"description"`
			Duration    int    `json:"duration"`
			Views       int    `json:"views"`
			Comments    int    `json:"comments"`
			Date        int    `json:"date"`
			Image       []struct {
				URL    string `json:"url"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"image"`
			Likes struct {
				Count int `json:"count"`
			} `json:"likes"`
			Reposts struct {
				Count int `json:"count"`
			} `json:"reposts"`
		} `json:"items"`
	}

	if err := v.vkApi.RequestUnmarshal(vkApiMethod, &response, params); err != nil {
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
	}

	return clipInfo, nil
}
