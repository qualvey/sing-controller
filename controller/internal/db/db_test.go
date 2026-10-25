package db

import (
	"sync"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestInitDBDisabledWithoutDSN(t *testing.T) {
	setDB(nil)
	if err := InitDB(""); err != ErrDisabled {
		t.Fatalf("InitDB(\"\") = %v, want %v", err, ErrDisabled)
	}
	if err := InitDB("   "); err != ErrDisabled {
		t.Fatalf("InitDB(spaces) = %v, want %v", err, ErrDisabled)
	}
	if Enabled() {
		t.Error("Enabled() = true after disabled init")
	}
	if _, err := Get(); err != ErrDisabled {
		t.Fatalf("Get() = %v, want %v", err, ErrDisabled)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	setDB(nil)
	if err := Close(); err != nil {
		t.Fatalf("Close() = %v, want nil", err)
	}
	if err := Close(); err != nil {
		t.Fatalf("second Close() = %v, want nil", err)
	}
	if Enabled() {
		t.Error("Enabled() = true after close")
	}
}

// TestCloseDetachesGlobalBeforeClosingPool 确认关闭后全局连接立刻失效。
// HTTP handler 仍在服务请求时直接解引用 DB 会拿到已关闭的连接池。
func TestCloseDetachesGlobalBeforeClosingPool(t *testing.T) {
	t.Cleanup(func() { setDB(nil) })
	database, err := gorm.Open(postgres.Open("host=127.0.0.1 port=1 user=none dbname=none sslmode=disable connect_timeout=1"), &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               gormlogger.Discard,
	})
	if err != nil {
		t.Fatalf("open unreachable database: %v", err)
	}
	setDB(database)
	if !Enabled() {
		t.Fatal("Enabled() = false after setDB")
	}
	if err := Close(); err != nil {
		t.Fatalf("Close() = %v", err)
	}
	if Enabled() {
		t.Error("Enabled() = true after Close")
	}
	if _, err := Get(); err != ErrDisabled {
		t.Fatalf("Get() = %v, want %v", err, ErrDisabled)
	}
}

// TestConcurrentAccessDuringClose 覆盖 Engine 关闭期的并发读写：
// Close 在一个协程里清空全局指针，HTTP handler 与采集协程同时在读。
// 配合 -race 运行（go test -race ./internal/db/...）。
func TestConcurrentAccessDuringClose(t *testing.T) {
	t.Cleanup(func() { setDB(nil) })
	database, err := gorm.Open(postgres.Open("host=127.0.0.1 port=1 user=none dbname=none sslmode=disable connect_timeout=1"), &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               gormlogger.Discard,
	})
	if err != nil {
		t.Fatalf("open unreachable database: %v", err)
	}
	setDB(database)

	var wg sync.WaitGroup
	stop := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				_ = Enabled()
				_, _ = Get()
			}
		}()
	}
	if err := Close(); err != nil {
		t.Fatalf("Close() = %v", err)
	}
	close(stop)
	wg.Wait()
	if Enabled() {
		t.Error("Enabled() = true after Close")
	}
}
