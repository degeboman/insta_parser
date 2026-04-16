package vk

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"inst_parser/internal/models"

	"github.com/SevereCloud/vksdk/v3/api"
	"github.com/SevereCloud/vksdk/v3/object"
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
	const vkApiMethod = "wall.getById"

	params := api.Params{
		"posts": postID,
	}

	var response models.WallGetByIDResponse

	if err := r.vkApi.RequestUnmarshal(vkApiMethod, &response, params); err != nil {
		fmt.Errorf("failed to get post info: post_id = %s, err = %w", postID, err)
		return nil, err
	}

	if len(response.Items) == 0 {
		return nil, fmt.Errorf("post not found")
	}

	var (
		description     string
		comments, views int
	)

	//todo добавить условие && response.Items[0].Attachments[0].Type == "video"
	if len(response.Items[0].Attachments) > 0 {
		description = response.Items[0].Attachments[0].Video.Description
		views = response.Items[0].Attachments[0].Video.Views
		comments = response.Items[0].Attachments[0].Video.Comments

		if views == 0 {
			views = response.Items[0].Views.Count
		}

		if description == "" {
			description = response.Items[0].Text
		}

		if comments == 0 {
			comments = response.Items[0].Comments.Count
		}

		return &models.VKClipInfo{
			Description: description,
			OwnerID:     response.Items[0].OwnerID,
			ClipID:      response.Items[0].ID,
			Views:       views,
			Likes:       response.Items[0].Likes.Count,
			Comments:    comments,
			Shares:      response.Items[0].Reposts.Count,
			Date:        time.Unix(int64(response.Items[0].Date), 0),
			OwnerUrl:    channelUrl(response.Items[0].OwnerID, response.Items[0].ID),
			ErID:        response.Items[0].AuthorAd.AdMarker,
		}, nil
	}

	postInfo := &models.VKClipInfo{
		Description: response.Items[0].Text,
		OwnerID:     response.Items[0].OwnerID,
		ClipID:      response.Items[0].ID,
		Views:       response.Items[0].Views.Count,
		Likes:       response.Items[0].Likes.Count,
		Comments:    response.Items[0].Comments.Count,
		Shares:      response.Items[0].Reposts.Count,
		Date:        time.Unix(int64(response.Items[0].Date), 0),
		OwnerUrl:    channelUrl(response.Items[0].OwnerID, response.Items[0].ID),
		ErID:        response.Items[0].AuthorAd.AdMarker,
	}

	return postInfo, nil
}

func (r *Repository) ClipInfo(ownerID, clipID int) (*models.VKClipInfo, error) {
	const vkApiMethod = "video.get"
	params := api.Params{
		"videos":   fmt.Sprintf("%d_%d", ownerID, clipID),
		"extended": 1,
	}

	var response models.VideoGetResponse
	if err := r.vkApi.RequestUnmarshal(vkApiMethod, &response, params); err != nil {
		return nil, fmt.Errorf("failed to get clip info: %w", err)
	}

	if response.Count == 0 {
		return nil, fmt.Errorf("clip not found")
	}

	item := response.Items[0]

	var advertiser models.AdvertiserInfo
	if len(item.OrdInfo.Advertisers) > 0 {
		advertiser = item.OrdInfo.Advertisers[0]
	}

	clipInfo := &models.VKClipInfo{
		Description: item.Description,
		OwnerID:     item.OwnerID,
		ClipID:      item.ID,
		Views:       item.Views,
		Likes:       item.Likes.Count,
		Comments:    item.Comments,
		Shares:      item.Reposts.Count,
		Date:        time.Unix(int64(item.Date), 0),
		DownloadURL: findHighQualityLink(item.Files),
		OwnerUrl:    channelUrl(item.OwnerID, item.UserID),
		PostID:      item.PostID,
		ErID:        advertiser.ErID,
	}

	return clipInfo, nil
}

func channelUrl(ownerID, userID int) string {
	return fmt.Sprintf("https://vk.com/club%d", ownerID*(-1))
}

func findHighQualityLink(item object.VideoVideoFiles) string {
	if item.Mp4_2160 != "" {
		return item.Mp4_2160
	}

	if item.Mp4_1440 != "" {
		return item.Mp4_1440
	}

	if item.Mp4_1080 != "" {
		return item.Mp4_1080
	}

	if item.Mp4_720 != "" {
		return item.Mp4_720
	}

	if item.Mp4_480 != "" {
		return item.Mp4_480
	}

	if item.Mp4_360 != "" {
		return item.Mp4_360
	}

	if item.Mp4_240 != "" {
		return item.Mp4_240
	}

	return ""
}
