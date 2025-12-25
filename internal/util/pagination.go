package util

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
)

// PageQuery is a common pagination query.
// PageQuery 是通用分页查询参数。
type PageQuery struct {
	Page     int
	PageSize int
}

// ParsePageQuery parses ?page=&page_size= with sane defaults.
// ParsePageQuery 解析分页参数并提供合理默认值。
func ParsePageQuery(c fiber.Ctx) PageQuery {
	page := atoiDefault(c.Query("page"), 1)
	size := atoiDefault(c.Query("page_size"), 20)

	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return PageQuery{Page: page, PageSize: size}
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

// OffsetLimit calculates SQL offset/limit.
// OffsetLimit 计算 SQL 的 offset/limit。
func (p PageQuery) OffsetLimit() (offset int, limit int) {
	return (p.Page - 1) * p.PageSize, p.PageSize
}
