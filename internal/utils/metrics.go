package utils

import "fmt"

func GetER(likes, shares, comments, views int64) string {
	if likes+shares+comments <= 0 || views <= 0 {
		return "0"
	}

	return fmt.Sprintf("%.2f%%", float64(likes+shares+comments)/float64(views)*100)
}

func GetVirality(shares, views int64) string {
	if shares <= 0 || views <= 0 {
		return "0"
	}
	return fmt.Sprintf("%.2f%%", float64(shares)/float64(views)*100)
}
