// sing-box route 规则匹配字段元数据
// 从 sing-box fork 源码 option/rule.go RawDefaultRule 整理（testing/1.14 dev）：
// 覆盖除 interface_address / network_interface_address / default_interface_address
// （复杂 map 结构，走 extraJson 兜底）与 deprecated rule_set_ipcidr_match_source 外的全部字段。
// 后端 CRUD 使用 option.Rule 严格解码，任意合法字段天然支持，前端据此动态渲染表单。

export type RuleFieldType = 'string-list' | 'uint-list' | 'int-list' | 'bool' | 'string' | 'select'

export interface RuleField {
  key: string
  label: string
  group: string
  type: RuleFieldType
  /** 枚举选项（select 单选或 list 的可选项）；为空则仅 allow-create 输入 */
  options?: string[]
  placeholder?: string
}

export const RULE_GROUPS = [
  '入站与网络',
  '域名',
  'IP',
  '端口',
  '进程与包',
  '用户',
  '网络环境',
  '其他'
] as const

export const NETWORK_OPTIONS = ['tcp', 'udp', 'icmp']
export const PROTOCOL_OPTIONS = ['tls', 'http', 'quic', 'dns', 'stun', 'bittorrent', 'dtls', 'ssh', 'rdp', 'ntp']
export const NETWORK_TYPE_OPTIONS = ['wifi', 'cellular', 'ethernet', 'other']

export const RULE_FIELDS: RuleField[] = [
  // 入站与网络
  { key: 'inbound', label: '入站', group: '入站与网络', type: 'string-list', placeholder: '选择或输入入站 tag，回车添加' },
  { key: 'ip_version', label: 'IP 版本', group: '入站与网络', type: 'select', options: ['4', '6'] },
  { key: 'network', label: '网络', group: '入站与网络', type: 'string-list', options: NETWORK_OPTIONS },
  { key: 'protocol', label: '协议', group: '入站与网络', type: 'string-list', options: PROTOCOL_OPTIONS },
  { key: 'auth_user', label: '认证用户', group: '入站与网络', type: 'string-list' },
  { key: 'client', label: '客户端', group: '入站与网络', type: 'string-list' },
  // 域名
  { key: 'domain', label: '完整域名', group: '域名', type: 'string-list', placeholder: '如 example.com' },
  { key: 'domain_suffix', label: '域名后缀', group: '域名', type: 'string-list', placeholder: '如 .google.com' },
  { key: 'domain_keyword', label: '域名关键词', group: '域名', type: 'string-list' },
  { key: 'domain_regex', label: '域名正则', group: '域名', type: 'string-list' },
  { key: 'geosite', label: 'GeoSite 分类', group: '域名', type: 'string-list', placeholder: '如 cn、geolocation-!cn' },
  { key: 'rule_set', label: '规则集', group: '域名', type: 'string-list', placeholder: 'rule_set 出站 tag' },
  // IP
  { key: 'source_ip_cidr', label: '源 IP/CIDR', group: 'IP', type: 'string-list', placeholder: '如 192.168.0.0/16' },
  { key: 'source_ip_is_private', label: '源 IP 私网', group: 'IP', type: 'bool' },
  { key: 'ip_cidr', label: '目标 IP/CIDR', group: 'IP', type: 'string-list', placeholder: '如 1.1.1.1、10.0.0.0/8' },
  { key: 'ip_is_private', label: '目标 IP 私网', group: 'IP', type: 'bool' },
  { key: 'source_geoip', label: '源 GeoIP', group: 'IP', type: 'string-list', placeholder: '如 cn' },
  { key: 'geoip', label: '目标 GeoIP', group: 'IP', type: 'string-list', placeholder: '如 cn' },
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
  { key: 'network_type', label: '网络类型', group: '网络环境', type: 'string-list', options: NETWORK_TYPE_OPTIONS },
  { key: 'network_is_expensive', label: '计费网络', group: '网络环境', type: 'bool' },
  { key: 'network_is_constrained', label: '受限网络', group: '网络环境', type: 'bool' },
  { key: 'wifi_ssid', label: 'WiFi SSID', group: '网络环境', type: 'string-list' },
  { key: 'wifi_bssid', label: 'WiFi BSSID', group: '网络环境', type: 'string-list' },
  { key: 'source_mac_address', label: '源 MAC', group: '网络环境', type: 'string-list', placeholder: '如 00:11:22:33:44:55' },
  { key: 'source_hostname', label: '源主机名', group: '网络环境', type: 'string-list' },
  // 其他
  { key: 'preferred_by', label: '偏好来源', group: '其他', type: 'string-list' },
  { key: 'clash_mode', label: 'Clash 模式', group: '其他', type: 'string', placeholder: '如 global' },
  { key: 'invert', label: '取反', group: '其他', type: 'bool' }
]

export const RULE_FIELD_KEYS = RULE_FIELDS.map((f) => f.key)

/** 列表页 rule 摘要的展示优先级 */
export const RULE_SUMMARY_ORDER = [
  'inbound',
  'network',
  'protocol',
  'domain',
  'domain_suffix',
  'domain_keyword',
  'domain_regex',
  'ip_cidr',
  'source_ip_cidr',
  'ip_is_private',
  'source_ip_is_private',
  'ip_accept_any',
  'port',
  'port_range',
  'source_port',
  'source_port_range',
  'process_name',
  'process_path',
  'package_name',
  'user',
  'auth_user',
  'geoip',
  'source_geoip',
  'geosite',
  'rule_set',
  'clash_mode'
]
