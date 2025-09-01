package demo

import (
	"context"
	"time"
)

// NewUser 构造函数 - 测试构造函数生成
func NewUser(name, email string, age int) *User {
	return &User{
		Name:     name,
		Email:    email,
		Age:      age,
		IsActive: true,
		Created:  time.Now(),
	}
}

// NewConfig 构造函数 - 测试复杂构造函数
func NewConfig(dbHost string, dbPort int, serverHost string, serverPort int) *Config {
	return &Config{
		Database: DatabaseConfig{
			Host: dbHost,
			Port: dbPort,
		},
		Server: ServerConfig{
			Host: serverHost,
			Port: serverPort,
		},
		Cache: CacheConfig{
			Enabled: true,
			TTL:     time.Hour,
		},
	}
}

// ValidateUser 验证函数 - 测试基本函数
func ValidateUser(user *User) error {
	if user.Name == "" {
		return ErrInvalidName
	}
	if user.Email == "" {
		return ErrInvalidEmail
	}
	if user.Age < 0 || user.Age > 150 {
		return ErrInvalidAge
	}
	return nil
}

// ProcessUsers 处理函数 - 测试多参数函数
func ProcessUsers(users []*User, filter func(*User) bool, processor func(*User) error) error {
	for _, user := range users {
		if filter != nil && !filter(user) {
			continue
		}
		if err := processor(user); err != nil {
			return err
		}
	}
	return nil
}

// GetUserByID 查询函数 - 测试带上下文的函数
func GetUserByID(ctx context.Context, id int64) (*User, error) {
	// 模拟数据库查询
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(100 * time.Millisecond):
		return &User{
			ID:       id,
			Name:     "Demo User",
			Email:    "demo@example.com",
			Age:      25,
			IsActive: true,
			Created:  time.Now(),
		}, nil
	}
}

// CreateEvent 创建事件函数 - 测试复杂参数函数
func CreateEvent(eventType string, data map[string]any, tags []string) *Event {
	return &Event{
		ID:        generateID(),
		Type:      eventType,
		Data:      data,
		Tags:      tags,
		Metadata:  make(map[string]string),
		Timestamp: time.Now(),
	}
}

// generateID 内部函数 - 测试私有函数（不应该被生成）
func generateID() string {
	return "event_" + time.Now().Format("20060102150405")
}

// 错误定义
var (
	ErrInvalidName  = NewError("invalid name")
	ErrInvalidEmail = NewError("invalid email")
	ErrInvalidAge   = NewError("invalid age")
)

// NewError 错误构造函数
func NewError(message string) error {
	return &DemoError{Message: message}
}

// DemoError 自定义错误类型
type DemoError struct {
	Message string
}

func (e *DemoError) Error() string {
	return e.Message
}
