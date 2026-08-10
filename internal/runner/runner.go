// Package runner 管理 sing-box 实例生命周期。
// reload 语义与 sing-box 一致：先关闭旧实例（释放端口），再创建新实例；
// 新实例启动失败时尝试用旧配置回滚，旧实例继续运行。
package runner

import (
	"context"
	"sync"

	"github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
)

type Runner struct {
	mu          sync.Mutex
	instance    *box.Box
	cancel      context.CancelFunc
	lastOptions option.Options
}

func New() *Runner { return &Runner{} }

func (r *Runner) Running() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.instance != nil
}

// Start 启动/替换实例（先停旧的）。
func (r *Runner) Start(ctx context.Context, options option.Options) error {
	return r.replace(ctx, options)
}

// Reload 用新配置重建实例；失败时回滚到旧配置继续运行。
func (r *Runner) Reload(ctx context.Context, options option.Options) error {
	return r.replace(ctx, options)
}

func (r *Runner) replace(ctx context.Context, options option.Options) error {
	r.mu.Lock()
	oldInstance, oldCancel, oldOptions := r.instance, r.cancel, r.lastOptions
	r.mu.Unlock()
	// 先停旧实例释放端口（sing-box SIGHUP 语义）
	if oldInstance != nil {
		_ = oldInstance.Close()
		if oldCancel != nil {
			oldCancel()
		}
	}
	if err := r.startNew(ctx, options); err != nil {
		// 启动失败：尝试回滚旧配置
		if oldInstance != nil {
			if rollbackErr := r.startNew(ctx, oldOptions); rollbackErr != nil {
				return E.Cause(err, "restore previous instance: ", rollbackErr)
			}
		}
		return err
	}
	return nil
}

func (r *Runner) startNew(ctx context.Context, options option.Options) error {
	newCtx, cancel := context.WithCancel(context.Background())
	instance, err := box.New(box.Options{Context: include.Context(newCtx), Options: options})
	if err == nil {
		err = instance.Start()
	}
	if err != nil {
		if instance != nil {
			_ = instance.Close()
		}
		cancel()
		return err
	}
	r.mu.Lock()
	r.instance, r.cancel, r.lastOptions = instance, cancel, options
	r.mu.Unlock()
	return nil
}

// Stop 停止实例。
func (r *Runner) Stop() error {
	r.mu.Lock()
	instance := r.instance
	cancel := r.cancel
	r.instance = nil
	r.cancel = nil
	r.mu.Unlock()
	if instance == nil {
		return nil
	}
	err := instance.Close()
	if cancel != nil {
		cancel()
	}
	return err
}
