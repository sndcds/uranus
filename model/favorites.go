package model

type FavoriteList struct {
	Uuid        string  `json:"uuid"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}
