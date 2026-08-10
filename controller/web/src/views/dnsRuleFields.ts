// DNS 规则匹配字段元数据
// 从 sing-box fork 源码 option/rule_dns.go RawDefaultDNSRule 整理（testing/1.14 dev）：
// 覆盖全部有效字段（含 rule_set / match_response / response_* 等），无「附加字段」兜底——
// 简单字段用原生控件，复杂对象字段（map/RR 记录）用字段级 JSON 输入，每个字段都有对应 JSON 本体。
// 已废弃字段（outbound、geosite、geoip、source_geoip、rule_set_ip_cidr_accept_empty）不展示。

export type DnsRuleFieldType = 'string-list' | 'uint-list' | 'int-list' | 'bool' | 'string' | 'select' | 'json'

export interface DnsRuleField {
  key: string
  label: string
  group: string
  type: DnsRuleFieldType
  /** 枚举选项（select 单选或 list 的可选项）；为空则仅 allow-create 输入 */
  options?: string[]
  placeholder?: string
}

export const DNS_RULE_GROUPS = ['入站与网络', '域名', 'IP', '端口', '进程与包', '用户', '网络环境', '接口与响应', '其他'] as const

export const DNS_QUERY_TYPES = ['A', 'AAAA', 'CNAME', 'MX', 'NS', 'PTR', 'SOA', 'SRV', 'TXT']
export const DNS_NETWORK_OPTIONS = ['tcp', 'udp']
export const DNS_PROTOCOL_OPTIONS = ['tls', 'http', 'quic', 'dns', 'stun', 'bittorrent', 'dtls', 'ssh', 'rdp', 'ntp']
export const DNS_NETWORK_TYPE_OPTIONS = ['wifi', 'cellular', 'ethernet', 'other']
export const DNS_RCODE_OPTIONS = ['NOERROR', 'FORMERR', 'SERVFAIL', 'NXDOMAIN', 'NOTIMP', 'REFUSED']

export const DNS_RULE_FIELDS: DnsRuleField[] = [
  // 入站与网络
  { key: 'inbound', label: '入站', group: '入站与网络', type: 'string-list', placeholder: '选择或输入入站 tag，回车添加' },
  { key: 'ip_version', label: 'IP 版本', group: '入站与网络', type: 'select', options: ['4', '6'] },
  { key: 'query_type', label: '查询类型', group: '入站与网络', type: 'string-list', options: DNS_QUERY_TYPES },
  { key: 'network', label: '网络', group: '入站与网络', type: 'string-list', options: DNS_NETWORK_OPTIONS },
  { key: 'auth_user', label: '认证用户', group: '入站与网络', type: 'string-list' },
  { key: 'protocol', label: '协议', group: '入站与网络', type: 'string-list', options: DNS_PROTOCOL_OPTIONS },
  // 域名
  { key: 'domain', label: '完整域名', group: '域名', type: 'string-list', placeholder: '如 example.com' },
  { key: 'domain_suffix', label: '域名后缀', group: '域名', type: 'string-list', placeholder: '如 .google.com' },
  { key: 'domain_keyword', label: '域名关键词', group: '域名', type: 'string-list' },
  { key: 'domain_regex', label: '域名正则', group: '域名', type: 'string-list' },
  { key: 'rule_set', label: '规则集', group: '域名', type: 'string-list', placeholder: 'rule_set tag，回车添加' },
  // IP
  { key: 'source_ip_cidr', label: '源 IP/CIDR', group: 'IP', type: 'string-list' },
  { key: 'source_ip_is_private', label: '源 IP 私网', group: 'IP', type: 'bool' },
  { key: 'ip_cidr', label: '目标 IP/CIDR', group: 'IP', type: 'string-list' },
  { key: 'ip_is_private', label: '目标 IP 私网', group: 'IP', type: 'bool' },
  { key: 'ip_accept_any', label: 'IP 任意匹配', group: 'IP', type: 'bool' },
  // 端口
  { key: 'source_port', label: '源端口', group: '端口', type: 'uint-list', placeholder: '输入端口号回车，可多值' },
  { key: 'source_port_range', label: '源端口范围', group: '端口', type: 'string-list', placeholder: '如 1000:2000' },
  { key: 'port', label: '目标端口', group: '端口', type: 'uint-list', placeholder: '输入端口号回车，可多值' },
  { key: 'port_range', label: '端口范围', group: '端口', type: 'string-list', placeholder: '如 1000:2000' },
  // 进程与包
  { key: 'process_name', label: '进程名', group: '进程与包', type: 'string-list' },
  { key: 'process_path', label: '进程路径', group: '进程与包', type: 'string-list' },
  { key: 'process_path_regex', label: '进程路径正则', group: '进程与包', type: 'string-list' },
  { key: 'package_name', label: '包名', group: '进程与包', type: 'string-list' },
  { key: 'package_name_regex', label: '包名正则', group: '进程与包', type: 'string-list' },
  // 用户
  { key: 'user', label: '用户', group: '用户', type: 'string-list' },
  { key: 'user_id', label: '用户 ID', group: '用户', type: 'int-list', placeholder: '输入用户 ID 回车，可多值' },
  // 网络环境
  { key: 'network_type', label: '网络类型', group: '网络环境', type: 'string-list', options: DNS_NETWORK_TYPE_OPTIONS },
  { key: 'network_is_expensive', label: '计费网络', group: '网络环境', type: 'bool' },
  { key: 'network_is_constrained', label: '受限网络', group: '网络环境', type: 'bool' },
  { key: 'wifi_ssid', label: 'WiFi SSID', group: '网络环境', type: 'string-list' },
  { key: 'wifi_bssid', label: 'WiFi BSSID', group: '网络环境', type: 'string-list' },
  { key: 'default_interface_address', label: '默认接口地址', group: '网络环境', type: 'string-list', placeholder: '如 10.0.0.0/8' },
  { key: 'source_mac_address', label: '源 MAC', group: '网络环境', type: 'string-list', placeholder: '如 00:11:22:33:44:55' },
  { key: 'source_hostname', label: '源主机名', group: '网络环境', type: 'string-list' },
  { key: 'preferred_by', label: '偏好来源', group: '网络环境', type: 'string-list' },
  // 接口与响应（复杂结构，字段级 JSON）
  { key: 'interface_address', label: '接口地址', group: '接口与响应', type: 'json', placeholder: '{"eth0": ["10.0.0.0/8"]}' },
  { key: 'network_interface_address', label: '网络接口地址', group: '接口与响应', type: 'json', placeholder: '{"wifi": ["192.168.1.0/24"]}' },
  { key: 'match_response', label: '响应匹配', group: '接口与响应', type: 'json', placeholder: 'true 或 "tag"（evaluate 规则定义的 tag；启用后 ip_cidr/response_* 字段才生效）' },
  { key: 'response_rcode', label: '响应 RCODE', group: '接口与响应', type: 'string', placeholder: '如 NXDOMAIN（留空默认）' },
  { key: 'response_answer', label: '应答记录', group: '接口与响应', type: 'json', placeholder: '[{"name": "example.com", "type": "A", "ttl": 60, "rdata": "1.2.3.4"}]' },
  { key: 'response_ns', label: 'NS 记录', group: '接口与响应', type: 'json' },
  { key: 'response_extra', label: '附加记录', group: '接口与响应', type: 'json' },
  // 其他
  { key: 'clash_mode', label: 'Clash 模式', group: '其他', type: 'string', placeholder: '如 global' },
  { key: 'rule_set_ip_cidr_match_source', label: '规则集匹配源 IP', group: '其他', type: 'bool' },
  { key: 'invert', label: '取反', group: '其他', type: 'bool' }
]

export const DNS_RULE_FIELD_KEYS = DNS_RULE_FIELDS.map((f) => f.key)

/** DNS 规则类型（option/rule_dns.go _DNSRule）：default 匹配字段 / logical 逻辑组合 */
export const DNS_RULE_TYPES = [
  { value: 'default', label: '普通规则（匹配字段）' },
  { value: 'logical', label: '逻辑组合（and/or 嵌套子规则）' }
]

export const LOGICAL_MODES = ['and', 'or']

/** 列表页 DNS 规则摘要的展示优先级 */
export const DNS_RULE_SUMMARY_ORDER = [
  'inbound',
  'network',
  'protocol',
  'query_type',
  'domain',
  'domain_suffix',
  'domain_keyword',
  'domain_regex',
  'rule_set',
  'ip_cidr',
  'source_ip_cidr',
  'port',
  'port_range',
  'source_port',
  'preferred_by'
]

/** DNS 规则动作（option/rule_action.go DNSRuleAction） */
export const DNS_RULE_ACTIONS = [
  { value: 'route', label: '路由到 server（默认）' },
  { value: 'reject', label: '拒绝（reject）' },
  { value: 'respond', label: '直接应答（respond，无参）' },
  { value: 'evaluate', label: '评估（evaluate）' },
  { value: 'route-options', label: '路由选项（route-options）' },
  { value: 'predefined', label: '预定义应答（predefined）' }
] as const
