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
	apiAddr := ""
	if cfg.Values().API.V2Ray != nil {
		apiAddr = cfg.Values().API.V2Ray.Address
	}
	if apiAddr == "" {
		logger.Warn("[Collector] api.v2ray.address is empty, collector disabled")
		return // 直接退出，defer 会正常调用 wg.Done()
	}
	if !db.Enabled() {
		logger.Warn("[Collector] database is not configured, collector disabled")
		return
	}
	interval := cfg.PollIntervalDuration()
	conn, err := grpc.NewClient(apiAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("[Collector] dial sing-box gRPC failed: %v", err)
		return
	}
	defer conn.Close()
	client := command.NewStatsServiceClient(conn)

	logger.Info("[Collector] Connected to sing-box gRPC at %s", apiAddr)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 每日定时清理超出保留期的时序明细日志（保留 60 天），防止数据库体积无限膨胀
	cleanupTicker := time.NewTicker(24 * time.Hour)
	defer cleanupTicker.Stop()
	go func() {
		_, _ = CleanOldLogs(ctx, 60)
	}()

	// 记录上一轮统计数值，用于在 ResetOnQuery=false 时计算真实增量 (Delta)
	lastStats := make(map[string][2]int64)

	for {
		select {
		case <-ctx.Done(): // 收到外部退出或取消信号
			logger.Info("[Collector] stopped: %v", ctx.Err())
			return // 触发 defer 释放 ticker 和 conn
		case <-cleanupTicker.C:
			go func() {
				_, _ = CleanOldLogs(ctx, 60)
			}()
		case <-ticker.C:
			collect(ctx, client, cfg, rt, lastStats)
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

func collect(parentCtx context.Context, client command.StatsServiceClient, cfg *settings.Manager, rt *realtime.Broadcaster, lastStats map[string][2]int64) {
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()

	resetOnQuery := cfg.Values().Stats.ResetOnQuery

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
		if cfg.Values().Stats.Pattern != "" {
			queryPattern = cfg.Values().Stats.Pattern + queryPattern
		}
		resp, err := client.QueryStats(ctx, &command.QueryStatsRequest{
			Pattern: queryPattern,
			Reset_:  resetOnQuery,
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
			statKey := c.category + ">>>" + tag
			var deltaUp, deltaDown int64

			if resetOnQuery {
				// 若开启了 reset_on_query，每次 query 内核已将计数器清零，返回的值即为周期增量
				deltaUp = v[0]
				deltaDown = v[1]
			} else {
				// 未开启 reset_on_query 时内核返回累计总量，需通过差值计算真实增量 (Delta)
				last := lastStats[statKey]
				if v[0] >= last[0] {
					deltaUp = v[0] - last[0]
				} else {
					// 内核重启或计数器被外部重置
					deltaUp = v[0]
				}

				if v[1] >= last[1] {
					deltaDown = v[1] - last[1]
				} else {
					deltaDown = v[1]
				}

				lastStats[statKey] = [2]int64{v[0], v[1]}
			}

			if err := record(now, c.category, tag, deltaUp, deltaDown); err != nil {
				logger.Error("[Collector] record %s/%s failed: %v", c.category, tag, err)
				continue
			}
			rt.Submit(tag, deltaUp, deltaDown)
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
	// 取一次连接快照：Engine 关闭会在另一个协程里把全局 DB 置空，
	// 直接解引用 db.DB 会与那次写入形成数据竞争（甚至 panic）。
	database, err := db.Get()
	if err != nil {
		return err
	}
	// 增量日志
	logRow := models.TrafficLog{
		Category:      category,
		TargetName:    tag,
		UplinkDelta:   up,
		DownlinkDelta: down,
		Timestamp:     ts,
	}
	if err := database.Create(&logRow).Error; err != nil {
		return err
	}

	// 累计表：存在则累加，不存在则创建（采集为单协程，无并发竞争）
	var total models.TrafficTotal
	err = database.Where("category = ? AND target_name = ?", category, tag).First(&total).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return database.Create(&models.TrafficTotal{
			Category:      category,
			TargetName:    tag,
			UplinkBytes:   up,
			DownlinkBytes: down,
			TotalBytes:    up + down,
			UpdatedAt:     ts,
		}).Error
	}
	return database.Model(&models.TrafficTotal{}).
		Where("id = ?", total.ID).
		Updates(map[string]interface{}{
			"uplink_bytes":   gorm.Expr("uplink_bytes + ?", up),
			"downlink_bytes": gorm.Expr("downlink_bytes + ?", down),
			"total_bytes":    gorm.Expr("total_bytes + ?", up+down),
			"updated_at":     ts,
		}).Error
}

// CleanOldLogs 定期清理指定天数前的 TrafficLog 增量明细，防止 SQLite/Postgres 时序表无限膨胀
func CleanOldLogs(ctx context.Context, retentionDays int) (int64, error) {
	database, err := db.Get()
	if err != nil || retentionDays <= 0 {
		return 0, nil
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	res := database.WithContext(ctx).Where("timestamp < ?", cutoff).Delete(&models.TrafficLog{})
	if res.Error != nil {
		logger.Error("[Collector] CleanOldLogs error: %v", res.Error)
		return 0, res.Error
	}
	if res.RowsAffected > 0 {
		logger.Info("[Collector] Cleaned %d traffic logs older than %s", res.RowsAffected, cutoff.Format("2006-01-02"))
	}
	return res.RowsAffected, nil
}
