// Package store 负责 sing-box 配置文件的加载、校验、持久化。
// 校验管线完全复用 sing-box 自身的解码器 + 实例化器：
//   parse(严格解码+checkOptions) → box.New 干跑 → 原子写 → (runner reload)
package store

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	sjson "github.com/sagernet/sing/common/json"
)

// Meta 是配置的旁车元数据（sing-box 配置本身没有的字段，如规则 id）。
// 存于 <config>.meta，与配置同目录，保证规则 CRUD 有稳定 id。
type Meta struct {
	RuleIDs               []string    `json:"rules,omitempty"`
	DNSRuleIDs            []string    `json:"dns_rules,omitempty"`
	RuleSetIDs            []string    `json:"rule_sets,omitempty"`
	CertificateProviderIDs []string   `json:"certificate_providers,omitempty"`
	Users                 []UserMeta  `json:"users,omitempty"`
}

// UserMeta 全局用户池：一个用户可绑定多个入站实例（多对多）
// 绑定时按入站类型把用户注入对应 inbounds 的 users[]
type UserMeta struct {
	Name          string   `json:"name"`
	UUID          string   `json:"uuid,omitempty"`
	Password      string   `json:"password,omitempty"`
	Flow          string   `json:"flow,omitempty"`
	BoundInbounds []string `json:"bound_inbounds,omitempty"`
}

type Store struct {
	mu       sync.Mutex
	path     string
	metaPath string
	Options  option.Options
	Meta     Meta
}

func New(path string) *Store {
	return &Store{path: path, metaPath: path + ".meta"}
}

func (s *Store) Path() string { return s.path }

// Parse 严格解码（未知字段报错、重复 tag 检查），不做实例化。
// 内部强制注入 include.Context（含全部 registry），保证任何调用入口
// （启动加载、API 请求）都能解码 inbounds/outbounds/endpoints/services 等多态类型。
func Parse(ctx context.Context, content []byte) (option.Options, error) {
	ctx = include.Context(ctx)
	return sjson.UnmarshalExtendedContext[option.Options](ctx, content)
}

// Validate 完整校验：解码 + box.New 干跑（复用 daemon.CheckConfig 模式）。
func Validate(ctx context.Context, content []byte) error {
	options, err := Parse(ctx, content)
	if err != nil {
		return err
	}
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()
	instance, err := box.New(box.Options{Context: cctx, Options: options})
	if err != nil {
		return err
	}
	instance.Close()
	return nil
}

func defaultConfig(inboundType string, listen string, listenPort uint16) option.Options {
	content := []byte(`{
  "log": { "level": "info", "timestamp": true },
  "inbounds": [
    { "type": "` + inboundType + `", "tag": "in-main", "listen": "` + listen + `", "listen_port": ` + itoa(listenPort) + ` }
  ],
  "outbounds": [
    { "type": "direct", "tag": "direct" },
    { "type": "block", "tag": "block" }
  ],
  "route": { "final": "direct" }
}`)
	options, err := Parse(include.Context(context.Background()), content)
	if err != nil {
		panic(fmt.Sprintf("bad default config: %v", err))
	}
	return options
}

func itoa(value uint16) string { return strconv.FormatUint(uint64(value), 10) }

// Load 读取主配置文件；不存在时按默认值生成骨架（由调用方传入 settings 默认值）。
func (s *Store) Load(ctx context.Context, defaults DefaultConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	content, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		s.Options = defaultConfig(defaults.InboundType, defaults.Listen, defaults.ListenPort)
		if err := s.saveLocked(ctx); err != nil {
			return err
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	options, err := Parse(ctx, content)
	if err != nil {
		return fmt.Errorf("decode config: %w", err)
	}
	s.Options = options
	dnsRuleCount := 0
	if options.DNS != nil {
		dnsRuleCount = len(options.DNS.Rules)
	}
	ruleSetCount := 0
	routeRuleCount := 0
	if options.Route != nil {
		ruleSetCount = len(options.Route.RuleSet)
		routeRuleCount = len(options.Route.Rules)
	}
	s.Meta = loadMeta(s.metaPath, routeRuleCount, dnsRuleCount, ruleSetCount, len(options.CertificateProviders))
	return nil
}

// DefaultConfig 新建配置骨架时使用的默认值。
type DefaultConfig struct {
	InboundType string
	Listen      string
	ListenPort  uint16
}

// SetPath 切换主配置文件路径（settings.config 变更时使用）。
func (s *Store) SetPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.path = path
	s.metaPath = path + ".meta"
}

// Content 返回当前配置的格式化 JSON。
func (s *Store) Content(ctx context.Context) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return marshalOptions(ctx, s.Options)
}

func marshalOptions(ctx context.Context, options option.Options) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := sjson.NewEncoderContext(ctx, &buffer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(options); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// Save 将当前内存中的 Options 原子写盘（temp + rename），保留 .bak 备份。
func (s *Store) Save(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked(ctx)
}

// RawContent 返回主配置文件的原始内容（未解析，保留注释/格式）；文件不存在返回 nil。
// Users 返回用户池副本
func (s *Store) Users() []UserMeta {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]UserMeta(nil), s.Meta.Users...)
}

// UpdateUsers 原子更新用户池并落盘 meta
func (s *Store) UpdateUsers(mutate func([]UserMeta) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := mutate(s.Meta.Users); err != nil {
		return err
	}
	content, err := json.Marshal(s.Meta)
	if err != nil {
		return err
	}
	return os.WriteFile(s.metaPath, content, 0o644)
}

func (s *Store) RawContent() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	content, err := os.ReadFile(s.path)
	if err != nil {
		return nil
	}
	return content
}

// RawSave 原样保存配置文本（保留注释/格式）：解析+干跑校验通过后原子写盘原始内容，
// 并同步内存 Options（后续 CRUD 基于新配置）。失败不落盘、内存回滚。
func (s *Store) RawSave(ctx context.Context, content []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshotOptions, snapshotMeta := s.snapshot(ctx)
	options, err := Parse(ctx, content)
	if err != nil {
		return err
	}
	if err := Validate(ctx, content); err != nil {
		s.restore(snapshotOptions, snapshotMeta)
		return err
	}
	s.Options = options
	return s.saveRawLocked(ctx, content)
}

func (s *Store) saveRawLocked(ctx context.Context, content []byte) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// 备份
	if _, err := os.Stat(s.path); err == nil {
		_ = os.WriteFile(s.path+".bak", content, 0o644)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return err
	}
	// 元数据
	metaContent, err := json.Marshal(s.Meta)
	if err != nil {
		return err
	}
	return os.WriteFile(s.metaPath, metaContent, 0o644)
}

func (s *Store) saveLocked(ctx context.Context) error {
	content, err := marshalOptions(ctx, s.Options)
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// 备份
	if _, err := os.Stat(s.path); err == nil {
		_ = os.WriteFile(s.path+".bak", content, 0o644)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return err
	}
	// 元数据
	metaContent, err := json.Marshal(s.Meta)
	if err != nil {
		return err
	}
	return os.WriteFile(s.metaPath, metaContent, 0o644)
}

func loadMeta(path string, ruleCount, dnsRuleCount, ruleSetCount, certProviderCount int) Meta {
	var meta Meta
	content, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(content, &meta)
	}
	// 数量不匹配（外部手动改过配置）→ 重新生成 id
	if len(meta.RuleIDs) != ruleCount {
		meta.RuleIDs = make([]string, ruleCount)
		for i := range meta.RuleIDs {
			meta.RuleIDs[i] = NewUUID()
		}
	}
	if len(meta.DNSRuleIDs) != dnsRuleCount {
		meta.DNSRuleIDs = make([]string, dnsRuleCount)
		for i := range meta.DNSRuleIDs {
			meta.DNSRuleIDs[i] = NewUUID()
		}
	}
	if len(meta.RuleSetIDs) != ruleSetCount {
		meta.RuleSetIDs = make([]string, ruleSetCount)
		for i := range meta.RuleSetIDs {
			meta.RuleSetIDs[i] = NewUUID()
		}
	}
	if len(meta.CertificateProviderIDs) != certProviderCount {
		meta.CertificateProviderIDs = make([]string, certProviderCount)
		for i := range meta.CertificateProviderIDs {
			meta.CertificateProviderIDs[i] = NewUUID()
		}
	}
	return meta
}

// AlignMeta 确保 meta id 数组与配置段数量对齐（外部编辑后自愈）。
func (s *Store) AlignMeta() {
	s.mu.Lock()
	defer s.mu.Unlock()
	dnsRuleCount := 0
	if s.Options.DNS != nil {
		dnsRuleCount = len(s.Options.DNS.Rules)
	}
	ruleSetCount := 0
	if s.Options.Route != nil {
		ruleSetCount = len(s.Options.Route.RuleSet)
	}
	s.Meta = loadMeta(s.metaPath, len(s.Options.Route.Rules), dnsRuleCount, ruleSetCount, len(s.Options.CertificateProviders))
}

// Update 加锁执行 mutate，然后全量校验（解码 + box.New 干跑）并原子写盘。
// 校验失败不落盘，内存状态回滚（深拷贝快照），返回错误。
func (s *Store) Update(ctx context.Context, mutate func(*option.Options, *Meta) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshotOptions, snapshotMeta := s.snapshot(ctx)
	if err := mutate(&s.Options, &s.Meta); err != nil {
		return err
	}
	content, err := marshalOptions(ctx, s.Options)
	if err != nil {
		s.restore(snapshotOptions, snapshotMeta)
		return err
	}
	if err := Validate(ctx, content); err != nil {
		s.restore(snapshotOptions, snapshotMeta)
		return err
	}
	return s.saveLocked(ctx)
}

// snapshot 通过 JSON 往返深拷贝当前状态。
func (s *Store) snapshot(ctx context.Context) (option.Options, *Meta) {
	content, err := marshalOptions(ctx, s.Options)
	if err != nil {
		return s.Options, &s.Meta
	}
	options, err := Parse(ctx, content)
	if err != nil {
		return s.Options, &s.Meta
	}
	ruleIDs := make([]string, len(s.Meta.RuleIDs))
	copy(ruleIDs, s.Meta.RuleIDs)
	return options, &Meta{RuleIDs: ruleIDs}
}

func (s *Store) restore(options option.Options, meta *Meta) {
	s.Options = options
	s.Meta = *meta
}

// NewUUID 生成 v4 风格 uuid（无横杠校验，仅作稳定 id 用）。
func NewUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
