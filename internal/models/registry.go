package models

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// Registry maps DB table name -> model schema/type.
// Registry：tableName -> model 类型/信息
type Registry struct {
	mu      sync.RWMutex
	byTable map[string]ModelInfo
}

// ModelInfo describes a model type and optional default preloads.
// ModelInfo：模型类型 + 默认 Preload
type ModelInfo struct {
	// StructType must be a non-pointer struct type, e.g. reflect.TypeOf(User{})
	StructType reflect.Type

	// Table is resolved by GORM schema (honors TableName() and naming strategy)
	Table string

	// DefaultPreloads are applied on every FindByTablePK call.
	DefaultPreloads []string
}

func NewRegistry() *Registry {
	return &Registry{byTable: make(map[string]ModelInfo)}
}

// Register parses GORM schema to resolve the table name and registers model type.
// Register：注册模型（解析 TableName()/命名策略得到 table 名）
func (r *Registry) Register(db *gorm.DB, model any, opts ...ModelOpt) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if model == nil {
		return fmt.Errorf("model is nil")
	}

	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return fmt.Errorf("model must be struct or *struct, got %s", t.Kind())
	}

	// Parse schema to get resolved table name (TableName() + naming strategy)
	var cache sync.Map
	sch, err := schema.Parse(reflect.New(t).Interface(), &cache, db.NamingStrategy)
	if err != nil {
		return fmt.Errorf("schema.Parse failed: %w", err)
	}

	info := ModelInfo{
		StructType: t,
		Table:      sch.Table,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(&info)
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.byTable[info.Table] = info
	return nil
}

// FindByTablePK only needs table + pkValue.
// It will query using db.Model(&T{}).First(&T{}, pkValue) to keep soft delete/default scopes/hooks.
// FindByTablePK：只传 table + pkValue；内部用 Model(&T{}) + First(pk) 方式查询
func (r *Registry) FindByTablePK(ctx context.Context, db *gorm.DB, table string, pkValue any) (any, error) {
	return r.FindByTablePKWith(ctx, db, table, pkValue /* no opts */)
}

// FindByTablePKWith is the same as FindByTablePK, but allows optional preloads/scopes.
// FindByTablePKWith：在只传 table+pkValue 的基础上，允许可选 Preload/Scope
func (r *Registry) FindByTablePKWith(ctx context.Context, db *gorm.DB, table string, pkValue any, qopts ...QueryOpt) (any, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if table == "" {
		return nil, fmt.Errorf("table is empty")
	}

	r.mu.RLock()
	info, ok := r.byTable[table]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("table %q not registered", table)
	}

	// Create new instance *T
	dstPtr := reflect.New(info.StructType).Interface()

	// Build options
	o := &queryOptions{}
	for _, opt := range qopts {
		if opt != nil {
			opt(o)
		}
	}

	// IMPORTANT:
	// Use Model(dstPtr) (NOT Table(table)) to keep:
	// - soft delete
	// - default scopes / callbacks
	// - hooks
	// - association preloads behavior
	//
	// 关键点：不用 Table(table)，用 Model(&T{}) 来确保软删除/默认scope/hook都生效
	q := db.WithContext(ctx).Model(dstPtr)

	// Apply model default preloads
	for _, p := range info.DefaultPreloads {
		if p != "" {
			q = q.Preload(p)
		}
	}

	// Apply extra preloads
	for _, p := range o.extraPreloads {
		if p != "" {
			q = q.Preload(p)
		}
	}

	// Apply extra scopes
	for _, s := range o.scopes {
		if s != nil {
			q = q.Scopes(s)
		}
	}

	// First by primary key value (works for int/uint/string/uuid, etc.)
	// 按主键值查询（支持 int/uint/string/uuid 等；复合主键不适用）
	if err := q.First(dstPtr, pkValue).Error; err != nil {
		return nil, err
	}
	return dstPtr, nil
}

// FindByTablePKValue returns a non-pointer struct value (T) instead of *T.
// FindByTablePKValue：返回 struct 值（非指针）
func (r *Registry) FindByTablePKValue(ctx context.Context, db *gorm.DB, table string, pkValue any) (any, error) {
	ptr, err := r.FindByTablePK(ctx, db, table, pkValue)
	if err != nil {
		return nil, err
	}
	v := reflect.ValueOf(ptr)
	if v.Kind() == reflect.Pointer {
		return v.Elem().Interface(), nil
	}
	return ptr, nil
}

// QueryOpt allows customizing query (e.g., extra preloads).
// QueryOpt：可选的查询配置（例如额外 Preload）
type QueryOpt func(*queryOptions)

type queryOptions struct {
	extraPreloads []string
	scopes        []func(*gorm.DB) *gorm.DB
}

// WithPreloads adds extra preloads for this query.
// WithPreloads：为本次查询额外指定 Preload（不影响注册时默认 Preload）
func WithPreloads(paths ...string) QueryOpt {
	return func(o *queryOptions) {
		o.extraPreloads = append(o.extraPreloads, paths...)
	}
}

// WithScopes applies extra gorm scopes for this query.
// WithScopes：为本次查询额外套用 scopes
func WithScopes(scopes ...func(*gorm.DB) *gorm.DB) QueryOpt {
	return func(o *queryOptions) {
		o.scopes = append(o.scopes, scopes...)
	}
}

// ModelOpt allows configuring model info at register time.
// ModelOpt：注册模型时的配置项
type ModelOpt func(*ModelInfo)

// WithDefaultPreloads sets default preloads for this model.
// WithDefaultPreloads：给该模型设置“默认 Preload 列表”（每次 Find 都会自动 Preload）
func WithDefaultPreloads(paths ...string) ModelOpt {
	return func(mi *ModelInfo) {
		mi.DefaultPreloads = append(mi.DefaultPreloads, paths...)
	}
}
