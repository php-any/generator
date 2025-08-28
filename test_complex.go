package main

import (
	"net/http"
	"time"
)

// User 用户结构体
type User struct {
	ID       int                    `json:"id"`
	Name     string                 `json:"name"`
	Email    string                 `json:"email"`
	Created  time.Time              `json:"created"`
	Profile  *Profile               `json:"profile"`
	Settings map[string]interface{} `json:"settings"`
}

// Profile 用户档案
type Profile struct {
	Bio      string            `json:"bio"`
	Avatar   string            `json:"avatar"`
	Tags     []string          `json:"tags"`
	Metadata map[string]string `json:"metadata"`
}

// UserService 用户服务
type UserService struct {
	Client  *http.Client   `json:"client"`
	BaseURL string         `json:"base_url"`
	Config  *ServiceConfig `json:"config"`
}

// ServiceConfig 服务配置
type ServiceConfig struct {
	Timeout    time.Duration     `json:"timeout"`
	RetryCount int               `json:"retry_count"`
	Headers    map[string]string `json:"headers"`
}

// GetUser 获取用户
func (s *UserService) GetUser(id int) (*User, error) {
	return &User{ID: id, Name: "Test User"}, nil
}

// UpdateUser 更新用户
func (s *UserService) UpdateUser(user *User) error {
	return nil
}

// GetUsers 获取用户列表
func (s *UserService) GetUsers(limit int, offset int) ([]*User, error) {
	return []*User{{ID: 1, Name: "User 1"}, {ID: 2, Name: "User 2"}}, nil
}
