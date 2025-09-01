package demo

import (
	"time"
)

// User 的方法实现

// GetID 获取用户ID
func (u *User) GetID() int64 {
	return u.ID
}

// GetName 获取用户名
func (u *User) GetName() string {
	return u.Name
}

// SetName 设置用户名
func (u *User) SetName(name string) {
	u.Name = name
}

// IsUserActive 检查用户是否激活
func (u *User) IsUserActive() bool {
	return u.IsActive
}

// Activate 激活用户
func (u *User) Activate() {
	u.IsActive = true
}

// Deactivate 停用用户
func (u *User) Deactivate() {
	u.IsActive = false
}

// UpdateLastLogin 更新最后登录时间
func (u *User) UpdateLastLogin() {
	// 模拟更新
}

// GetAge 获取年龄
func (u *User) GetAge() int {
	return u.Age
}

// SetAge 设置年龄
func (u *User) SetAge(age int) error {
	if age < 0 || age > 150 {
		return ErrInvalidAge
	}
	u.Age = age
	return nil
}

// Validate 验证用户数据
func (u *User) Validate() error {
	return ValidateUser(u)
}

// Clone 克隆用户
func (u *User) Clone() *User {
	return &User{
		ID:       u.ID,
		Name:     u.Name,
		Email:    u.Email,
		Age:      u.Age,
		IsActive: u.IsActive,
		Created:  u.Created,
	}
}

// Config 的方法实现

// GetDatabaseConfig 获取数据库配置
func (c *Config) GetDatabaseConfig() DatabaseConfig {
	return c.Database
}

// SetDatabaseConfig 设置数据库配置
func (c *Config) SetDatabaseConfig(dbConfig DatabaseConfig) {
	c.Database = dbConfig
}

// GetServerConfig 获取服务器配置
func (c *Config) GetServerConfig() ServerConfig {
	return c.Server
}

// IsCacheEnabled 检查缓存是否启用
func (c *Config) IsCacheEnabled() bool {
	return c.Cache.Enabled
}

// SetCacheEnabled 设置缓存启用状态
func (c *Config) SetCacheEnabled(enabled bool) {
	c.Cache.Enabled = enabled
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Database.Host == "" {
		return NewError("database host is required")
	}
	if c.Server.Port <= 0 {
		return NewError("server port must be positive")
	}
	return nil
}

// Event 的方法实现

// AddTag 添加标签
func (e *Event) AddTag(tag string) {
	e.Tags = append(e.Tags, tag)
}

// RemoveTag 移除标签
func (e *Event) RemoveTag(tag string) {
	for i, t := range e.Tags {
		if t == tag {
			e.Tags = append(e.Tags[:i], e.Tags[i+1:]...)
			break
		}
	}
}

// SetMetadata 设置元数据
func (e *Event) SetMetadata(key, value string) {
	e.Metadata[key] = value
}

// GetMetadata 获取元数据
func (e *Event) GetMetadata(key string) (string, bool) {
	value, exists := e.Metadata[key]
	return value, exists
}

// AddSubscriber 添加订阅者
func (e *Event) AddSubscriber(subscriber string) {
	e.Subscribers = append(e.Subscribers, subscriber)
}

// IsExpired 检查事件是否过期
func (e *Event) IsExpired(ttl time.Duration) bool {
	return time.Since(e.Timestamp) > ttl
}

// GetData 获取事件数据
func (e *Event) GetData(key string) (any, bool) {
	value, exists := e.Data[key]
	return value, exists
}

// SetData 设置事件数据
func (e *Event) SetData(key string, value any) {
	e.Data[key] = value
}
