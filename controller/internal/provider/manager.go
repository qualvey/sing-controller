package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	go m.startCronLoop()
	return m, nil
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
	enabledChanged := existing.Enabled != p.Enabled

	existing.Name = p.Name
	existing.URL = p.URL
	existing.Format = p.Format
	existing.UpdateIntervalMin = p.UpdateIntervalMin
	existing.Enabled = p.Enabled
	existing.UpdatedAt = time.Now()
	_ = m.saveProvidersLocked()
	m.mu.Unlock()

	// 如果 URL、格式或启用状态发生变化且处于启用状态，立即自动重新拉取
	if (urlChanged || formatChanged || enabledChanged) && p.Enabled && p.Type == ProviderTypeRemote && p.URL != "" {
		go func() {
			_, _ = m.RefreshProvider(context.Background(), p.ID)
		}()
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

	if shouldReapply {
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

// MergeAndValidateProfile 核心管线：将 Base 主配置与选中的 Provider 节点片段合并并进行 Validate
func (m *Manager) MergeAndValidateProfile(ctx context.Context, prof Profile) ([]byte, error) {
	if m.store == nil {
		return nil, errors.New("store 未初始化")
	}

	// 1. 读取基础配置副本并解析（若文件尚不存在，使用内存中的默认骨架）
	var baseOpts option.Options
	baseContent, err := os.ReadFile(m.store.Path())
	if errors.Is(err, os.ErrNotExist) {
		baseOpts = m.store.Options
	} else if err != nil {
		return nil, fmt.Errorf("读取主配置失败: %w", err)
	} else {
		parsedOpts, err := store.Parse(ctx, baseContent)
		if err != nil {
			return nil, fmt.Errorf("解析主配置失败: %w", err)
		}
		baseOpts = parsedOpts
	}

	// 2. 收集所有历史上注入过或属于任何 provider 的节点 tag，用于彻底清除旧节点
	m.mu.RLock()
	tagsToPurge := make(map[string]bool)
	for _, t := range m.injectedTags {
		tagsToPurge[t] = true
	}
	for _, p := range m.providers {
		if obs, err := m.GetProviderOutbounds(p.Tag); err == nil {
			for _, ob := range obs {
				tagsToPurge[ob.Tag] = true
			}
		}
	}
	m.mu.RUnlock()

	// 3. 从 baseOpts.Outbounds 中剔除历史注入的 provider 节点，恢复纯净基础节点
	cleanBaseOutbounds := make([]option.Outbound, 0, len(baseOpts.Outbounds))
	for _, ob := range baseOpts.Outbounds {
		if !tagsToPurge[ob.Tag] {
			cleanBaseOutbounds = append(cleanBaseOutbounds, ob)
		}
	}
	baseOpts.Outbounds = cleanBaseOutbounds

	// 4. 从各个 selector / urltest 组中剔除历史 provider 节点 tag
	for i := range baseOpts.Outbounds {
		ob := &baseOpts.Outbounds[i]
		if ob.Type == "selector" || ob.Type == "urltest" {
			groupTags := getGroupOutbounds(ob)
			cleanGroupTags := make([]string, 0, len(groupTags))
			for _, gt := range groupTags {
				if !tagsToPurge[gt] {
					cleanGroupTags = append(cleanGroupTags, gt)
				}
			}
			setGroupOutbounds(ob, cleanGroupTags)
		}
	}

	// 5. 收集当前 Profile 所选 Provider 的节点
	var providerOutbounds []option.Outbound
	providerNodeTags := make([]string, 0)

	for _, pTag := range prof.ProviderTags {
		obs, err := m.GetProviderOutbounds(pTag)
		if err != nil {
			logger.Warn("[Merge] 读取 Provider %s 节点失败: %v", pTag, err)
			continue
		}
		for _, ob := range obs {
			providerOutbounds = append(providerOutbounds, ob)
			providerNodeTags = append(providerNodeTags, ob.Tag)
		}
	}

	// 6. 将当前 Profile 的节点合并到 baseOpts.Outbounds（去重）
	existingTags := make(map[string]bool)
	for _, ob := range baseOpts.Outbounds {
		existingTags[ob.Tag] = true
	}

	for _, ob := range providerOutbounds {
		if !existingTags[ob.Tag] {
			baseOpts.Outbounds = append(baseOpts.Outbounds, ob)
			existingTags[ob.Tag] = true
		}
	}

	// 7. 将节点自动并入目标 Selector / URLTest 组
	targetSet := make(map[string]bool)
	for _, t := range prof.TargetSelectors {
		targetSet[strings.ToLower(t)] = true
	}

	for i := range baseOpts.Outbounds {
		ob := &baseOpts.Outbounds[i]
		if ob.Type == "selector" || ob.Type == "urltest" {
			if targetSet[strings.ToLower(ob.Tag)] || len(targetSet) == 0 {
				groupTags := getGroupOutbounds(ob)
				tagMap := make(map[string]bool)

				for _, t := range groupTags {
					tagMap[t] = true
				}
				for _, t := range providerNodeTags {
					if !tagMap[t] {
						groupTags = append(groupTags, t)
						tagMap[t] = true
					}
				}
				setGroupOutbounds(ob, groupTags)
			}
		}
	}

	// 8. 序列化为最终 JSON
	finalJSON, err := store.MarshalOptions(ctx, baseOpts)
	if err != nil {
		return nil, fmt.Errorf("合并配置序列化失败: %w", err)
	}

	// 9. 执行全量 Validate（box.New 干跑预检）
	if err := store.Validate(ctx, finalJSON); err != nil {
		return nil, fmt.Errorf("合并后配置预检失败: %w", err)
	}

	// 10. 更新当前已注入的节点 tags 并持久化
	m.mu.Lock()
	m.injectedTags = providerNodeTags
	_ = m.saveInjectedTagsLocked()
	m.mu.Unlock()

	return finalJSON, nil
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

// ApplyProfile 应用指定方案：合并、预检、原子覆写盘并重载内核
func (m *Manager) ApplyProfile(ctx context.Context, id string) error {
	m.mu.RLock()
	prof, ok := m.profiles[id]
	m.mu.RUnlock()
	if !ok {
		return errors.New("profile not found")
	}

	finalJSON, err := m.MergeAndValidateProfile(ctx, *prof)
	if err != nil {
		return err
	}

	// 原子写盘
	if err := m.store.RawSave(ctx, finalJSON); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	// 标记活动方案
	m.mu.Lock()
	for _, p := range m.profiles {
		p.IsActive = (p.ID == id)
	}
	m.activeProfID = id
	_ = m.saveProfilesLocked()
	m.mu.Unlock()

	// 若启用了自动重载且 Supervisor 存在，立即重启内核
	if prof.AutoReload && m.supervisor != nil {
		if err := m.supervisor.Restart(ctx); err != nil {
			logger.Warn("[Profile] 切换方案后重载内核失败: %v", err)
		}
	}

	logger.Info("[Profile] 成功应用配置组方案: %s (%s)", prof.Name, prof.Tag)
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
			if activeProf != nil && activeProf.AutoReload {
				_ = m.ApplyProfile(context.Background(), activeID)
			}
		}
	}
}
