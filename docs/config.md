```json
{
  "config": "./sing-box-config.json",
  "listen": "127.0.0.1:8080",
  "min_port": 8000,
  "poll_interval": "10s",
  "pattern": "",
  "reset_on_query": false,
  "v2ray_api": "127.0.0.1:10085",
  "log": {
    "level": "info"
  },
  "defaults": {
    "inbound_type": "mixed",
    "outbound_type": "vless",
    "listen": "127.0.0.1",
    "listen_port": 2080,
    "attach_to_selector": true,
    "proxy_selector": "Proxy"
  },
  "reload": {
    "mode": "auto",
    "service": "sing-box",
    "pid_file": "/var/run/sing-box.pid",
    "hook": "/usr/local/bin/reload-sing-box.sh",
    "after_save": false
  },
  "clash_api": {
    "address": "http://127.0.0.1:9090",
    "secret": ""
  },
  "service_api": {
    "address": "http://127.0.0.1:9900",
    "secret": ""
  },
  "real_time": {
    "enabled": false,
    "interval_ms": 1000,
    "online_threshold_sec": 60
  }
}
```