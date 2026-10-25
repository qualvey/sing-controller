package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/qualvey/sing-controller/internal/logger"
	"github.com/qualvey/sing-controller/internal/store"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	sjson "github.com/sagernet/sing/common/json"
)

// SupervisorInterface 定义与 Supervisor 的交互契约
type SupervisorInterface interface {
	Restart(ctx context.Context) error
}

// Manager 节点源与配置组管理器
type Manager struct {
	mu           sync.RWMutex
	dir          string // 存储根目录，如 providers/
	providers    map[string]*Provider
	profiles     map[string]*Profile
	injectedTags []string // 记录已动态注入到主配置的节点 tag 列表
	fetcher      *Fetcher
	supervisor   SupervisorInterface
	store        *store.Store
	activeProfID string
}

func NewManager(dir string, st *store.Store, sup SupervisorInterface) (*Manager, error) {
	if dir == "" {
		dir = "providers"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建 provider 目录失败: %w", err)
	}

	m := &Manager{
		dir:          dir,
		providers:    make(map[string]*Provider),
		profiles:     make(map[string]*Profile),
		injectedTags: make([]string, 0),
		fetcher:      NewFetcher(30 * time.Second),
		supervisor:   sup,
		store:        st,
	}

	m.loadProviders()
	m.loadInjectedTags()
	m.loadProfiles()
	m.cleanBaseConfig()

	go m.startCronLoop()
	return m, nil
}

// cleanBaseConfig 清洗主配置，将此前历史遗留写入主配置的外部节点剥离，恢复 Base 纯净
func (m *Manager) cleanBaseConfig() {
	if m.store == nil {
		return
	}
	m.store.RLock()
	hasInjected := false
	tagsToPurge := make(map[string]bool)
	for _, t := range m.injectedTags {
		tagsToPurge[t] = true
	}
	for _, ob := range m.store.Options.Outbounds {
		if tagsToPurge[ob.Tag] {
			hasInjected = true
			break
		}
	}
	m.store.RUnlock()

	if !hasInjected {
		return
	}

	logger.Info("[Provider] 检测到主配置中残留有历史外部订阅节点，正在自动脱水净化...")
	_ = m.store.Update(context.Background(), func(opts *option.Options, meta *store.Meta) error {
		var cleanOutbounds []option.Outbound
		for _, ob := range opts.Outbounds {
			if !tagsToPurge[ob.Tag] {
				cleanOutbounds = append(cleanOutbounds, ob)
			}
		}
		opts.Outbounds = cleanOutbounds
		return nil
	})
	logger.Info("[Provider] 主配置脱水净化完成，已恢复 Clean Base 纯净态")
}

func (m *Manager) providersMetaPath() string {
	return filepath.Join(m.dir, "providers.json")
}

func (m *Manager) profilesMetaPath() string {
	return filepath.Join(m.dir, "profiles.json")
}

func (m *Manager) injectedTagsPath() string {
	return filepath.Join(m.dir, "injected_tags.json")
}

func (m *Manager) loadInjectedTags() {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.injectedTagsPath())
	if err == nil {
		_ = json.Unmarshal(data, &m.injectedTags)
	}

	// 自动扫描目录中现有的所有 fragment 节点，确保历史注入节点也被追踪
	tagSet := make(map[string]bool)
	for _, t := range m.injectedTags {
		tagSet[t] = true
	}
	for _, p := range m.providers {
		if obs, err := m.GetProviderOutbounds(p.Tag); err == nil {
			for _, ob := range obs {
				if !tagSet[ob.Tag] {
					m.injectedTags = append(m.injectedTags, ob.Tag)
					tagSet[ob.Tag] = true
				}
			}
		}
	}
}

func (m *Manager) saveInjectedTagsLocked() error {
	data, err := json.MarshalIndent(m.injectedTags, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.injectedTagsPath(), data, 0o644)
}

func (m *Manager) providerOutboundPath(tag string) string {
	return filepath.Join(m.dir, fmt.Sprintf("outbound-%s.json", tag))
}

func (m *Manager) providerFullPath(tag string) string {
	return filepath.Join(m.dir, fmt.Sprintf("full-%s.json", tag))
}

func (m *Manager) providerRuntimePath() string {
	return filepath.Join(filepath.Dir(m.store.Path()), "10-provider-config.json")
}

// withControllerAPIService 保留控制器所需的 API service。
// 完整订阅通常不包含该 service，而控制器的 gRPC/Clash 代理依赖它。
func (m *Manager) withControllerAPIService(fullConfig []byte) ([]byte, error) {
	var full map[string]json.RawMessage
	if err := json.Unmarshal(fullConfig, &full); err != nil {
		return nil, err
	}
	baseContent := m.store.RawContent()
	var base map[string]json.RawMessage
	if len(baseContent) > 0 {
		_ = json.Unmarshal(baseContent, &base)
	}

	var baseServices []json.RawMessage
	_ = json.Unmarshal(base["services"], &baseServices)
	var controllerService json.RawMessage
	for _, raw := range baseServices {
		var service struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(raw, &service) == nil && service.Type == "api" {
			controllerService = raw
			break
		}
	}
	if len(controllerService) == 0 {
		controllerService = json.RawMessage(`{"type":"api","listen":"127.0.0.1","listen_port":9900}`)
	}

	var fullServices []json.RawMessage
	_ = json.Unmarshal(full["services"], &fullServices)
	filtered := make([]json.RawMessage, 0, len(fullServices)+1)
	for _, raw := range fullServices {
		var service struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(raw, &service) == nil && service.Type == "api" {
			continue
		}
		filtered = append(filtered, raw)
	}
	filtered = append(filtered, controllerService)
	full["services"], _ = json.Marshal(filtered)
	return json.MarshalIndent(full, "", "  ")
}

func (m *Manager) loadProviders() {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.providersMetaPath())
	if err != nil {
		return
	}

	var list []*Provider
	if err := json.Unmarshal(data, &list); err == nil {
		for _, p := range list {
			if p.UseFullConfig {
				p.UseProviderRules = false
			}
			m.providers[p.ID] = p
		}
	}
}

func (m *Manager) saveProvidersLocked() error {
	list := make([]*Provider, 0, len(m.providers))
	for _, p := range m.providers {
		list = append(list, p)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.providersMetaPath(), data, 0o644)
}

func (m *Manager) loadProfiles() {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.profilesMetaPath())
	if err != nil {
		// 初始化默认 Profile
		defaultTags := make([]string, 0, len(m.providers))
		for _, p := range m.providers {
			defaultTags = append(defaultTags, p.Tag)
		}
		defaultProfile := &Profile{
			ID:              "default",
			Tag:             "default",
			Name:            "默认方案",
			Description:     "包含基础主配置与所有已启用订阅节点",
			ProviderTags:    defaultTags,
			TargetSelectors: []string{"Proxy", "Auto"},
			AutoReload:      true,
			IsActive:        true,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		m.profiles[defaultProfile.ID] = defaultProfile
		m.activeProfID = defaultProfile.ID
		_ = m.saveProfilesLocked()
		return
	}

	var list []*Profile
	if err := json.Unmarshal(data, &list); err == nil {
		for _, p := range list {
			if p.ID == "default" && len(p.ProviderTags) == 0 && len(m.providers) > 0 {
				for _, prov := range m.providers {
					p.ProviderTags = append(p.ProviderTags, prov.Tag)
				}
			}
			m.profiles[p.ID] = p
			if p.IsActive {
				m.activeProfID = p.ID
			}
		}
	}
}

func (m *Manager) saveProfilesLocked() error {
	list := make([]*Profile, 0, len(m.profiles))
	for _, p := range m.profiles {
		list = append(list, p)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.profilesMetaPath(), data, 0o644)
}

// ListProviders 获取所有节点源
func (m *Manager) ListProviders() []*Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*Provider, 0, len(m.providers))
	for _, p := range m.providers {
		cp := *p
		list = append(list, &cp)
	}
	return list
}

// GetProvider 获取指定节点源
func (m *Manager) GetProvider(id string) (*Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.providers[id]
	if !ok {
		return nil, errors.New("provider not found")
	}
	cp := *p
	return &cp, nil
}

// AddProvider 添加新订阅/节点源并初次解析
func (m *Manager) AddProvider(ctx context.Context, p Provider) (*Provider, error) {
	if p.Tag == "" {
		return nil, errors.New("tag 不能为空")
	}
	if p.Name == "" {
		p.Name = p.Tag
	}
	if p.ID == "" {
		p.ID = store.NewUUID()
	}
	if p.UseFullConfig {
		p.UseProviderRules = false
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now

	m.mu.Lock()
	for _, existing := range m.providers {
		if existing.Tag == p.Tag {
			m.mu.Unlock()
			return nil, fmt.Errorf("tag %s 已存在", p.Tag)
		}
	}
	m.providers[p.ID] = &p
	_ = m.saveProvidersLocked()

	// 自动将新订阅加入默认 Profile（如果存在）
	if defProf, ok := m.profiles["default"]; ok {
		hasTag := false
		for _, t := range defProf.ProviderTags {
			if t == p.Tag {
				hasTag = true
				break
			}
		}
		if !hasTag {
			defProf.ProviderTags = append(defProf.ProviderTags, p.Tag)
			defProf.UpdatedAt = now
			_ = m.saveProfilesLocked()
		}
	}
	m.mu.Unlock()

	// 若为远程源且启用，立即异步拉取一次
	if p.Type == ProviderTypeRemote && p.URL != "" && p.Enabled {
		go func() {
			_, _ = m.RefreshProvider(context.Background(), p.ID)
		}()
	}

	return &p, nil
}

// UpdateProvider 更新订阅源属性
func (m *Manager) UpdateProvider(ctx context.Context, p Provider) error {
	m.mu.Lock()
	existing, ok := m.providers[p.ID]
	if !ok {
		m.mu.Unlock()
		return errors.New("provider not found")
	}

	urlChanged := existing.URL != p.URL
	formatChanged := existing.Format != p.Format
	fullConfigChanged := existing.UseFullConfig != p.UseFullConfig
	rulesChanged := existing.UseProviderRules != p.UseProviderRules
	enabledChanged := existing.Enabled != p.Enabled
	providerType := existing.Type

	existing.Name = p.Name
	existing.URL = p.URL
	existing.Format = p.Format
	existing.UseFullConfig = p.UseFullConfig
	existing.UseProviderRules = p.UseProviderRules && !p.UseFullConfig
	existing.UpdateIntervalMin = p.UpdateIntervalMin
	existing.Enabled = p.Enabled
	existing.UpdatedAt = time.Now()
	_ = m.saveProvidersLocked()
	m.mu.Unlock()

	// 如果 URL、格式或启用状态发生变化且处于启用状态，立即自动重新拉取
	if (urlChanged || formatChanged || fullConfigChanged || rulesChanged || enabledChanged) && p.Enabled && providerType == ProviderTypeRemote && p.URL != "" {
		if _, err := m.RefreshProvider(ctx, p.ID); err != nil {
			return err
		}
	}

	return nil
}

// DeleteProvider 删除订阅源及其独立片段文件，并从所有方案中注销、重新应用活动方案
func (m *Manager) DeleteProvider(id string) error {
	m.mu.Lock()
	p, ok := m.providers[id]
	if !ok {
		m.mu.Unlock()
		return errors.New("provider not found")
	}

	tag := p.Tag
	delete(m.providers, id)
	_ = m.saveProvidersLocked()

	// 从所有配置方案中剔除该订阅源
	for _, prof := range m.profiles {
		newTags := make([]string, 0, len(prof.ProviderTags))
		for _, pt := range prof.ProviderTags {
			if pt != tag {
				newTags = append(newTags, pt)
			}
		}
		prof.ProviderTags = newTags
		prof.UpdatedAt = time.Now()
	}
	_ = m.saveProfilesLocked()
	activeID := m.activeProfID
	m.mu.Unlock()

	// 读取待删除片段中的节点 tags，确保加入历史清除列表
	if obs, err := m.GetProviderOutbounds(tag); err == nil {
		m.mu.Lock()
		for _, ob := range obs {
			m.injectedTags = append(m.injectedTags, ob.Tag)
		}
		_ = m.saveInjectedTagsLocked()
		m.mu.Unlock()
	}

	// 删除磁盘上的独立片段
	_ = os.Remove(m.providerOutboundPath(tag))
	_ = os.Remove(m.providerFullPath(tag))
	_ = os.Remove(m.providerRuntimePath())

	// 重新应用活动方案，彻底将已删除订阅的节点从内核和主配置中清除
	if activeID != "" {
		if err := m.ApplyProfile(context.Background(), activeID); err != nil {
			logger.Warn("[Provider] 删除订阅 %s 后重新应用方案失败: %v", tag, err)
		}
	}
	return nil
}

// RefreshProvider 拉取/刷新指定节点源，并将解析后的节点落盘为 outbound-<tag>.json
func (m *Manager) RefreshProvider(ctx context.Context, id string) ([]option.Outbound, error) {
	m.mu.RLock()
	p, ok := m.providers[id]
	if !ok {
		m.mu.RUnlock()
		return nil, errors.New("provider not found")
	}
	urlStr := p.URL
	etag := p.LastETag
	format := p.Format
	tag := p.Tag
	useFullConfig := p.UseFullConfig
	useProviderRules := p.UseProviderRules
	m.mu.RUnlock()

	if urlStr == "" {
		return nil, errors.New("远程订阅 URL 为空")
	}

	body, newETag, err := m.fetcher.Fetch(ctx, urlStr, etag)
	if err != nil {
		if errors.Is(err, ErrNotModified) {
			logger.Info("[Provider] 订阅 %s 未修改 (304), 使用缓存节点", tag)
			return m.GetProviderOutbounds(tag)
		}
		m.updateProviderError(id, err.Error())
		return nil, err
	}

	if useFullConfig && format != FormatSingBox && format != FormatAuto {
		err := fmt.Errorf("完整配置模式仅支持 sing-box JSON 格式")
		m.updateProviderError(id, err.Error())
		return nil, err
	}

	var fullConfig []byte
	var fullOptions option.Options
	var hasFullConfig bool
	if useFullConfig {
		parsed, err := store.Parse(ctx, body)
		if err != nil {
			m.updateProviderError(id, fmt.Sprintf("完整 sing-box 配置无效: %v", err))
			return nil, fmt.Errorf("完整 sing-box 配置无效: %w", err)
		}
		fullOptions = parsed
		hasFullConfig = true
		fullConfig = append([]byte(nil), body...)
	}
	if useProviderRules && !hasFullConfig {
		parsed, err := store.Parse(ctx, body)
		if err != nil || (parsed.Route == nil && parsed.DNS == nil) {
			if err == nil {
				err = fmt.Errorf("订阅中未找到 route 或 dns 配置")
			}
			m.updateProviderError(id, fmt.Sprintf("订阅规则配置无效: %v", err))
			return nil, fmt.Errorf("订阅规则配置无效: %w", err)
		}
		fullOptions = parsed
		hasFullConfig = true
	}
	if hasFullConfig && (fullOptions.Route != nil || fullOptions.DNS != nil) {
		fullConfig = append([]byte(nil), body...)
	}

	outbounds, err := AutoParse(body, format)
	if err != nil {
		m.updateProviderError(id, err.Error())
		return nil, err
	}

	// 保证节点 tag 带有 provider 前缀或唯一性，避免重名冲突
	dedupOutbounds := make([]option.Outbound, 0, len(outbounds))
	tagSeen := make(map[string]bool)
	for i, ob := range outbounds {
		if ob.Tag == "" {
			ob.Tag = fmt.Sprintf("%s-%d", tag, i+1)
		}
		cleanTag := ob.Tag
		count := 1
		for tagSeen[cleanTag] {
			cleanTag = fmt.Sprintf("%s-%d", ob.Tag, count)
			count++
		}
		ob.Tag = cleanTag
		tagSeen[cleanTag] = true
		dedupOutbounds = append(dedupOutbounds, ob)
	}

	// 序列化并落盘为独立的 outbound-<tag>.json 片段（必须使用 sjson 包含上下文，保留完整节点字段）
	var buf bytes.Buffer
	enc := sjson.NewEncoderContext(include.Context(context.Background()), &buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(dedupOutbounds); err != nil {
		return nil, fmt.Errorf("序列化节点片段失败: %w", err)
	}

	if err := os.WriteFile(m.providerOutboundPath(tag), buf.Bytes(), 0o644); err != nil {
		return nil, fmt.Errorf("写入节点片段文件失败: %w", err)
	}
	if hasFullConfig {
		if err := os.WriteFile(m.providerFullPath(tag), fullConfig, 0o644); err != nil {
			return nil, fmt.Errorf("写入完整配置失败: %w", err)
		}
	} else {
		_ = os.Remove(m.providerFullPath(tag))
	}

	// 更新元数据
	m.mu.Lock()
	if existing, ok := m.providers[id]; ok {
		now := time.Now()
		existing.LastUpdated = &now
		existing.LastETag = newETag
		existing.NodeCount = len(dedupOutbounds)
		existing.LastError = ""
		_ = m.saveProvidersLocked()
	}
	m.mu.Unlock()

	logger.Info("[Provider] 成功更新订阅 %s (节点数: %d)", tag, len(dedupOutbounds))

	// 自动闭环：若活动方案中包含该订阅源（或默认方案），自动合并并重载内核
	m.mu.RLock()
	activeID := m.activeProfID
	var shouldReapply bool
	if activeID != "" {
		if prof, ok := m.profiles[activeID]; ok && prof.AutoReload {
			for _, pTag := range prof.ProviderTags {
				if pTag == tag {
					shouldReapply = true
					break
				}
			}
		}
	}
	m.mu.RUnlock()

	if shouldReapply && m.store.Exists() {
		go func() {
			if err := m.ApplyProfile(context.Background(), activeID); err != nil {
				logger.Warn("[Provider] 订阅 %s 更新后自动应用方案失败: %v", tag, err)
			}
		}()
	}

	return dedupOutbounds, nil
}

func (m *Manager) updateProviderError(id string, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.providers[id]; ok {
		p.LastError = errMsg
		_ = m.saveProvidersLocked()
	}
}

// GetProviderOutbounds 读取指定 provider 的独立片段节点
func (m *Manager) GetProviderOutbounds(tag string) ([]option.Outbound, error) {
	filePath := m.providerOutboundPath(tag)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	ctx := include.Context(context.Background())
	outbounds, err := sjson.UnmarshalExtendedContext[[]option.Outbound](ctx, data)
	if err != nil {
		return nil, err
	}
	return outbounds, nil
}

// ActiveProfileID 返回当前活动方案 ID
func (m *Manager) ActiveProfileID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeProfID
}

// ListProfiles 获取所有配置组方案
func (m *Manager) ListProfiles() []*Profile {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*Profile, 0, len(m.profiles))
	for _, p := range m.profiles {
		cp := *p
		list = append(list, &cp)
	}
	return list
}

// GetProfile 获取指定配置组方案
func (m *Manager) GetProfile(id string) (*Profile, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.profiles[id]
	if !ok {
		return nil, errors.New("profile not found")
	}
	cp := *p
	return &cp, nil
}

// AddProfile 创建新配置组方案
func (m *Manager) AddProfile(p Profile) (*Profile, error) {
	if p.Tag == "" {
		return nil, errors.New("tag 不能为空")
	}
	if p.Name == "" {
		p.Name = p.Tag
	}
	if p.ID == "" {
		p.ID = store.NewUUID()
	}
	if len(p.TargetSelectors) == 0 {
		p.TargetSelectors = []string{"Proxy", "Auto"}
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now

	m.mu.Lock()
	m.profiles[p.ID] = &p
	_ = m.saveProfilesLocked()
	m.mu.Unlock()

	return &p, nil
}

// UpdateProfile 更新配置组方案
func (m *Manager) UpdateProfile(p Profile) error {
	m.mu.Lock()
	existing, ok := m.profiles[p.ID]
	if !ok {
		m.mu.Unlock()
		return errors.New("profile not found")
	}

	existing.Name = p.Name
	existing.Description = p.Description
	existing.ProviderTags = p.ProviderTags
	existing.TargetSelectors = p.TargetSelectors
	existing.AutoReload = p.AutoReload
	existing.UpdatedAt = time.Now()
	_ = m.saveProfilesLocked()
	m.mu.Unlock()

	return nil
}

// DeleteProfile 删除配置组方案
func (m *Manager) DeleteProfile(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if id == "default" {
		return errors.New("默认方案不可删除")
	}
	delete(m.profiles, id)
	return m.saveProfilesLocked()
}

// CompileOutbounds 核心管线：合成独立的出站配置文件 (10-outbounds.json)，包含目标选择器组与选中的 Provider 节点
func (m *Manager) CompileOutbounds(ctx context.Context, prof Profile) ([]byte, error) {
	if m.store == nil {
		return nil, errors.New("store 未初始化")
	}

	// 完整配置模式以订阅配置为唯一主体，不与本地配置或其他 Provider 合并。
	var fullConfigTag string
	var rulesConfigTag string
	for _, pTag := range prof.ProviderTags {
		m.mu.RLock()
		var selected *Provider
		for _, p := range m.providers {
			if p.Tag == pTag {
				copy := *p
				selected = &copy
				break
			}
		}
		m.mu.RUnlock()
		if selected != nil && selected.UseFullConfig {
			if fullConfigTag != "" {
				return nil, errors.New("一个配置方案最多只能使用一个完整配置订阅")
			}
			fullConfigTag = selected.Tag
		}
		if selected != nil && selected.UseProviderRules {
			if rulesConfigTag != "" {
				return nil, errors.New("一个配置方案最多只能使用一个订阅规则配置")
			}
			rulesConfigTag = selected.Tag
		}
	}
	if fullConfigTag != "" {
		if len(prof.ProviderTags) != 1 {
			return nil, errors.New("完整配置订阅不能与其他订阅源同时加入同一配置方案")
		}
		content, err := os.ReadFile(m.providerFullPath(fullConfigTag))
		if err != nil {
			return nil, fmt.Errorf("读取完整配置订阅 %s 失败，请先刷新订阅: %w", fullConfigTag, err)
		}
		if _, err := store.Parse(ctx, content); err != nil {
			return nil, fmt.Errorf("完整配置订阅 %s 无效: %w", fullConfigTag, err)
		}
		return content, nil
	}

	// 1. 收集当前 Profile 所选 Provider 的所有节点片段
	var providerOutbounds []option.Outbound
	providerNodeTags := make([]string, 0)

	for _, pTag := range prof.ProviderTags {
		obs, err := m.GetProviderOutbounds(pTag)
		if err != nil {
			logger.Warn("[CompileOutbounds] 读取 Provider %s 节点失败: %v", pTag, err)
			continue
		}
		for _, ob := range obs {
			providerOutbounds = append(providerOutbounds, ob)
			providerNodeTags = append(providerNodeTags, ob.Tag)
		}
	}

	// 2. 组装出站列表（包含基础出站、选择器组以及全部订阅节点）
	var outbounds []option.Outbound

	// 默认兜底直连与阻断
	directJSON := []byte(`{"type": "direct", "tag": "direct"}`)
	if directOb, err := sjson.UnmarshalExtendedContext[option.Outbound](include.Context(ctx), directJSON); err == nil {
		outbounds = append(outbounds, directOb)
	}

	// 构建 Proxy 选择器组
	proxyCandidateTags := make([]string, 0, len(providerNodeTags)+2)
	proxyCandidateTags = append(proxyCandidateTags, "Auto", "direct")
	proxyCandidateTags = append(proxyCandidateTags, providerNodeTags...)

	proxyGroupJSON := map[string]any{
		"type":      "selector",
		"tag":       "Proxy",
		"outbounds": proxyCandidateTags,
	}
	if pData, err := json.Marshal(proxyGroupJSON); err == nil {
		if pOb, err := sjson.UnmarshalExtendedContext[option.Outbound](include.Context(ctx), pData); err == nil {
			outbounds = append(outbounds, pOb)
		}
	}

	// 构建 Auto 自动测速组
	autoCandidateTags := providerNodeTags
	if len(autoCandidateTags) == 0 {
		autoCandidateTags = []string{"direct"}
	}
	autoGroupJSON := map[string]any{
		"type":      "urltest",
		"tag":       "Auto",
		"outbounds": autoCandidateTags,
		"url":       "http://cp.cloudflare.com/generate_204",
		"interval":  "3m0s",
	}
	if aData, err := json.Marshal(autoGroupJSON); err == nil {
		if aOb, err := sjson.UnmarshalExtendedContext[option.Outbound](include.Context(ctx), aData); err == nil {
			outbounds = append(outbounds, aOb)
		}
	}

	// 3. 追加平铺的所有外部节点（去重）
	tagSeen := make(map[string]bool)
	for _, ob := range outbounds {
		tagSeen[ob.Tag] = true
	}
	for _, ob := range providerOutbounds {
		if !tagSeen[ob.Tag] {
			outbounds = append(outbounds, ob)
			tagSeen[ob.Tag] = true
		}
	}

	// 4. 序列化为独立出站 JSON
	outboundOpts := option.Options{Outbounds: outbounds}
	if rulesConfigTag != "" {
		rulesContent, err := os.ReadFile(m.providerFullPath(rulesConfigTag))
		if err != nil {
			return nil, fmt.Errorf("读取订阅规则配置 %s 失败，请先刷新订阅: %w", rulesConfigTag, err)
		}
		rulesOptions, err := store.Parse(ctx, rulesContent)
		if err != nil {
			return nil, fmt.Errorf("订阅规则配置 %s 无效: %w", rulesConfigTag, err)
		}
		outboundOpts.Route = rulesOptions.Route
		outboundOpts.DNS = rulesOptions.DNS
	}
	finalJSON, err := store.MarshalOptions(ctx, outboundOpts)
	if err != nil {
		return nil, fmt.Errorf("序列化独立出站配置失败: %w", err)
	}

	// 5. 联合预检：与当前主配置一起提交 sing-box check 验证
	baseContent := m.store.RawContent()
	if len(baseContent) > 0 {
		validationBase, _, err := store.FilterOverriddenTags(baseContent, finalJSON)
		if err != nil {
			return nil, fmt.Errorf("处理默认配置与外部出站冲突失败: %w", err)
		}
		if err := store.ValidateMulti(ctx, map[string][]byte{
			"00-base.json":      validationBase,
			"10-outbounds.json": finalJSON,
		}); err != nil {
			logger.Warn("[CompileOutbounds] 联合校验预警: %v", err)
		}
	}

	return finalJSON, nil
}

// MergeAndValidateProfile 保留向前兼容接口，内部委托给 CompileOutbounds
func (m *Manager) MergeAndValidateProfile(ctx context.Context, prof Profile) ([]byte, error) {
	return m.CompileOutbounds(ctx, prof)
}

func getGroupOutbounds(ob *option.Outbound) []string {
	switch opts := ob.Options.(type) {
	case *option.SelectorOutboundOptions:
		return opts.Outbounds
	case option.SelectorOutboundOptions:
		return opts.Outbounds
	case *option.URLTestOutboundOptions:
		return opts.Outbounds
	case option.URLTestOutboundOptions:
		return opts.Outbounds
	default:
		return nil
	}
}

func setGroupOutbounds(ob *option.Outbound, tags []string) {
	switch opts := ob.Options.(type) {
	case *option.SelectorOutboundOptions:
		opts.Outbounds = tags
	case option.SelectorOutboundOptions:
		opts.Outbounds = tags
		ob.Options = opts
	case *option.URLTestOutboundOptions:
		opts.Outbounds = tags
	case option.URLTestOutboundOptions:
		opts.Outbounds = tags
		ob.Options = opts
	}
}

// prepareProfile 编译指定方案并写出动态出站配置，不触发内核重启。
func (m *Manager) prepareProfile(ctx context.Context, id string) (*Profile, error) {
	m.mu.RLock()
	prof, ok := m.profiles[id]
	if ok {
		profileCopy := *prof
		prof = &profileCopy
	}
	m.mu.RUnlock()
	if !ok {
		return nil, errors.New("profile not found")
	}

	logger.Info("[Profile] 正在编译方案 %s (%s)...", prof.Name, id)
	outboundsJSON, err := m.CompileOutbounds(ctx, *prof)
	if err != nil {
		logger.Error("[Profile] 方案 %s 出站合成失败: %v", prof.Name, err)
		return nil, err
	}

	// 原子写入独立的 10-outbounds.json
	outboundsPath := m.store.OutboundsPath()
	tmpPath := outboundsPath + ".tmp"
	if err := os.WriteFile(tmpPath, outboundsJSON, 0o644); err != nil {
		return nil, fmt.Errorf("写入独立出站配置文件失败: %w", err)
	}
	if err := os.Rename(tmpPath, outboundsPath); err != nil {
		return nil, fmt.Errorf("生效独立出站配置文件失败: %w", err)
	}
	_ = os.Remove(m.providerRuntimePath())

	if fullConfigTag := m.profileFullConfigTag(*prof); fullConfigTag != "" {
		fullConfig, err := os.ReadFile(m.providerFullPath(fullConfigTag))
		if err != nil {
			return nil, fmt.Errorf("读取完整配置订阅失败: %w", err)
		}
		fullConfig, err = m.withControllerAPIService(fullConfig)
		if err != nil {
			return nil, fmt.Errorf("补充控制器 API service 失败: %w", err)
		}
		if err := os.WriteFile(m.providerRuntimePath(), fullConfig, 0o644); err != nil {
			return nil, fmt.Errorf("写入完整配置运行文件失败: %w", err)
		}
		_ = os.Remove(outboundsPath)
	}
	logger.Info("[Profile] 独立出站配置 %s 已成功写入 (大小: %d 字节)", outboundsPath, len(outboundsJSON))

	// 标记活动方案
	m.mu.Lock()
	for _, p := range m.profiles {
		p.IsActive = (p.ID == id)
	}
	m.activeProfID = id
	_ = m.saveProfilesLocked()
	m.mu.Unlock()
	return prof, nil
}

func (m *Manager) profileFullConfigTag(prof Profile) string {
	for _, pTag := range prof.ProviderTags {
		m.mu.RLock()
		for _, p := range m.providers {
			if p.Tag == pTag && p.UseFullConfig {
				m.mu.RUnlock()
				return p.Tag
			}
		}
		m.mu.RUnlock()
	}
	return ""
}

// PrepareActiveProfile 在启动核心前生成当前方案的动态出站配置。
func (m *Manager) PrepareActiveProfile(ctx context.Context) error {
	id := m.ActiveProfileID()
	if id == "" {
		return nil
	}
	_, err := m.prepareProfile(ctx, id)
	return err
}

// ApplyProfile 应用指定方案：写入动态出站配置，首次应用时创建基础配置，然后重启内核。
func (m *Manager) ApplyProfile(ctx context.Context, id string) error {
	prof, err := m.prepareProfile(ctx, id)
	if err != nil {
		return err
	}

	// 首次初始化时 Store.Load 只在内存中准备默认配置骨架；用户明确应用方案后再落盘。
	if !m.store.Exists() {
		if err := m.store.Save(ctx); err != nil {
			return fmt.Errorf("创建 sing-box 主配置失败: %w", err)
		}
		logger.Info("[Profile] 首次初始化已生成 sing-box 主配置: %s", m.store.Path())
	}

	// 若启用了自动重载且 Supervisor 存在，通知重载或重启内核
	if prof.AutoReload && m.supervisor != nil {
		logger.Info("[Profile] 正在通知 Supervisor 启动/重载 sing-box 内核...")
		if err := m.supervisor.Restart(ctx); err != nil {
			logger.Warn("[Profile] 内核重启失败: %v", err)
			return fmt.Errorf("配置方案已写入，但 sing-box 内核启动失败: %w", err)
		}
	}

	logger.Info("[Profile] 成功应用配置组方案: %s (%s)", prof.Name, prof.Tag)
	return nil
}

// DisableTun 从当前主配置中安全移除所有 type 为 "tun" 的入站，并重启内核
func (m *Manager) DisableTun(ctx context.Context) error {
	if m.store == nil {
		return errors.New("store 未初始化")
	}

	err := m.store.Update(ctx, func(opts *option.Options, meta *store.Meta) error {
		newInbounds := make([]option.Inbound, 0, len(opts.Inbounds))
		for _, in := range opts.Inbounds {
			if in.Type != "tun" {
				newInbounds = append(newInbounds, in)
			}
		}

		// 确保至少有一个入站（若全部被移除则补充默认 mixed 2080 入站）
		if len(newInbounds) == 0 {
			mixedJSON := []byte(`{"type": "mixed", "tag": "in-main", "listen": "127.0.0.1", "listen_port": 2080}`)
			if mixedIn, err := sjson.UnmarshalExtendedContext[option.Inbound](include.Context(ctx), mixedJSON); err == nil {
				newInbounds = append(newInbounds, mixedIn)
			}
		}
		opts.Inbounds = newInbounds
		return nil
	})
	if err != nil {
		return err
	}

	logger.Info("[Provider] 已成功禁用 TUN 入站并原子更新主配置")

	if m.supervisor != nil {
		if err := m.supervisor.Restart(ctx); err != nil {
			logger.Warn("[Provider] 重启内核失败: %v", err)
		}
	}
	return nil
}

// startCronLoop 后台定时轮询自动更新订阅
func (m *Manager) startCronLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.RLock()
		providersToUpdate := make([]string, 0)
		now := time.Now()
		for id, p := range m.providers {
			if p.Type == ProviderTypeRemote && p.Enabled && p.UpdateIntervalMin > 0 {
				if p.LastUpdated == nil || now.Sub(*p.LastUpdated) >= time.Duration(p.UpdateIntervalMin)*time.Minute {
					providersToUpdate = append(providersToUpdate, id)
				}
			}
		}
		activeID := m.activeProfID
		m.mu.RUnlock()

		updatedAny := false
		for _, id := range providersToUpdate {
			if _, err := m.RefreshProvider(context.Background(), id); err == nil {
				updatedAny = true
			}
		}

		// 若当前活动方案包含被更新的 Provider 且开启了 AutoReload，自动重载应用
		if updatedAny && activeID != "" {
			m.mu.RLock()
			activeProf := m.profiles[activeID]
			m.mu.RUnlock()
			if activeProf != nil && activeProf.AutoReload && m.store.Exists() {
				_ = m.ApplyProfile(context.Background(), activeID)
			}
		}
	}
}
