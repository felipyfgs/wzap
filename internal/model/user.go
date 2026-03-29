package model

import "time"

type User struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Token      string    `json:"token"`
	Webhook    string    `json:"webhook"`
	Events     string    `json:"events"`
	Expiration int       `json:"expiration"`
	ProxyUrl   string    `json:"proxyUrl"`
	History    int       `json:"history"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type UserCreateReq struct {
	Name  string `json:"name" validate:"required"`
	Token string `json:"token"`
}
