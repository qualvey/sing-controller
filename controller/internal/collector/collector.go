package collector

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/qualvey/sing-controller/internal/db"
	"github.com/qualvey/sing-controller/internal/logger"
	"github.com/qualvey/sing-controller/internal/models"
	"github.com/qualvey/sing-controller/internal/realtime"

	"github.com/qualvey/sing-controller/internal/settings"

	"github.com/v2fly/v2ray-core/v5/app/stats/command"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"
)

// StartCollector 启动采集循环：拉 sing-box gRPC 统计 → 增量入库 + 累计 upsert + 实时推送
func StartCollector(ctx context.Context, wg *sync.WaitGroup, cfg *settings.Manager, rt *realtime.Broadcaster) {
	defer wg.Done() // 1. 退出前通知主流程：采集协程已完全结束
	apiAddr := cfg.Values().V2rayAPI
	if apiAddr == "" {
		logger.Warn("[Collector] v2ray_api is empty, collector disabled")
		return // 直接退出，defer 会正常调用 wg.Done()
	}
	interval := cfg.PollIntervalDuration()
	conn, err := grpc.NewClient(cfg.Values().V2rayAPI, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("[Collector] dial sing-box gRPC failed: %v", err)
		return
	}
	defer conn.Close()
	client := command.NewStatsServiceClient(conn)

	logger.Info("[Collector] Connected to sing-box gRPC at %s", cfg.Values().V2rayAPI)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done(): // 收到外部退出或取消信号
			logger.Info("[Collector] stopped: %v", ctx.Err())
			return // 触发 defer 释放 ticker 和 conn
		case <-ticker.C:
			collect(ctx, client, cfg, rt)
			// 动态间隔：有实时监控订阅者时切换到前端指定/默认高频，否则回默认
			target := interval
			if fast := rt.EffectivePollInterval(); fast > 0 {
				target = fast
			}
			if target != interval {
				ticker.Reset(target)
				interval = target
				logger.Info("[Collector] poll interval -> %s", interval)
			}
		}
	}
}

func collect(parentCtx context.Context, client command.StatsServiceClient, cfg *settings.Manager, rt *realtime.Broadcaster) {
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()

	// 分 user / inbound 两类查询，避免大响应
	categories := []struct{ pattern, category string }{
		{"user>>>", "user"},
		{"inbound>>>", "inbound"},
	}
	for _, c := range categories {
		// 每次进入下一个 category 查询前，先检查父 context 是否已取消
		if ctx.Err() != nil {
			logger.Warn("[Collector] collect canceled: %v", ctx.Err())
			return
		}
		queryPattern := c.pattern
		if cfg.Values().Pattern != "" {
			queryPattern = cfg.Values().Pattern + queryPattern
		}
		resp, err := client.QueryStats(ctx, &command.QueryStatsRequest{
			Pattern: queryPattern,
			Reset_:  cfg.Values().ResetOnQuery,
		})
		if err != nil {
			logger.Error("[Collector] QueryStats(%s) error: %v", c.category, err)
			continue
		}
		now := time.Now()
		perTarget := make(map[string]*[2]int64) // tag -> [up, down]

		for _, stat := range resp.Stat {
			// 名称格式: user>>><tag>>>>traffic>>>uplink|downlink 或 inbound>>>...
			parts := strings.Split(stat.Name, ">>>")
			if len(parts) < 4 || parts[0] != c.category || parts[2] != "traffic" {
				continue
			}
			tag := parts[1]
			dir := parts[3]
			entry, ok := perTarget[tag]
			if !ok {
				entry = &[2]int64{}
				perTarget[tag] = entry
			}
			switch dir {
			case "uplink", "up":
				entry[0] = stat.Value
			case "downlink", "down":
				entry[1] = stat.Value
			}
		}

		for tag, v := range perTarget {
			if err := record(now, c.category, tag, v[0], v[1]); err != nil {
				logger.Error("[Collector] record %s/%s failed: %v", c.category, tag, err)
				continue
			}
			rt.Submit(tag, v[0], v[1])
		}
		if len(perTarget) > 0 {
			logger.Debug("[Collector] %s: recorded %d targets", c.category, len(perTarget))
		}
	}
}

// record 写入增量日志并累加总量
func record(ts time.Time, category, tag string, up, down int64) error {
	if up == 0 && down == 0 {
		return nil
	}
	// 增量日志
	logRow := models.TrafficLog{
		Category:      category,
		TargetName:    tag,
		UplinkDelta:   up,
		DownlinkDelta: down,
		Timestamp:     ts,
	}
	if err := db.DB.Create(&logRow).Error; err != nil {
		return err
	}

	// 累计表：存在则累加，不存在则创建（采集为单协程，无并发竞争）
	var total models.TrafficTotal
	err := db.DB.Where("category = ? AND target_name = ?", category, tag).First(&total).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return db.DB.Create(&models.TrafficTotal{
			Category:      category,
			TargetName:    tag,
			UplinkBytes:   up,
			DownlinkBytes: down,
			TotalBytes:    up + down,
			UpdatedAt:     ts,
		}).Error
	}
	return db.DB.Model(&models.TrafficTotal{}).
		Where("id = ?", total.ID).
		Updates(map[string]interface{}{
			"uplink_bytes":   gorm.Expr("uplink_bytes + ?", up),
			"downlink_bytes": gorm.Expr("downlink_bytes + ?", down),
			"total_bytes":    gorm.Expr("total_bytes + ?", up+down),
			"updated_at":     ts,
		}).Error
}
