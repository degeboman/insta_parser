package models

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"inst_parser/internal/utils"
)

type YoutubeShortInfoApiResponse struct {
	ID            string `json:"videoId"`
	Title         string `json:"title"`
	LikeCount     int    `json:"likeCount"`
	ViewCount     string `json:"viewCount"`
	Description   string `json:"description"`
	PublishedDate string `json:"publishedDate"`
	PublishDate   string `json:"publishDate"`
	CommentCount  string `json:"commentCount"`
	ShareCount    int    `json:"shareCount,omitempty"`
	AccountURL    string
}

////////////////////////////////////////////////////////////////////////////////////////////////
////
//// ORIGINAL API YOUTUBE VIDEOS
////
////////////////////////////////////////////////////////////////////////////////////////////////

type VideosAPIResponse struct {
	Items []VideoItem `json:"items"`
}

type VideoItem struct {
	ID         string     `json:"id"`
	Snippet    Snippet    `json:"snippet"`
	Statistics Statistics `json:"statistics"`
}

// VideoInfo содержит полную информацию о видео
type VideoInfo struct {
	ID            string `json:"videoId"`
	Title         string `json:"title"`
	LikeCount     int    `json:"likeCount"`
	ViewCount     string `json:"viewCount"`
	Description   string `json:"description"`
	PublishedDate string `json:"publishedDate"`
	CommentCount  string `json:"commentCount"`
}

// Snippet YouTube API response structures
type Snippet struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	PublishedAt string `json:"publishedAt"`
	ChannelId   string `json:"channelId"`
}

type Statistics struct {
	ViewCount    string `json:"viewCount"`
	LikeCount    string `json:"likeCount"`
	CommentCount string `json:"commentCount"`
}

////////////////////////////////////////////////////////////////////////////////////////////////
////
//// ORIGINAL API YOUTUBE CHANNEL VIDEOS
////
////////////////////////////////////////////////////////////////////////////////////////////////

type ChannelResponse struct {
	Items []YouTubeChannel `json:"items"`
}

type YouTubeChannel struct {
	ID             string `json:"id"`
	ContentDetails struct {
		RelatedPlaylists struct {
			Uploads string `json:"uploads"`
		} `json:"relatedPlaylists"`
	} `json:"contentDetails"`
}

type PlaylistItemsResponse struct {
	Items         []PlaylistItem `json:"items"`
	NextPageToken string         `json:"nextPageToken"`
}

type PlaylistItem struct {
	Snippet struct {
		ResourceID struct {
			VideoID string `json:"videoId"`
		} `json:"resourceId"`
		Title       string `json:"title"`
		PublishedAt string `json:"publishedAt"`
	} `json:"snippet"`
}

type YoutubeChannelShortsApiResponse struct {
	Meta         YoutubeChannelShortsMeta    `json:"meta"`
	Data         []*YoutubeChannelShortsData `json:"data"`
	Continuation string                      `json:"continuation"`
	Msg          string                      `json:"msg"`
}

type YoutubeChannelShortsMeta struct {
	VideoCount string `json:"videoCount"`
}

type YoutubeChannelShortsData struct {
	Type    string `json:"type"`
	VideoId string `json:"videoId"`
}

func (y YoutubeShortInfoApiResponse) ToResultRow(originalUrl string) *ResultRowUrl {
	likes := y.LikeCount
	comments, err := strconv.Atoi(y.CommentCount)
	if err != nil {
		log.Println("Error converting views count to int")
		comments = 0
	}
	views, err := strconv.Atoi(y.ViewCount)
	if err != nil {
		log.Println("Error converting views count to int")
		views = 0
	}
	shares := 0

	var publishDate string

	if y.PublishedDate == "" {
		publishDate = y.PublishDate
	} else if y.PublishDate == "" {
		publishDate = y.PublishedDate
	}

	result := &ResultRowUrl{
		OwnerUrl:    y.AccountURL,
		URL:         originalUrl,
		Description: fmt.Sprintf("%s.%s", y.Title, y.Description),
		Views:       int64(views),
		Likes:       int64(likes),
		Comments:    int64(comments),
		Shares:      int64(shares),
		ER:          utils.GetER(int64(likes), int64(shares), int64(comments), int64(views)),
		Virality:    utils.GetVirality(int64(shares), int64(views)),
		ParsingDate: utils.ParsingDate(),
		PublishDate: utils.FormatParsingDate(publishDate),
	}

	return result
}

func (y YoutubeShortInfoApiResponse) ToInterface(accountURL string) []interface{} {
	likes := y.LikeCount
	comments, err := strconv.Atoi(y.CommentCount)
	if err != nil {
		log.Println("Error converting comments count to int")
		comments = 0
	}
	views, err := strconv.Atoi(y.ViewCount)
	if err != nil {
		log.Println("Error converting views count to int")
		views = 0
	}
	shares := 0

	var publishDate string

	if y.PublishedDate == "" {
		publishDate = y.PublishDate
	} else if y.PublishDate == "" {
		publishDate = y.PublishedDate
	}

	return []interface{}{
		accountURL,
		fmt.Sprintf("https://www.youtube.com/shorts/%s", y.ID),
		fmt.Sprintf("%s.%s", y.Title, y.Description),
		int64(views),
		int64(likes),
		int64(comments),
		int64(shares),
		utils.GetER(int64(likes), int64(shares), int64(comments), int64(views)),
		utils.GetVirality(int64(shares), int64(views)),
		utils.ParsingDate(),
		publishDate,
	}

}

func ExtractYouTubeShortsID(url string) (string, bool) {
	// Проверяем, содержит ли ссылка /shorts/
	shortsIndex := strings.Index(url, "/shorts/")
	if shortsIndex == -1 {
		return "", false
	}

	// Получаем подстроку после /shorts/
	startIndex := shortsIndex + len("/shorts/")
	idPart := url[startIndex:]

	// Убираем возможные параметры после ?
	if questionMarkIndex := strings.Index(idPart, "?"); questionMarkIndex != -1 {
		idPart = idPart[:questionMarkIndex]
	}

	// Убираем возможные символы в конце (слеши и т.д.)
	idPart = strings.TrimRight(idPart, "/")

	// Проверяем, что ID не пустой
	if idPart == "" {
		return "", false
	}

	return idPart, true
}

func YoutubeShortInfoApiResponseToInterface(data []*YoutubeShortInfoApiResponse, accountUrl string) [][]interface{} {
	values := make([][]interface{}, 0, len(data))

	for i := range data {
		if data == nil {
			continue
		}
		values = append(values, data[i].ToInterface(accountUrl))
	}

	return values
}
