package utils

import (
	"os"
	"path/filepath"
	"strings"
)

// PathUtils 路径工具函数集合
type PathUtils struct{}

// NewPathUtils 创建新的路径工具实例
func NewPathUtils() *PathUtils {
	return &PathUtils{}
}

// Join 连接路径片段
func (pu *PathUtils) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Clean 清理路径
func (pu *PathUtils) Clean(path string) string {
	return filepath.Clean(path)
}

// Abs 获取绝对路径
func (pu *PathUtils) Abs(path string) (string, error) {
	return filepath.Abs(path)
}

// Rel 获取相对路径
func (pu *PathUtils) Rel(basepath, targpath string) (string, error) {
	return filepath.Rel(basepath, targpath)
}

// Split 分割路径
func (pu *PathUtils) Split(path string) (dir, file string) {
	return filepath.Split(path)
}

// Dir 获取目录部分
func (pu *PathUtils) Dir(path string) string {
	return filepath.Dir(path)
}

// Base 获取文件名部分
func (pu *PathUtils) Base(path string) string {
	return filepath.Base(path)
}

// Ext 获取文件扩展名
func (pu *PathUtils) Ext(path string) string {
	return filepath.Ext(path)
}

// IsAbs 检查是否为绝对路径
func (pu *PathUtils) IsAbs(path string) bool {
	return filepath.IsAbs(path)
}

// VolumeName 获取卷名（Windows）
func (pu *PathUtils) VolumeName(path string) string {
	return filepath.VolumeName(path)
}

// FromSlash 将正斜杠转换为系统分隔符
func (pu *PathUtils) FromSlash(path string) string {
	return filepath.FromSlash(path)
}

// ToSlash 将系统分隔符转换为正斜杠
func (pu *PathUtils) ToSlash(path string) string {
	return filepath.ToSlash(path)
}

// Match 匹配路径模式
func (pu *PathUtils) Match(pattern, name string) (matched bool, err error) {
	return filepath.Match(pattern, name)
}

// Glob 查找匹配的文件
func (pu *PathUtils) Glob(pattern string) (matches []string, err error) {
	return filepath.Glob(pattern)
}

// Walk 遍历目录
func (pu *PathUtils) Walk(root string, fn filepath.WalkFunc) error {
	return filepath.Walk(root, fn)
}

// WalkDir 遍历目录（Go 1.16+）
func (pu *PathUtils) WalkDir(root string, fn func(string, os.DirEntry, error) error) error {
	return filepath.WalkDir(root, fn)
}

// CreateDirectory 创建目录
func (pu *PathUtils) CreateDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

// RemoveDirectory 删除目录
func (pu *PathUtils) RemoveDirectory(path string) error {
	return os.RemoveAll(path)
}

// DirectoryExists 检查目录是否存在
func (pu *PathUtils) DirectoryExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// FileExists 检查文件是否存在
func (pu *PathUtils) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDirectory 检查是否为目录
func (pu *PathUtils) IsDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsFile 检查是否为文件
func (pu *PathUtils) IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// GetFileSize 获取文件大小
func (pu *PathUtils) GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// GetFileMode 获取文件模式
func (pu *PathUtils) GetFileMode(path string) (os.FileMode, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Mode(), nil
}

// GetFileModTime 获取文件修改时间
func (pu *PathUtils) GetFileModTime(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.ModTime().Unix(), nil
}

// NormalizePath 标准化路径
func (pu *PathUtils) NormalizePath(path string) string {
	// 转换为系统分隔符
	path = filepath.FromSlash(path)

	// 清理路径
	path = filepath.Clean(path)

	// 处理相对路径
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err == nil {
			path = absPath
		}
	}

	return path
}

// GetRelativePath 获取相对路径
func (pu *PathUtils) GetRelativePath(basePath, targetPath string) (string, error) {
	baseAbs, err := filepath.Abs(basePath)
	if err != nil {
		return "", err
	}

	targetAbs, err := filepath.Abs(targetPath)
	if err != nil {
		return "", err
	}

	return filepath.Rel(baseAbs, targetAbs)
}

// EnsureDirectory 确保目录存在
func (pu *PathUtils) EnsureDirectory(path string) error {
	if pu.DirectoryExists(path) {
		return nil
	}
	return pu.CreateDirectory(path)
}

// GetParentDirectory 获取父目录
func (pu *PathUtils) GetParentDirectory(path string) string {
	return filepath.Dir(path)
}

// GetFileNameWithoutExt 获取不带扩展名的文件名
func (pu *PathUtils) GetFileNameWithoutExt(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

// IsSubPath 检查是否为子路径
func (pu *PathUtils) IsSubPath(parent, child string) bool {
	parentAbs, err := filepath.Abs(parent)
	if err != nil {
		return false
	}

	childAbs, err := filepath.Abs(child)
	if err != nil {
		return false
	}

	rel, err := filepath.Rel(parentAbs, childAbs)
	if err != nil {
		return false
	}

	return !strings.HasPrefix(rel, "..") && rel != "."
}
