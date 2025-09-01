package demo

import (
	"context"
	"time"
)

// User 用户结构体 - 测试基本结构体生成
type User struct {
	ID       int64     `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Age      int       `json:"age"`
	IsActive bool      `json:"is_active"`
	Created  time.Time `json:"created"`
	Intface  UserService
}

// UserService 用户服务接口 - 测试接口生成
type UserService interface {
	// 基本方法
	GetUser(id int64) (*User, error)
	CreateUser(user *User) error
	UpdateUser(user *User) error
	DeleteUser(id int64) error

	// 带多个参数的方法
	SearchUsers(name string, age int, active bool) ([]*User, error)

	// 带上下文的方法
	GetUserWithContext(ctx context.Context, id int64) (*User, error)

	// 无返回值的方法
	LogUser(user *User)
}

// Config 配置结构体 - 测试嵌套结构体
type Config struct {
	Database DatabaseConfig `json:"database"`
	Server   ServerConfig   `json:"server"`
	Cache    CacheConfig    `json:"cache"`
}

type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
}

type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type CacheConfig struct {
	Enabled bool          `json:"enabled"`
	TTL     time.Duration `json:"ttl"`
}

// Options 选项结构体 - 测试指针字段
type Options struct {
	Timeout *time.Duration `json:"timeout,omitempty"`
	Retries *int           `json:"retries,omitempty"`
	Debug   *bool          `json:"debug,omitempty"`
}

// Result 结果结构体 - 测试泛型相关
type Result[T any] struct {
	Data    T      `json:"data"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Event 事件结构体 - 测试切片和映射
type Event struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Data        map[string]any    `json:"data"`
	Tags        []string          `json:"tags"`
	Metadata    map[string]string `json:"metadata"`
	Timestamp   time.Time         `json:"timestamp"`
	Subscribers []string          `json:"subscribers"`
}

// Node 测试循环引用的结构体
type Node struct {
	ID       int
	Value    string
	Parent   *Node
	Children []*Node
}

// privateStruct 小写开头的私有结构体（不应该生成）
type privateStruct struct {
	value string
}

// PrivateInterface 大写开头的接口（应该生成）
type PrivateInterface interface {
	GetValue() string
}
