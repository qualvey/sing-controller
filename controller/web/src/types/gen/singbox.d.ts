/* 自动由 sing-box schema 生成，请勿手动修改 */

export type DNSServer =
  | {
      type: "dhcp";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      prefer_go?: boolean;
      neighbor_domain?: string | string[];
      interface?: string;
    }
  | {
      type: "fakeip";
      tag?: string;
      inet4_range?: string;
      inet6_range?: string;
    }
  | {
      type: "h3";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
      tls?: OutboundTLSOptions;
      path?: string;
      method?: string;
      headers?: HTTPHeader;
    }
  | {
      type: "hosts";
      tag?: string;
      path?: string | string[];
      predefined?: {
        [k: string]: string | string[];
      };
    }
  | {
      type: "https";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
      tls?: OutboundTLSOptions;
      path?: string;
      method?: string;
      headers?: HTTPHeader;
    }
  | {
      type: "local";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      prefer_go?: boolean;
      neighbor_domain?: string | string[];
    }
  | {
      type: "mdns";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      prefer_go?: boolean;
      neighbor_domain?: string | string[];
      interface?: string | string[];
    }
  | {
      type: "openconnect";
      tag?: string;
      endpoint?: string;
      accept_default_resolvers?: boolean;
      accept_search_domain?: boolean;
    }
  | {
      type: "openvpn";
      tag?: string;
      endpoint?: string;
      accept_default_resolvers?: boolean;
      accept_search_domain?: boolean;
    }
  | {
      type: "quic";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
      tls?: OutboundTLSOptions;
    }
  | {
      type: "resolved";
      tag?: string;
      service?: string;
      accept_default_resolvers?: boolean;
    }
  | {
      type: "tailscale";
      tag?: string;
      endpoint?: string;
      accept_default_resolvers?: boolean;
      accept_search_domain?: boolean;
    }
  | {
      type: "tcp";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
    }
  | {
      type: "tls";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
      tls?: OutboundTLSOptions;
    }
  | {
      type: "udp";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
    };
export type Duration = string;
export type DomainResolver =
  | string
  | {
      server: string;
      timeout?: Duration;
      strategy?: DomainStrategy;
      disable_cache?: boolean;
      disable_optimistic_cache?: boolean;
      rewrite_ttl?: number;
      client_subnet?: string;
    };
export type DomainStrategy = "" | "as_is" | "prefer_ipv4" | "prefer_ipv6" | "ipv4_only" | "ipv6_only";
export type InterfaceType = "cellular" | "ethernet" | "other" | "wifi";
export type DNSRule =
  | ({
      type?: "default" | "";
      inbound?: string | string[];
      ip_version?: 4 | 6;
      query_type?: DNSQueryType | DNSQueryType[];
      query_client_subnet?: string | string[];
      query_dnssec?: boolean;
      network?: ("tcp" | "udp") | ("tcp" | "udp")[];
      auth_user?: string | string[];
      protocol?:
        | ("tls" | "http" | "quic" | "dns" | "stun" | "bittorrent" | "dtls" | "ssh" | "rdp" | "ntp")
        | ("tls" | "http" | "quic" | "dns" | "stun" | "bittorrent" | "dtls" | "ssh" | "rdp" | "ntp")[];
      domain?: string | string[];
      domain_suffix?: string | string[];
      domain_keyword?: string | string[];
      domain_regex?: string | string[];
      source_ip_cidr?: string | string[];
      source_ip_is_private?: boolean;
      source_port?: number | number[];
      source_port_range?: string | string[];
      port?: number | number[];
      port_range?: string | string[];
      process_name?: string | string[];
      process_path?: string | string[];
      process_path_regex?: string | string[];
      package_name?: string | string[];
      package_name_regex?: string | string[];
      user?: string | string[];
      user_id?: number | number[];
      clash_mode?: string;
      network_type?: InterfaceType | InterfaceType[];
      network_is_expensive?: boolean;
      network_is_constrained?: boolean;
      wifi_ssid?: string | string[];
      wifi_bssid?: string | string[];
      interface_address?: {
        [k: string]: string | string[];
      };
      network_interface_address?: {
        [k: string]: string | string[];
      };
      default_interface_address?: string | string[];
      source_mac_address?: string | string[];
      source_hostname?: string | string[];
      preferred_by?: string | string[];
      rule_set?: string | string[];
      rule_set_ip_cidr_match_source?: boolean;
      match_response?: boolean | string;
      ip_cidr?: string | string[];
      ip_is_private?: boolean;
      ip_accept_any?: boolean;
      response_rcode?: DNSRCode;
      response_answer?: string | string[];
      response_ns?: string | string[];
      response_extra?: string | string[];
      invert?: boolean;
      [k: string]: unknown;
    } & DNSRuleAction)
  | ({
      type: "logical";
      mode: "and" | "or";
      rules: NestedDNSRule[];
      invert?: boolean;
      [k: string]: unknown;
    } & DNSRuleAction);
export type DNSQueryType =
  | number
  | (
      | "A"
      | "AAAA"
      | "AFSDB"
      | "AMTRELAY"
      | "ANY"
      | "APL"
      | "ATMA"
      | "AVC"
      | "AXFR"
      | "CAA"
      | "CDNSKEY"
      | "CDS"
      | "CERT"
      | "CNAME"
      | "CSYNC"
      | "DHCID"
      | "DLV"
      | "DNAME"
      | "DNSKEY"
      | "DS"
      | "EID"
      | "EUI48"
      | "EUI64"
      | "GID"
      | "GPOS"
      | "HINFO"
      | "HIP"
      | "HTTPS"
      | "IPSECKEY"
      | "ISDN"
      | "IXFR"
      | "KEY"
      | "KX"
      | "L32"
      | "L64"
      | "LOC"
      | "LP"
      | "MAILA"
      | "MAILB"
      | "MB"
      | "MD"
      | "MF"
      | "MG"
      | "MINFO"
      | "MR"
      | "MX"
      | "NAPTR"
      | "NID"
      | "NIMLOC"
      | "NINFO"
      | "NS"
      | "NSAP-PTR"
      | "NSEC"
      | "NSEC3"
      | "NSEC3PARAM"
      | "NULL"
      | "NXNAME"
      | "NXT"
      | "None"
      | "OPENPGPKEY"
      | "OPT"
      | "PTR"
      | "PX"
      | "RESINFO"
      | "RKEY"
      | "RP"
      | "RRSIG"
      | "RT"
      | "Reserved"
      | "SIG"
      | "SMIMEA"
      | "SOA"
      | "SPF"
      | "SRV"
      | "SSHFP"
      | "SVCB"
      | "TA"
      | "TALINK"
      | "TKEY"
      | "TLSA"
      | "TSIG"
      | "TXT"
      | "UID"
      | "UINFO"
      | "UNSPEC"
      | "URI"
      | "X25"
      | "ZONEMD"
    );
export type DNSRCode =
  | number
  | (
      | "NOERROR"
      | "FORMERR"
      | "SERVFAIL"
      | "NXDOMAIN"
      | "NOTIMP"
      | "NOTIMPL"
      | "REFUSED"
      | "YXDOMAIN"
      | "YXRRSET"
      | "NXRRSET"
      | "NOTAUTH"
      | "NOTZONE"
      | "DSOTYPENI"
      | "BADSIG"
      | "BADKEY"
      | "BADTIME"
      | "BADMODE"
      | "BADNAME"
      | "BADALG"
      | "BADTRUNC"
      | "BADCOOKIE"
    );
export type DNSRuleAction =
  | {
      action?: "route";
      race?: boolean;
      server?: string;
      speculative?: boolean;
      timeout?: Duration;
      disable_cache?: boolean;
      disable_optimistic_cache?: boolean;
      rewrite_ttl?: number;
      client_subnet?: string;
      remove_client_subnet?: boolean;
      [k: string]: unknown;
    }
  | {
      action: "evaluate";
      race?: boolean;
      server?: string;
      tag?: string;
      speculative?: boolean;
      timeout?: Duration;
      disable_cache?: boolean;
      disable_optimistic_cache?: boolean;
      rewrite_ttl?: number;
      client_subnet?: string;
      remove_client_subnet?: boolean;
      [k: string]: unknown;
    }
  | {
      action: "respond";
      race?: boolean;
      [k: string]: unknown;
    }
  | {
      action: "route-options";
      race?: boolean;
      timeout?: Duration;
      disable_cache?: boolean;
      disable_optimistic_cache?: boolean;
      rewrite_ttl?: number;
      client_subnet?: string;
      remove_client_subnet?: boolean;
      [k: string]: unknown;
    }
  | {
      action: "reject";
      race?: boolean;
      method?: "" | "default" | "drop" | "reply";
      no_drop?: boolean;
      [k: string]: unknown;
    }
  | {
      action: "predefined";
      race?: boolean;
      rcode?: DNSRCode;
      answer?: string | string[];
      ns?: string | string[];
      extra?: string | string[];
      [k: string]: unknown;
    };
export type NestedDNSRule =
  | {
      type?: "default" | "";
      inbound?: string | string[];
      ip_version?: 4 | 6;
      query_type?: DNSQueryType | DNSQueryType[];
      query_client_subnet?: string | string[];
      query_dnssec?: boolean;
      network?: ("tcp" | "udp") | ("tcp" | "udp")[];
      auth_user?: string | string[];
      protocol?:
        | ("tls" | "http" | "quic" | "dns" | "stun" | "bittorrent" | "dtls" | "ssh" | "rdp" | "ntp")
        | ("tls" | "http" | "quic" | "dns" | "stun" | "bittorrent" | "dtls" | "ssh" | "rdp" | "ntp")[];
      domain?: string | string[];
      domain_suffix?: string | string[];
      domain_keyword?: string | string[];
      domain_regex?: string | string[];
      source_ip_cidr?: string | string[];
      source_ip_is_private?: boolean;
      source_port?: number | number[];
      source_port_range?: string | string[];
      port?: number | number[];
      port_range?: string | string[];
      process_name?: string | string[];
      process_path?: string | string[];
      process_path_regex?: string | string[];
      package_name?: string | string[];
      package_name_regex?: string | string[];
      user?: string | string[];
      user_id?: number | number[];
      clash_mode?: string;
      network_type?: InterfaceType | InterfaceType[];
      network_is_expensive?: boolean;
      network_is_constrained?: boolean;
      wifi_ssid?: string | string[];
      wifi_bssid?: string | string[];
      interface_address?: {
        [k: string]: string | string[];
      };
      network_interface_address?: {
        [k: string]: string | string[];
      };
      default_interface_address?: string | string[];
      source_mac_address?: string | string[];
      source_hostname?: string | string[];
      preferred_by?: string | string[];
      rule_set?: string | string[];
      rule_set_ip_cidr_match_source?: boolean;
      match_response?: boolean | string;
      ip_cidr?: string | string[];
      ip_is_private?: boolean;
      ip_accept_any?: boolean;
      response_rcode?: DNSRCode;
      response_answer?: string | string[];
      response_ns?: string | string[];
      response_extra?: string | string[];
      invert?: boolean;
    }
  | {
      type: "logical";
      mode: "and" | "or";
      rules: NestedDNSRule[];
      invert?: boolean;
    };
export type CertificateProvider =
  | {
      type: "acme";
      tag?: string;
      domain?: string | string[];
      data_directory?: string;
      default_server_name?: string;
      email?: string;
      provider?: string;
      account_key?: string;
      disable_http_challenge?: boolean;
      disable_tls_alpn_challenge?: boolean;
      alternative_http_port?: number;
      alternative_tls_port?: number;
      external_account?: ACMEExternalAccountOptions;
      dns01_challenge?: ACMEProviderDNS01Challenge;
      key_type?: "ed25519" | "p256" | "p384" | "rsa2048" | "rsa4096";
      profile?: string;
      http_client?: HTTPClientReference;
    }
  | {
      type: "cloudflare-origin-ca";
      tag?: string;
      domain?: string | string[];
      data_directory?: string;
      api_token?: string;
      origin_ca_key?: string;
      request_type?: "origin-rsa" | "origin-ecc";
      requested_validity?: 0 | 7 | 30 | 90 | 365 | 730 | 1095 | 5475;
      http_client?: HTTPClientReference;
    }
  | {
      type: "tailscale";
      tag?: string;
      endpoint?: string;
    };
export type ACMEProviderDNS01Challenge =
  | {
      provider: "alidns";
      ttl?: Duration;
      propagation_delay?: Duration;
      propagation_timeout?: Duration;
      resolvers?: string | string[];
      override_domain?: string;
      access_key_id?: string;
      access_key_secret?: string;
      region_id?: string;
      security_token?: string;
    }
  | {
      provider: "cloudflare";
      ttl?: Duration;
      propagation_delay?: Duration;
      propagation_timeout?: Duration;
      resolvers?: string | string[];
      override_domain?: string;
      api_token?: string;
      zone_token?: string;
    }
  | {
      provider: "acmedns";
      ttl?: Duration;
      propagation_delay?: Duration;
      propagation_timeout?: Duration;
      resolvers?: string | string[];
      override_domain?: string;
      username?: string;
      password?: string;
      subdomain?: string;
      server_url?: string;
    };
export type HTTPClientReference =
  | string
  | {
      engine?: "go" | "apple";
      version?: 0 | 1 | 2 | 3;
      disable_version_fallback?: boolean;
      headers?: HTTPHeader;
      tls?: OutboundTLSOptions;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      idle_timeout?: Duration;
      keep_alive_period?: Duration;
      stream_receive_window?: number | string;
      connection_receive_window?: number | string;
      max_concurrent_streams?: number;
      initial_packet_size?: number;
      disable_path_mtu_discovery?: boolean;
    };
export type NetworkNamespace =
  | {
      type?: "default" | "";
      tag: string;
      path?: string;
    }
  | {
      type: "unshare";
      tag: string;
      pid_file?: string;
    };
export type Endpoint =
  | {
      type: "openconnect";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      system?: boolean;
      name?: string;
      udp_timeout?: Duration;
      udp_mapping?: "" | "endpoint_independent" | "address_dependent" | "address_and_port_dependent";
      udp_filtering?: "" | "endpoint_independent" | "address_dependent" | "address_and_port_dependent";
      udp_nat_max?: number;
      server?: string;
      flavor?: "anyconnect" | "gp" | "fortinet" | "f5" | "pulse" | "nc";
      username?: string;
      password?: string;
      auth_group?: string;
      cookie?: string;
      token?: OpenConnectTokenOptions;
      reported_os?: string;
      user_agent?: string;
      version?: string;
      local_hostname?: string;
      mobile?: OpenConnectMobileOptions;
      csd?: OpenConnectCSDOptions;
      hip?: OpenConnectHIPOptions;
      tncc?: OpenConnectTNCCOptions;
      fortinet_host_check?: OpenConnectFortinetHostCheckOptions;
      no_udp?: boolean;
      dtls_local_port?: number;
      compression_disabled?: boolean;
      compression_mode?: "stateless" | "all";
      ipv6_disabled?: boolean;
      http_keepalive_disabled?: boolean;
      xml_post_disabled?: boolean;
      external_auth_disabled?: boolean;
      password_authentication_disabled?: boolean;
      tcp_keep_alive_enabled?: boolean;
      pfs?: boolean;
      mtu?: number;
      base_mtu?: number;
      dpd_interval?: Duration;
      reconnect_timeout?: Duration;
      trojan_interval?: Duration;
      queue_length?: number;
      allow_insecure_crypto?: boolean;
      tls?: OpenConnectTLSOptions;
      form_entries?: OpenConnectFormEntryOptions[];
    }
  | {
      type: "openvpn-client";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
      system?: boolean;
      name?: string;
      mtu?: number;
      udp_mapping?: "" | "endpoint_independent" | "address_dependent" | "address_and_port_dependent";
      udp_filtering?: "" | "endpoint_independent" | "address_dependent" | "address_and_port_dependent";
      udp_nat_max?: number;
      mode?: "tls" | "static_key";
      network?: "udp" | "udp4" | "udp6" | "tcp" | "tcp4" | "tcp6";
      servers?: OpenVPNRemoteOptions[];
      remote_random?: boolean;
      address?: string | string[];
      peer_address?: string;
      peer_address_ipv6?: string;
      topology?: "net30" | "p2p" | "subnet";
      username?: string;
      password?: string;
      auth_retry?: "none" | "nointeract" | "interact";
      static_challenge?: string;
      static_challenge_echo?: boolean;
      static_key?: string | string[];
      static_key_path?: string;
      key_direction?: "server" | "client";
      tls?: OpenVPNOutboundTLSOptions;
      cipher?: string;
      data_ciphers?: string | string[];
      data_ciphers_fallback?: string;
      auth?: string;
      mss_fix?: number;
      mss_fix_disabled?: boolean;
      mss_fix_mode?: "mtu" | "fixed";
      fragment?: number;
      replay_window?: number;
      replay_window_time?: Duration;
      compression?: "none" | "no" | "lz4" | "lz4-v2" | "stub" | "stub-v2" | "disabled" | "off";
      compression_lzo?: "none" | "no" | "yes" | "adaptive" | "asym" | "disabled" | "off";
      allow_compression?: "no" | "asym" | "yes";
      route_no_pull?: boolean;
      pull_filters?: OpenVPNPullFilterOptions[];
      routes?: string | string[];
      route_gateway?: string;
      route_metric?: number;
      redirect_gateway?: boolean;
      redirect_gateway_flags?: string | string[];
      redirect_private?: boolean;
      block_ipv6?: boolean;
      ping_interval?: Duration;
      ping_restart?: Duration;
      ping_restart_disabled?: boolean;
      renegotiate_interval?: Duration;
      renegotiate_disabled?: boolean;
      renegotiate_bytes?: number;
      renegotiate_packets?: number;
      tls_timeout?: Duration;
      handshake_window?: Duration;
      explicit_exit_notify?: number;
      udp_timeout?: UDPTimeout;
    }
  | {
      type: "openvpn-server";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      system?: boolean;
      name?: string;
      mtu?: number;
      udp_mapping?: "" | "endpoint_independent" | "address_dependent" | "address_and_port_dependent";
      udp_filtering?: "" | "endpoint_independent" | "address_dependent" | "address_and_port_dependent";
      udp_nat_max?: number;
      mode?: "tls" | "static_key";
      network?: "tcp" | "udp";
      remote?: string;
      remote_port?: number;
      max_clients?: number;
      address?: string | string[];
      peer_address?: string;
      peer_address_ipv6?: string;
      topology?: "net30" | "p2p" | "subnet";
      duplicate_cn?: boolean;
      users?: User[];
      static_key?: string | string[];
      static_key_path?: string;
      key_direction?: "server" | "client";
      tls?: OpenVPNInboundTLSOptions;
      cipher?: string;
      data_ciphers?: string | string[];
      data_ciphers_fallback?: string;
      auth?: string;
      mss_fix?: number;
      mss_fix_disabled?: boolean;
      mss_fix_mode?: "mtu" | "fixed";
      replay_window?: number;
      replay_window_time?: Duration;
      push?: OpenVPNPushOptions;
      ping_interval?: Duration;
      ping_restart?: Duration;
      renegotiate_interval?: Duration;
      renegotiate_disabled?: boolean;
      renegotiate_bytes?: number;
      renegotiate_packets?: number;
      handshake_window?: Duration;
    }
  | {
      type: "tailscale";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      state_directory?: string;
      auth_key?: string;
      control_url?: string;
      ephemeral?: boolean;
      hostname?: string;
      accept_routes?: boolean;
      exit_node?: string;
      exit_node_allow_lan_access?: boolean;
      advertise_routes?: string[];
      advertise_exit_node?: boolean;
      advertise_tags?: string | string[];
      listen_port?: number;
      relay_server_port?: number;
      relay_server_static_endpoints?: string[];
      system_interface?: boolean;
      system_interface_name?: string;
      system_interface_mtu?: number;
      udp_timeout?: UDPTimeout;
      ssh_server?:
        | boolean
        | {
            enabled?: boolean;
            disable_pty?: boolean;
            disable_sftp?: boolean;
            disable_forwarding?: boolean;
          };
      taildrop_directory?: string;
    }
  | {
      type: "wireguard";
      tag?: string;
      system?: boolean;
      name?: string;
      mtu?: number;
      address?: string | string[];
      private_key?: string;
      listen_port?: number;
      peers?: WireGuardPeer[];
      udp_timeout?: Duration;
      udp_mapping?: "" | "endpoint_independent" | "address_dependent" | "address_and_port_dependent";
      udp_filtering?: "" | "endpoint_independent" | "address_dependent" | "address_and_port_dependent";
      udp_nat_max?: number;
      workers?: number;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
    };
export type UDPTimeout = number | string;
export type Inbound =
  | {
      type: "anytls";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      tls?: InboundTLSOptions;
      users?: AnyTLSUser[];
      padding_scheme?: string | string[];
    }
  | {
      type: "cloudflared";
      tag?: string;
      token?: string;
      ha_connections?: number;
      protocol?: "auto" | "quic" | "http2" | "h2mux";
      post_quantum?: boolean;
      edge_ip_version?: 0 | 4 | 6;
      datagram_version?: "v2" | "v3";
      grace_period?: Duration;
      region?: string;
      control_dialer?: DialerOptions;
      tunnel_dialer?: DialerOptions;
    }
  | {
      type: "direct";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      network?: ("tcp" | "udp") | ("tcp" | "udp")[];
      override_address?: string;
      override_port?: number;
    }
  | {
      type: "http";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      users?: User[];
      domain_resolver?: DomainResolver;
      set_system_proxy?: boolean;
      tls?: InboundTLSOptions;
    }
  | {
      type: "hysteria";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      up?: number | string;
      up_mbps?: number;
      down?: number | string;
      down_mbps?: number;
      obfs?: string;
      users?: HysteriaUser[];
      tls?: InboundTLSOptions;
      idle_timeout?: Duration;
      keep_alive_period?: Duration;
      stream_receive_window?: number | string;
      connection_receive_window?: number | string;
      max_concurrent_streams?: number;
      initial_packet_size?: number;
      disable_path_mtu_discovery?: boolean;
    }
  | {
      type: "hysteria2";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      up_mbps?: number;
      down_mbps?: number;
      obfs?:
        | {
            type: "salamander";
            password?: string;
          }
        | {
            type: "gecko";
            password?: string;
            min_packet_size?: number;
            max_packet_size?: number;
          };
      users?: Hysteria2User[];
      ignore_client_bandwidth?: boolean;
      tls?: InboundTLSOptions;
      idle_timeout?: Duration;
      keep_alive_period?: Duration;
      stream_receive_window?: number | string;
      connection_receive_window?: number | string;
      max_concurrent_streams?: number;
      initial_packet_size?: number;
      disable_path_mtu_discovery?: boolean;
      masquerade?:
        | string
        | (
            | {
                type: "file";
                directory?: string;
              }
            | {
                type: "proxy";
                url?: string;
                rewrite_host?: boolean;
              }
            | {
                type: "string";
                status_code?: number;
                headers?: HTTPHeader;
                content?: string;
              }
          );
      bbr_profile?: "standard" | "conservative" | "aggressive";
      brutal_debug?: boolean;
      realm?: Hysteria2InboundRealm;
    }
  | {
      type: "mixed";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      users?: User[];
      domain_resolver?: DomainResolver;
      set_system_proxy?: boolean;
      tls?: InboundTLSOptions;
    }
  | {
      type: "naive";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      users?: User[];
      network?: ("tcp" | "udp") | ("tcp" | "udp")[];
      quic_congestion_control?: "bbr" | "bbr_standard" | "bbr2" | "bbr2_variant" | "cubic" | "reno";
      tls?: InboundTLSOptions;
    }
  | {
      type: "redirect";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
    }
  | {
      type: "shadowsocks";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      network?: ("tcp" | "udp") | ("tcp" | "udp")[];
      method?:
        | "none"
        | "aes-128-gcm"
        | "aes-192-gcm"
        | "aes-256-gcm"
        | "chacha20-ietf-poly1305"
        | "xchacha20-ietf-poly1305"
        | "2022-blake3-aes-128-gcm"
        | "2022-blake3-aes-256-gcm"
        | "2022-blake3-chacha20-poly1305";
      password?: string;
      users?: ShadowsocksUser[];
      destinations?: ShadowsocksDestination[];
      multiplex?: InboundMultiplexOptions;
      managed?: boolean;
    }
  | {
      type: "shadowtls";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      version?: 1 | 2 | 3;
      password?: string;
      users?: ShadowTLSUser[];
      handshake?: ShadowTLSHandshakeOptions;
      handshake_for_server_name?: {
        [k: string]: ShadowTLSHandshakeOptions;
      };
      strict_mode?: boolean;
      wildcard_sni?: "" | "off" | "authed" | "all";
    }
  | (
      | {
          type: "snell";
          tag?: string;
          version: 5;
          listen?: string;
          listen_port?: number;
          bind_interface?: string;
          routing_mark?: number | string;
          reuse_addr?: boolean;
          netns?: string;
          disable_tcp_keep_alive?: boolean;
          tcp_keep_alive?: Duration;
          tcp_keep_alive_interval?: Duration;
          tcp_fast_open?: boolean;
          tcp_multi_path?: boolean;
          udp_fragment?: boolean;
          udp_timeout?: UDPTimeout;
          detour?: string;
          psk?: string;
          users?: SnellUser[];
          obfs_mode?: "none" | "http" | "tls";
        }
      | {
          type: "snell";
          tag?: string;
          version: 6;
          listen?: string;
          listen_port?: number;
          bind_interface?: string;
          routing_mark?: number | string;
          reuse_addr?: boolean;
          netns?: string;
          disable_tcp_keep_alive?: boolean;
          tcp_keep_alive?: Duration;
          tcp_keep_alive_interval?: Duration;
          tcp_fast_open?: boolean;
          tcp_multi_path?: boolean;
          udp_fragment?: boolean;
          udp_timeout?: UDPTimeout;
          detour?: string;
          psk?: string;
          users?: SnellUser[];
          mode?: "default" | "unshaped" | "unsafe-raw";
        }
    )
  | {
      type: "socks";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      users?: User[];
      domain_resolver?: DomainResolver;
    }
  | {
      type: "tproxy";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      network?: ("tcp" | "udp") | ("tcp" | "udp")[];
      udp_mapping?: "" | "endpoint_independent" | "address_dependent" | "address_and_port_dependent";
      udp_filtering?: "" | "endpoint_independent" | "address_dependent" | "address_and_port_dependent";
      udp_nat_max?: number;
    }
  | {
      type: "trojan";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      users?: TrojanUser[];
      tls?: InboundTLSOptions;
      fallback?: ServerOptions;
      fallback_for_alpn?: {
        [k: string]: ServerOptions;
      };
      multiplex?: InboundMultiplexOptions;
      transport?: V2RayTransport;
    }
  | {
      type: "tuic";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      users?: TUICUser[];
      congestion_control?: "cubic" | "new_reno" | "bbr";
      auth_timeout?: Duration;
      zero_rtt_handshake?: boolean;
      heartbeat?: Duration;
      tls?: InboundTLSOptions;
      idle_timeout?: Duration;
      keep_alive_period?: Duration;
      stream_receive_window?: number | string;
      connection_receive_window?: number | string;
      max_concurrent_streams?: number;
      initial_packet_size?: number;
      disable_path_mtu_discovery?: boolean;
    }
  | {
      type: "tun";
      tag?: string;
      interface_name?: string;
      netns?: string;
      mtu?: number;
      address?: string | string[];
      dns_mode?: "disabled" | "native" | "hijack";
      dns_address?: string | string[];
      auto_route?: boolean;
      iproute2_table_index?: number;
      iproute2_rule_index?: number;
      auto_redirect?: boolean;
      auto_redirect_input_mark?: number | string;
      auto_redirect_output_mark?: number | string;
      auto_redirect_reset_mark?: number | string;
      auto_redirect_nfqueue?: number;
      auto_redirect_iproute2_fallback_rule_index?: number;
      exclude_mptcp?: boolean;
      loopback_address?: string | string[];
      strict_route?: boolean;
      route_address?: string | string[];
      route_address_set?: string | string[];
      route_exclude_address?: string | string[];
      route_exclude_address_set?: string | string[];
      include_interface?: string | string[];
      exclude_interface?: string | string[];
      include_uid?: number | number[];
      include_uid_range?: string | string[];
      exclude_uid?: number | number[];
      exclude_uid_range?: string | string[];
      include_android_user?: number | number[];
      include_package?: string | string[];
      exclude_package?: string | string[];
      include_mac_address?: string | string[];
      exclude_mac_address?: string | string[];
      udp_timeout?: UDPTimeout;
      udp_mapping?: "" | "endpoint_independent" | "address_dependent" | "address_and_port_dependent";
      udp_filtering?: "" | "endpoint_independent" | "address_dependent" | "address_and_port_dependent";
      udp_nat_max?: number;
      stack?: "system" | "gvisor" | "mixed";
      platform?: TunPlatformOptions;
    }
  | {
      type: "vless";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      users?: VLESSUser[];
      tls?: InboundTLSOptions;
      multiplex?: InboundMultiplexOptions;
      transport?: V2RayTransport;
    }
  | {
      type: "vmess";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      users?: VMessUser[];
      tls?: InboundTLSOptions;
      multiplex?: InboundMultiplexOptions;
      transport?: V2RayTransport;
    };
export type CertificateProviderReference =
  | string
  | (
      | {
          type: "acme";
          domain?: string | string[];
          data_directory?: string;
          default_server_name?: string;
          email?: string;
          provider?: string;
          account_key?: string;
          disable_http_challenge?: boolean;
          disable_tls_alpn_challenge?: boolean;
          alternative_http_port?: number;
          alternative_tls_port?: number;
          external_account?: ACMEExternalAccountOptions;
          dns01_challenge?: ACMEProviderDNS01Challenge;
          key_type?: "ed25519" | "p256" | "p384" | "rsa2048" | "rsa4096";
          profile?: string;
          http_client?: HTTPClientReference;
        }
      | {
          type: "cloudflare-origin-ca";
          domain?: string | string[];
          data_directory?: string;
          api_token?: string;
          origin_ca_key?: string;
          request_type?: "origin-rsa" | "origin-ecc";
          requested_validity?: 0 | 7 | 30 | 90 | 365 | 730 | 1095 | 5475;
          http_client?: HTTPClientReference;
        }
      | {
          type: "tailscale";
          endpoint?: string;
        }
    );
export type V2RayTransport =
  | {
      type: "http";
      host?: string | string[];
      path?: string;
      method?: string;
      headers?: HTTPHeader;
      idle_timeout?: Duration;
      ping_timeout?: Duration;
    }
  | {
      type: "ws";
      path?: string;
      headers?: HTTPHeader;
      max_early_data?: number;
      early_data_header_name?: string;
    }
  | {
      type: "quic";
    }
  | {
      type: "grpc";
      service_name?: string;
      idle_timeout?: Duration;
      ping_timeout?: Duration;
      permit_without_stream?: boolean;
    }
  | {
      type: "httpupgrade";
      host?: string;
      path?: string;
      headers?: HTTPHeader;
    };
export type Outbound =
  | {
      type: "anytls";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
      tls?: OutboundTLSOptions;
      password?: string;
      idle_session_check_interval?: Duration;
      idle_session_timeout?: Duration;
      min_idle_session?: number;
      client_metadata?: string;
    }
  | {
      type: "block";
      tag?: string;
    }
  | {
      type: "bridge";
      tag?: string;
      interface?: string;
      bridge_name?: string;
      iproute2_table_index?: number;
      iproute2_rule_index?: number;
    }
  | {
      type: "direct";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
    }
  | {
      type: "http";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
      username?: string;
      password?: string;
      tls?: OutboundTLSOptions;
      path?: string;
      headers?: HTTPHeader;
    }
  | {
      type: "hysteria";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
      server_ports?: string | string[];
      hop_interval?: Duration;
      up?: number | string;
      up_mbps?: number;
      down?: number | string;
      down_mbps?: number;
      obfs?: string;
      auth?: string | number[];
      auth_str?: string;
      network?: ("tcp" | "udp") | ("tcp" | "udp")[];
      tls?: OutboundTLSOptions;
      idle_timeout?: Duration;
      keep_alive_period?: Duration;
      stream_receive_window?: number | string;
      connection_receive_window?: number | string;
      max_concurrent_streams?: number;
      initial_packet_size?: number;
      disable_path_mtu_discovery?: boolean;
    }
  | {
      type: "hysteria2";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
      server_ports?: string | string[];
      hop_interval?: Duration;
      hop_interval_max?: Duration;
      up_mbps?: number;
      down_mbps?: number;
      obfs?:
        | {
            type: "salamander";
            password?: string;
          }
        | {
            type: "gecko";
            password?: string;
            min_packet_size?: number;
            max_packet_size?: number;
          };
      password?: string;
      network?: ("tcp" | "udp") | ("tcp" | "udp")[];
      tls?: OutboundTLSOptions;
      idle_timeout?: Duration;
      keep_alive_period?: Duration;
      stream_receive_window?: number | string;
      connection_receive_window?: number | string;
      max_concurrent_streams?: number;
      initial_packet_size?: number;
      disable_path_mtu_discovery?: boolean;
      bbr_profile?: "standard" | "conservative" | "aggressive";
      brutal_debug?: boolean;
      disable_chrome_parrot?: boolean;
      realm?: Hysteria2Realm;
    }
  | {
      type: "naive";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
      username?: string;
      password?: string;
      insecure_concurrency?: number;
      extra_headers?: HTTPHeader;
      stream_receive_window?: number | string;
      udp_over_tcp?:
        | boolean
        | {
            enabled?: boolean;
            version?: 1 | 2;
          };
      quic?: boolean;
      quic_congestion_control?: "bbr" | "bbr2" | "cubic" | "reno";
      quic_session_receive_window?: number | string;
      tls?: OutboundTLSOptions;
    }
  | {
      type: "selector";
      tag?: string;
      outbounds?: string[];
      default?: string;
      interrupt_exist_connections?: boolean;
    }
  | {
      type: "shadowsocks";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
      method?:
        | "none"
        | "aes-128-gcm"
        | "aes-192-gcm"
        | "aes-256-gcm"
        | "chacha20-ietf-poly1305"
        | "xchacha20-ietf-poly1305"
        | "2022-blake3-aes-128-gcm"
        | "2022-blake3-aes-256-gcm"
        | "2022-blake3-chacha20-poly1305"
        | "aes-128-ctr"
        | "aes-192-ctr"
        | "aes-256-ctr"
        | "aes-128-cfb"
        | "aes-192-cfb"
        | "aes-256-cfb"
        | "rc4-md5"
        | "chacha20-ietf"
        | "xchacha20";
      password?: string;
      plugin?: string;
      plugin_opts?: string;
      network?: ("tcp" | "udp") | ("tcp" | "udp")[];
      udp_over_tcp?:
        | boolean
        | {
            enabled?: boolean;
            version?: 1 | 2;
          };
      multiplex?: OutboundMultiplexOptions;
    }
  | {
      type: "shadowtls";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
      version?: 1 | 2 | 3;
      password?: string;
      tls?: OutboundTLSOptions;
    }
  | (
      | {
          type: "snell";
          tag?: string;
          version: 4;
          detour?: string;
          bind_interface?: string;
          inet4_bind_address?: string;
          inet6_bind_address?: string;
          bind_address_no_port?: boolean;
          protect_path?: string;
          routing_mark?: number | string;
          reuse_addr?: boolean;
          netns?: string;
          connect_timeout?: Duration;
          tcp_fast_open?: boolean;
          tcp_multi_path?: boolean;
          disable_tcp_keep_alive?: boolean;
          tcp_keep_alive?: Duration;
          tcp_keep_alive_interval?: Duration;
          udp_fragment?: boolean;
          domain_resolver?: DomainResolver;
          network_strategy?: "default" | "fallback" | "hybrid";
          network_type?: InterfaceType | InterfaceType[];
          fallback_network_type?: InterfaceType | InterfaceType[];
          fallback_delay?: Duration;
          server?: string;
          server_port?: number;
          psk?: string;
          userkey?: string;
          reuse?: boolean;
          network?: ("tcp" | "udp") | ("tcp" | "udp")[];
          obfs_mode?: "none" | "http" | "tls";
          obfs_host?: string;
        }
      | {
          type: "snell";
          tag?: string;
          version: 6;
          detour?: string;
          bind_interface?: string;
          inet4_bind_address?: string;
          inet6_bind_address?: string;
          bind_address_no_port?: boolean;
          protect_path?: string;
          routing_mark?: number | string;
          reuse_addr?: boolean;
          netns?: string;
          connect_timeout?: Duration;
          tcp_fast_open?: boolean;
          tcp_multi_path?: boolean;
          disable_tcp_keep_alive?: boolean;
          tcp_keep_alive?: Duration;
          tcp_keep_alive_interval?: Duration;
          udp_fragment?: boolean;
          domain_resolver?: DomainResolver;
          network_strategy?: "default" | "fallback" | "hybrid";
          network_type?: InterfaceType | InterfaceType[];
          fallback_network_type?: InterfaceType | InterfaceType[];
          fallback_delay?: Duration;
          server?: string;
          server_port?: number;
          psk?: string;
          userkey?: string;
          reuse?: boolean;
          network?: ("tcp" | "udp") | ("tcp" | "udp")[];
          mode?: "default" | "unshaped" | "unsafe-raw";
        }
    )
  | {
      type: "socks";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
      version?: "4" | "4a" | "5";
      username?: string;
      password?: string;
      network?: ("tcp" | "udp") | ("tcp" | "udp")[];
      udp_over_tcp?:
        | boolean
        | {
            enabled?: boolean;
            version?: 1 | 2;
          };
    }
  | {
      type: "ssh";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
      user?: string;
      password?: string;
      private_key?: string | string[];
      private_key_path?: string;
      private_key_passphrase?: string;
      host_key?: string | string[];
      host_key_algorithms?: string | string[];
      client_version?: string;
      cipher?: string | string[];
      mac?: string | string[];
      kex_algorithm?: string | string[];
    }
  | {
      type: "tor";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      executable_path?: string;
      extra_args?: string[];
      data_directory?: string;
      torrc?: {
        [k: string]: string;
      };
    }
  | {
      type: "trojan";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
      password?: string;
      network?: ("tcp" | "udp") | ("tcp" | "udp")[];
      tls?: OutboundTLSOptions;
      multiplex?: OutboundMultiplexOptions;
      transport?: V2RayTransport;
    }
  | {
      type: "tuic";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
      uuid?: string;
      password?: string;
      congestion_control?: "cubic" | "new_reno" | "bbr";
      udp_relay_mode?: "native" | "quic";
      udp_over_stream?: boolean;
      zero_rtt_handshake?: boolean;
      heartbeat?: Duration;
      network?: ("tcp" | "udp") | ("tcp" | "udp")[];
      tls?: OutboundTLSOptions;
      idle_timeout?: Duration;
      keep_alive_period?: Duration;
      stream_receive_window?: number | string;
      connection_receive_window?: number | string;
      max_concurrent_streams?: number;
      initial_packet_size?: number;
      disable_path_mtu_discovery?: boolean;
    }
  | {
      type: "urltest";
      tag?: string;
      outbounds?: string[];
      url?: string;
      interval?: Duration;
      tolerance?: number;
      idle_timeout?: Duration;
      interrupt_exist_connections?: boolean;
    }
  | {
      type: "vless";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
      uuid?: string;
      flow?: string;
      network?: ("tcp" | "udp") | ("tcp" | "udp")[];
      tls?: OutboundTLSOptions;
      multiplex?: OutboundMultiplexOptions;
      transport?: V2RayTransport;
      packet_encoding?: string;
    }
  | {
      type: "vmess";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
      uuid?: string;
      security?: "auto" | "none" | "zero" | "aes-128-cfb" | "aes-128-gcm" | "chacha20-poly1305";
      alter_id?: number;
      global_padding?: boolean;
      authenticated_length?: boolean;
      network?: ("tcp" | "udp") | ("tcp" | "udp")[];
      tls?: OutboundTLSOptions;
      packet_encoding?: "packetaddr" | "xudp";
      multiplex?: OutboundMultiplexOptions;
      transport?: V2RayTransport;
    };
export type Rule =
  | ({
      type?: "default" | "";
      inbound?: string | string[];
      ip_version?: 4 | 6;
      network?: ("tcp" | "udp" | "icmp") | ("tcp" | "udp" | "icmp")[];
      auth_user?: string | string[];
      protocol?:
        | ("tls" | "http" | "quic" | "dns" | "stun" | "bittorrent" | "dtls" | "ssh" | "rdp" | "ntp")
        | ("tls" | "http" | "quic" | "dns" | "stun" | "bittorrent" | "dtls" | "ssh" | "rdp" | "ntp")[];
      client?: string | string[];
      domain?: string | string[];
      domain_suffix?: string | string[];
      domain_keyword?: string | string[];
      domain_regex?: string | string[];
      source_ip_cidr?: string | string[];
      source_ip_is_private?: boolean;
      ip_cidr?: string | string[];
      ip_is_private?: boolean;
      source_port?: number | number[];
      source_port_range?: string | string[];
      port?: number | number[];
      port_range?: string | string[];
      process_name?: string | string[];
      process_path?: string | string[];
      process_path_regex?: string | string[];
      package_name?: string | string[];
      package_name_regex?: string | string[];
      user?: string | string[];
      user_id?: number | number[];
      clash_mode?: string;
      network_type?: InterfaceType | InterfaceType[];
      network_is_expensive?: boolean;
      network_is_constrained?: boolean;
      wifi_ssid?: string | string[];
      wifi_bssid?: string | string[];
      interface_address?: {
        [k: string]: string | string[];
      };
      network_interface_address?: {
        [k: string]: string | string[];
      };
      default_interface_address?: string | string[];
      source_mac_address?: string | string[];
      source_hostname?: string | string[];
      preferred_by?: string | string[];
      rule_set?: string | string[];
      rule_set_ip_cidr_match_source?: boolean;
      invert?: boolean;
      [k: string]: unknown;
    } & RuleAction)
  | ({
      type: "logical";
      mode: "and" | "or";
      rules: NestedRule[];
      invert?: boolean;
      [k: string]: unknown;
    } & RuleAction);
export type RuleAction =
  | {
      action?: "route";
      outbound?: string;
      override_address?: string;
      override_port?: number;
      network_strategy?: "default" | "fallback" | "hybrid";
      fallback_delay?: number;
      udp_disable_domain_unmapping?: boolean;
      udp_connect?: boolean;
      udp_timeout?: Duration;
      tls_fragment?: boolean;
      tls_fragment_fallback_delay?: Duration;
      tls_record_fragment?: boolean;
      tls_spoof?: string;
      tls_spoof_method?: "wrong-sequence" | "wrong-checksum" | "wrong-ack" | "wrong-md5" | "wrong-timestamp";
      [k: string]: unknown;
    }
  | {
      action: "route-options";
      override_address?: string;
      override_port?: number;
      network_strategy?: "default" | "fallback" | "hybrid";
      fallback_delay?: number;
      udp_disable_domain_unmapping?: boolean;
      udp_connect?: boolean;
      udp_timeout?: Duration;
      tls_fragment?: boolean;
      tls_fragment_fallback_delay?: Duration;
      tls_record_fragment?: boolean;
      tls_spoof?: string;
      tls_spoof_method?: "wrong-sequence" | "wrong-checksum" | "wrong-ack" | "wrong-md5" | "wrong-timestamp";
      [k: string]: unknown;
    }
  | {
      action: "direct";
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      [k: string]: unknown;
    }
  | {
      action: "bypass";
      outbound?: string;
      override_address?: string;
      override_port?: number;
      network_strategy?: "default" | "fallback" | "hybrid";
      fallback_delay?: number;
      udp_disable_domain_unmapping?: boolean;
      udp_connect?: boolean;
      udp_timeout?: Duration;
      tls_fragment?: boolean;
      tls_fragment_fallback_delay?: Duration;
      tls_record_fragment?: boolean;
      tls_spoof?: string;
      tls_spoof_method?: "wrong-sequence" | "wrong-checksum" | "wrong-ack" | "wrong-md5" | "wrong-timestamp";
      [k: string]: unknown;
    }
  | {
      action: "reject";
      method?: "" | "default" | "drop" | "reply";
      no_drop?: boolean;
      [k: string]: unknown;
    }
  | {
      action: "hijack-dns";
      [k: string]: unknown;
    }
  | {
      action: "sniff";
      sniffer?:
        | ("tls" | "http" | "quic" | "dns" | "stun" | "bittorrent" | "dtls" | "ssh" | "rdp" | "ntp")
        | ("tls" | "http" | "quic" | "dns" | "stun" | "bittorrent" | "dtls" | "ssh" | "rdp" | "ntp")[];
      timeout?: Duration;
      [k: string]: unknown;
    }
  | {
      action: "resolve";
      server?: string;
      timeout?: Duration;
      strategy?: DomainStrategy;
      disable_cache?: boolean;
      disable_optimistic_cache?: boolean;
      rewrite_ttl?: number;
      client_subnet?: string;
      [k: string]: unknown;
    };
export type NestedRule =
  | {
      type?: "default" | "";
      inbound?: string | string[];
      ip_version?: 4 | 6;
      network?: ("tcp" | "udp" | "icmp") | ("tcp" | "udp" | "icmp")[];
      auth_user?: string | string[];
      protocol?:
        | ("tls" | "http" | "quic" | "dns" | "stun" | "bittorrent" | "dtls" | "ssh" | "rdp" | "ntp")
        | ("tls" | "http" | "quic" | "dns" | "stun" | "bittorrent" | "dtls" | "ssh" | "rdp" | "ntp")[];
      client?: string | string[];
      domain?: string | string[];
      domain_suffix?: string | string[];
      domain_keyword?: string | string[];
      domain_regex?: string | string[];
      source_ip_cidr?: string | string[];
      source_ip_is_private?: boolean;
      ip_cidr?: string | string[];
      ip_is_private?: boolean;
      source_port?: number | number[];
      source_port_range?: string | string[];
      port?: number | number[];
      port_range?: string | string[];
      process_name?: string | string[];
      process_path?: string | string[];
      process_path_regex?: string | string[];
      package_name?: string | string[];
      package_name_regex?: string | string[];
      user?: string | string[];
      user_id?: number | number[];
      clash_mode?: string;
      network_type?: InterfaceType | InterfaceType[];
      network_is_expensive?: boolean;
      network_is_constrained?: boolean;
      wifi_ssid?: string | string[];
      wifi_bssid?: string | string[];
      interface_address?: {
        [k: string]: string | string[];
      };
      network_interface_address?: {
        [k: string]: string | string[];
      };
      default_interface_address?: string | string[];
      source_mac_address?: string | string[];
      source_hostname?: string | string[];
      preferred_by?: string | string[];
      rule_set?: string | string[];
      rule_set_ip_cidr_match_source?: boolean;
      invert?: boolean;
    }
  | {
      type: "logical";
      mode: "and" | "or";
      rules: NestedRule[];
      invert?: boolean;
    };
export type RuleSet =
  | {
      type?: "inline" | "";
      tag: string | string[];
      rules?: HeadlessRule[];
    }
  | {
      type: "local";
      tag: string | string[];
      format?: "source" | "binary";
      path?: string;
    }
  | {
      type: "remote";
      tag: string | string[];
      format?: "source" | "binary";
      url?: string;
      initial_path?: string;
      http_client?: HTTPClientReference;
      update_interval?: Duration;
    };
export type HeadlessRule =
  | {
      type?: "default" | "";
      query_type?: DNSQueryType | DNSQueryType[];
      network?: ("tcp" | "udp" | "icmp") | ("tcp" | "udp" | "icmp")[];
      domain?: string | string[];
      domain_suffix?: string | string[];
      domain_keyword?: string | string[];
      domain_regex?: string | string[];
      source_ip_cidr?: string | string[];
      ip_cidr?: string | string[];
      source_port?: number | number[];
      source_port_range?: string | string[];
      port?: number | number[];
      port_range?: string | string[];
      process_name?: string | string[];
      process_path?: string | string[];
      process_path_regex?: string | string[];
      package_name?: string | string[];
      package_name_regex?: string | string[];
      network_type?: InterfaceType | InterfaceType[];
      network_is_expensive?: boolean;
      network_is_constrained?: boolean;
      wifi_ssid?: string | string[];
      wifi_bssid?: string | string[];
      network_interface_address?: {
        [k: string]: string | string[];
      };
      default_interface_address?: string | string[];
      invert?: boolean;
    }
  | {
      type: "logical";
      mode: "and" | "or";
      rules: HeadlessRule[];
      invert?: boolean;
    };
export type Service =
  | {
      type: "api";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      secret?: string;
      access_control_allow_origin?: string | string[];
      access_control_allow_private_network?: boolean;
      dashboard?:
        | boolean
        | string
        | {
            enabled?: boolean;
            path?: string;
            download_url?: string;
            http_client?: HTTPClientReference;
            update_interval?: Duration;
          };
      tls?: InboundTLSOptions;
    }
  | {
      type: "ccm";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      tls?: InboundTLSOptions;
      credential_path?: string;
      users?: CCMUser[];
      headers?: HTTPHeader;
      usages_path?: string;
    }
  | {
      type: "derp";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      tls?: InboundTLSOptions;
      config_path?: string;
      verify_client_endpoint?: string | string[];
      verify_client_url?:
        | (
            | string
            | {
                tag?: string;
                engine?: "go" | "apple";
                version?: 0 | 1 | 2 | 3;
                disable_version_fallback?: boolean;
                headers?: HTTPHeader;
                tls?: OutboundTLSOptions;
                detour?: string;
                bind_interface?: string;
                inet4_bind_address?: string;
                inet6_bind_address?: string;
                bind_address_no_port?: boolean;
                protect_path?: string;
                routing_mark?: number | string;
                reuse_addr?: boolean;
                netns?: string;
                connect_timeout?: Duration;
                tcp_fast_open?: boolean;
                tcp_multi_path?: boolean;
                disable_tcp_keep_alive?: boolean;
                tcp_keep_alive?: Duration;
                tcp_keep_alive_interval?: Duration;
                udp_fragment?: boolean;
                domain_resolver?: DomainResolver;
                network_strategy?: "default" | "fallback" | "hybrid";
                network_type?: InterfaceType | InterfaceType[];
                fallback_network_type?: InterfaceType | InterfaceType[];
                fallback_delay?: Duration;
                idle_timeout?: Duration;
                keep_alive_period?: Duration;
                stream_receive_window?: number | string;
                connection_receive_window?: number | string;
                max_concurrent_streams?: number;
                initial_packet_size?: number;
                disable_path_mtu_discovery?: boolean;
                url?: string;
              }
          )
        | (
            | string
            | {
                tag?: string;
                engine?: "go" | "apple";
                version?: 0 | 1 | 2 | 3;
                disable_version_fallback?: boolean;
                headers?: HTTPHeader;
                tls?: OutboundTLSOptions;
                detour?: string;
                bind_interface?: string;
                inet4_bind_address?: string;
                inet6_bind_address?: string;
                bind_address_no_port?: boolean;
                protect_path?: string;
                routing_mark?: number | string;
                reuse_addr?: boolean;
                netns?: string;
                connect_timeout?: Duration;
                tcp_fast_open?: boolean;
                tcp_multi_path?: boolean;
                disable_tcp_keep_alive?: boolean;
                tcp_keep_alive?: Duration;
                tcp_keep_alive_interval?: Duration;
                udp_fragment?: boolean;
                domain_resolver?: DomainResolver;
                network_strategy?: "default" | "fallback" | "hybrid";
                network_type?: InterfaceType | InterfaceType[];
                fallback_network_type?: InterfaceType | InterfaceType[];
                fallback_delay?: Duration;
                idle_timeout?: Duration;
                keep_alive_period?: Duration;
                stream_receive_window?: number | string;
                connection_receive_window?: number | string;
                max_concurrent_streams?: number;
                initial_packet_size?: number;
                disable_path_mtu_discovery?: boolean;
                url?: string;
              }
          )[];
      home?: string;
      mesh_with?: DERPMeshOptions | DERPMeshOptions[];
      mesh_psk?: string;
      mesh_psk_file?: string;
      stun?:
        | number
        | {
            enabled?: boolean;
            listen?: string;
            listen_port?: number;
            bind_interface?: string;
            routing_mark?: number | string;
            reuse_addr?: boolean;
            netns?: string;
            disable_tcp_keep_alive?: boolean;
            tcp_keep_alive?: Duration;
            tcp_keep_alive_interval?: Duration;
            tcp_fast_open?: boolean;
            tcp_multi_path?: boolean;
            udp_fragment?: boolean;
            udp_timeout?: UDPTimeout;
            detour?: string;
          };
    }
  | {
      type: "hysteria-realm";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      tls?: InboundTLSOptions;
      idle_timeout?: Duration;
      keep_alive_period?: Duration;
      stream_receive_window?: number | string;
      connection_receive_window?: number | string;
      max_concurrent_streams?: number;
      users?: HysteriaRealmUser[];
    }
  | {
      type: "ocm";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      tls?: InboundTLSOptions;
      credential_path?: string;
      users?: OCMUser[];
      headers?: HTTPHeader;
      usages_path?: string;
    }
  | {
      type: "oom-killer";
      tag?: string;
      memory_limit?: number | string;
      safety_margin?: number | string;
      min_interval?: Duration;
      max_interval?: Duration;
    }
  | {
      type: "resolved";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
    }
  | {
      type: "ssm-api";
      tag?: string;
      listen?: string;
      listen_port?: number;
      bind_interface?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      udp_fragment?: boolean;
      udp_timeout?: UDPTimeout;
      detour?: string;
      servers?: {
        [k: string]: string;
      };
      cache_path?: string;
      tls?: InboundTLSOptions;
    }
  | {
      type: "usbip-client";
      tag?: string;
      detour?: string;
      bind_interface?: string;
      inet4_bind_address?: string;
      inet6_bind_address?: string;
      bind_address_no_port?: boolean;
      protect_path?: string;
      routing_mark?: number | string;
      reuse_addr?: boolean;
      netns?: string;
      connect_timeout?: Duration;
      tcp_fast_open?: boolean;
      tcp_multi_path?: boolean;
      disable_tcp_keep_alive?: boolean;
      tcp_keep_alive?: Duration;
      tcp_keep_alive_interval?: Duration;
      udp_fragment?: boolean;
      domain_resolver?: DomainResolver;
      network_strategy?: "default" | "fallback" | "hybrid";
      network_type?: InterfaceType | InterfaceType[];
      fallback_network_type?: InterfaceType | InterfaceType[];
      fallback_delay?: Duration;
      server?: string;
      server_port?: number;
      devices?: USBIPDeviceMatch[];
    }
  | (
      | {
          type: "usbip-server";
          tag?: string;
          provider?: "default" | "";
          listen?: string;
          listen_port?: number;
          bind_interface?: string;
          routing_mark?: number | string;
          reuse_addr?: boolean;
          netns?: string;
          disable_tcp_keep_alive?: boolean;
          tcp_keep_alive?: Duration;
          tcp_keep_alive_interval?: Duration;
          tcp_fast_open?: boolean;
          tcp_multi_path?: boolean;
          udp_fragment?: boolean;
          udp_timeout?: UDPTimeout;
          detour?: string;
          devices?: USBIPDeviceMatch[];
        }
      | {
          type: "usbip-server";
          tag?: string;
          provider: "dynamic";
          listen?: string;
          listen_port?: number;
          bind_interface?: string;
          routing_mark?: number | string;
          reuse_addr?: boolean;
          netns?: string;
          disable_tcp_keep_alive?: boolean;
          tcp_keep_alive?: Duration;
          tcp_keep_alive_interval?: Duration;
          tcp_fast_open?: boolean;
          tcp_multi_path?: boolean;
          udp_fragment?: boolean;
          udp_timeout?: UDPTimeout;
          detour?: string;
        }
    );

export interface HttpsSingBoxSagernetOrgSchemaJson {
  $schema?: string;
  log?: LogOptions;
  dns?: DNS;
  ntp?: NTPOptions;
  certificate?: {
    store?: "system" | "mozilla" | "chrome" | "none";
    certificate?: string | string[];
    certificate_path?: string | string[];
    certificate_directory_path?: string | string[];
  };
  certificate_providers?: CertificateProvider[];
  http_clients?: HTTPClient[];
  network_namespaces?: NetworkNamespace[];
  endpoints?: Endpoint[];
  inbounds?: Inbound[];
  outbounds?: Outbound[];
  route?: RouteOptions;
  services?: Service[];
  experimental?: ExperimentalOptions;
}
export interface LogOptions {
  disabled?: boolean;
  level?: "trace" | "debug" | "info" | "warn" | "warning" | "error" | "fatal" | "panic";
  output?: string;
  timestamp?: boolean;
}
export interface DNS {
  servers?: DNSServer[];
  rules?: DNSRule[];
  final?: string;
  reverse_mapping?: boolean;
  strategy?: DomainStrategy;
  timeout?: Duration;
  disable_cache?: boolean;
  disable_expire?: boolean;
  cache_capacity?: number;
  optimistic?:
    | boolean
    | {
        enabled?: boolean;
        timeout?: Duration;
      };
  client_subnet?: string;
}
export interface OutboundTLSOptions {
  enabled?: boolean;
  engine?: "go" | "apple" | "windows";
  disable_sni?: boolean;
  server_name?: string;
  insecure?: boolean;
  alpn?: string | string[];
  min_version?: "1.0" | "1.1" | "1.2" | "1.3";
  max_version?: "1.0" | "1.1" | "1.2" | "1.3";
  cipher_suites?: string | string[];
  curve_preferences?:
    | ("P256" | "P384" | "P521" | "X25519" | "X25519MLKEM768")
    | ("P256" | "P384" | "P521" | "X25519" | "X25519MLKEM768")[];
  certificate?: string | string[];
  certificate_path?: string;
  certificate_public_key_sha256?: (string | number[]) | (string | number[])[];
  client_certificate?: string | string[];
  client_certificate_path?: string;
  client_key?: string | string[];
  client_key_path?: string;
  fragment?: boolean;
  fragment_fallback_delay?: Duration;
  record_fragment?: boolean;
  spoof?: string;
  spoof_method?: "wrong-sequence" | "wrong-checksum" | "wrong-ack" | "wrong-md5" | "wrong-timestamp";
  kernel_tx?: boolean;
  kernel_rx?: boolean;
  handshake_timeout?: Duration;
  ech?: OutboundECHOptions;
  utls?: OutboundUTLSOptions;
  reality?: OutboundRealityOptions;
}
export interface OutboundECHOptions {
  enabled?: boolean;
  config?: string | string[];
  config_path?: string;
  query_server_name?: string;
}
export interface OutboundUTLSOptions {
  enabled?: boolean;
  fingerprint?:
    | "chrome_psk"
    | "chrome_psk_shuffle"
    | "chrome_padding_psk_shuffle"
    | "chrome_pq"
    | "chrome_pq_psk"
    | "chrome"
    | "firefox"
    | "edge"
    | "safari"
    | "360"
    | "qq"
    | "ios"
    | "android"
    | "random"
    | "randomized";
}
export interface OutboundRealityOptions {
  enabled?: boolean;
  public_key?: string;
  short_id?: string;
}
export interface HTTPHeader {
  [k: string]: string | string[];
}
export interface NTPOptions {
  enabled?: boolean;
  interval?: Duration;
  write_to_system?: boolean;
  server?: string;
  server_port?: number;
  detour?: string;
  bind_interface?: string;
  inet4_bind_address?: string;
  inet6_bind_address?: string;
  bind_address_no_port?: boolean;
  protect_path?: string;
  routing_mark?: number | string;
  reuse_addr?: boolean;
  netns?: string;
  connect_timeout?: Duration;
  tcp_fast_open?: boolean;
  tcp_multi_path?: boolean;
  disable_tcp_keep_alive?: boolean;
  tcp_keep_alive?: Duration;
  tcp_keep_alive_interval?: Duration;
  udp_fragment?: boolean;
  domain_resolver?: DomainResolver;
  network_strategy?: "default" | "fallback" | "hybrid";
  network_type?: InterfaceType | InterfaceType[];
  fallback_network_type?: InterfaceType | InterfaceType[];
  fallback_delay?: Duration;
}
export interface ACMEExternalAccountOptions {
  key_id?: string;
  mac_key?: string;
}
export interface HTTPClient {
  tag?: string;
  engine?: "go" | "apple";
  version?: 0 | 1 | 2 | 3;
  disable_version_fallback?: boolean;
  headers?: HTTPHeader;
  tls?: OutboundTLSOptions;
  detour?: string;
  bind_interface?: string;
  inet4_bind_address?: string;
  inet6_bind_address?: string;
  bind_address_no_port?: boolean;
  protect_path?: string;
  routing_mark?: number | string;
  reuse_addr?: boolean;
  netns?: string;
  connect_timeout?: Duration;
  tcp_fast_open?: boolean;
  tcp_multi_path?: boolean;
  disable_tcp_keep_alive?: boolean;
  tcp_keep_alive?: Duration;
  tcp_keep_alive_interval?: Duration;
  udp_fragment?: boolean;
  domain_resolver?: DomainResolver;
  network_strategy?: "default" | "fallback" | "hybrid";
  network_type?: InterfaceType | InterfaceType[];
  fallback_network_type?: InterfaceType | InterfaceType[];
  fallback_delay?: Duration;
  idle_timeout?: Duration;
  keep_alive_period?: Duration;
  stream_receive_window?: number | string;
  connection_receive_window?: number | string;
  max_concurrent_streams?: number;
  initial_packet_size?: number;
  disable_path_mtu_discovery?: boolean;
}
export interface OpenConnectTokenOptions {
  mode?: "totp" | "hotp" | "stoken" | "oidc";
  secret?: string;
  secret_path?: string;
  pin?: string;
  password?: string;
  device_id?: string;
  counter?: number;
}
export interface OpenConnectMobileOptions {
  platform_version?: string;
  device_type?: string;
  device_unique_id?: string;
}
export interface OpenConnectCSDOptions {
  wrapper_path?: string;
}
export interface OpenConnectHIPOptions {
  wrapper_path?: string;
}
export interface OpenConnectTNCCOptions {
  wrapper_path?: string;
  device_id?: string;
  user_agent?: string;
  machine_identification_enabled?: boolean;
  certificates?: OpenConnectTNCCCertificateOptions[];
}
export interface OpenConnectTNCCCertificateOptions {
  certificate?: string | string[];
  certificate_path?: string;
}
export interface OpenConnectFortinetHostCheckOptions {
  hostcheck?: string;
  check_virtual_desktop?: string;
}
export interface OpenConnectTLSOptions {
  insecure?: boolean;
  server_name?: string;
  peer_fingerprint?: string | string[];
  system_trust_disabled?: boolean;
  certificate_authority?: string | string[];
  certificate_authority_path?: string;
  client_certificate?: string | string[];
  client_certificate_path?: string;
  client_key?: string | string[];
  client_key_path?: string;
  client_key_password?: string;
  mca_certificate?: string | string[];
  mca_certificate_path?: string;
  mca_key?: string | string[];
  mca_key_path?: string;
  mca_key_password?: string;
}
export interface OpenConnectFormEntryOptions {
  form_id?: string;
  submission_key?: string;
  name?: string;
  value?: string;
  promote?: boolean;
}
export interface OpenVPNRemoteOptions {
  server?: string;
  server_port?: number;
  network?: "udp" | "udp4" | "udp6" | "tcp" | "tcp4" | "tcp6";
}
export interface OpenVPNOutboundTLSOptions {
  server_name?: string;
  server_name_type?: "subject" | "name" | "name-prefix";
  certificate?: string | string[];
  certificate_path?: string;
  client_certificate?: string | string[];
  client_certificate_path?: string;
  client_key?: string | string[];
  client_key_path?: string;
  peer_fingerprint?: string | string[];
  crl_path?: string;
  remote_certificate_ku?: string | string[];
  remote_certificate_eku?: string;
  remote_certificate_tls?: "server" | "client" | "none";
  certificate_profile?: "legacy" | "preferred" | "insecure" | "suiteb";
  ns_certificate_type?: "server" | "client";
  version_min?: "1.0" | "1.1" | "1.2" | "1.3";
  version_max?: "1.0" | "1.1" | "1.2" | "1.3";
  cipher?: string;
  groups?: string;
  control_wrap?: OpenVPNControlWrapOptions;
}
export interface OpenVPNControlWrapOptions {
  type?: "tls_auth" | "tls_crypt" | "tls_crypt_v2";
  key?: string | string[];
  key_path?: string;
  direction?: "server" | "client";
}
export interface OpenVPNPullFilterOptions {
  action?: "accept" | "ignore" | "reject";
  text?: string;
}
export interface User {
  Username?: string;
  Password?: string;
}
export interface OpenVPNInboundTLSOptions {
  certificate?: string | string[];
  certificate_path?: string;
  key?: string | string[];
  key_path?: string;
  client_certificate?: string | string[];
  client_certificate_path?: string;
  verify_client_certificate?: "require" | "optional" | "none";
  client_name?: string;
  client_name_type?: "subject" | "name" | "name-prefix";
  peer_fingerprint?: string | string[];
  crl_path?: string;
  remote_certificate_ku?: string | string[];
  remote_certificate_eku?: string;
  remote_certificate_tls?: "server" | "client" | "none";
  certificate_profile?: "legacy" | "preferred" | "insecure" | "suiteb";
  ns_certificate_type?: "server" | "client";
  version_min?: "1.0" | "1.1" | "1.2" | "1.3";
  version_max?: "1.0" | "1.1" | "1.2" | "1.3";
  cipher?: string;
  groups?: string;
  control_wrap?: OpenVPNInboundControlWrapOptions;
}
export interface OpenVPNInboundControlWrapOptions {
  type?: "tls_auth" | "tls_crypt" | "tls_crypt_v2";
  key?: string | string[];
  key_path?: string;
  direction?: "server" | "client";
  force_cookie?: boolean;
}
export interface OpenVPNPushOptions {
  routes?: string | string[];
  dns?: string | string[];
  dns_servers?: OpenVPNPushDNSServerOptions[];
  search_domains?: string | string[];
  dhcp_options?: string | string[];
  redirect_gateway?: boolean;
  redirect_gateway_flags?: string | string[];
  block_outside_dns?: boolean;
  ping_interval?: Duration;
  ping_restart?: Duration;
}
export interface OpenVPNPushDNSServerOptions {
  priority?: number;
  addresses?: string | string[];
  resolve_domains?: string | string[];
  dnssec?: "yes" | "optional" | "no";
  transport?: "plain" | "dot" | "doh";
  sni?: string;
}
export interface WireGuardPeer {
  address?: string;
  port?: number;
  public_key?: string;
  pre_shared_key?: string;
  allowed_ips?: string | string[];
  persistent_keepalive_interval?: number;
  reserved?: string | number[];
}
export interface InboundTLSOptions {
  enabled?: boolean;
  server_name?: string;
  insecure?: boolean;
  alpn?: string | string[];
  min_version?: "1.0" | "1.1" | "1.2" | "1.3";
  max_version?: "1.0" | "1.1" | "1.2" | "1.3";
  cipher_suites?: string | string[];
  curve_preferences?:
    | ("P256" | "P384" | "P521" | "X25519" | "X25519MLKEM768")
    | ("P256" | "P384" | "P521" | "X25519" | "X25519MLKEM768")[];
  certificate?: string | string[];
  certificate_path?: string;
  client_authentication?: "no" | "request" | "require-any" | "verify-if-given" | "require-and-verify";
  client_certificate?: string | string[];
  client_certificate_path?: string | string[];
  client_certificate_public_key_sha256?: (string | number[]) | (string | number[])[];
  key?: string | string[];
  key_path?: string;
  kernel_tx?: boolean;
  kernel_rx?: boolean;
  handshake_timeout?: Duration;
  certificate_provider?: CertificateProviderReference;
  ech?: InboundECHOptions;
  reality?: InboundRealityOptions;
}
export interface InboundECHOptions {
  enabled?: boolean;
  key?: string | string[];
  key_path?: string;
}
export interface InboundRealityOptions {
  enabled?: boolean;
  handshake?: InboundRealityHandshakeOptions;
  private_key?: string;
  short_id?: string | string[];
  max_time_difference?: Duration;
}
export interface InboundRealityHandshakeOptions {
  server?: string;
  server_port?: number;
  detour?: string;
  bind_interface?: string;
  inet4_bind_address?: string;
  inet6_bind_address?: string;
  bind_address_no_port?: boolean;
  protect_path?: string;
  routing_mark?: number | string;
  reuse_addr?: boolean;
  netns?: string;
  connect_timeout?: Duration;
  tcp_fast_open?: boolean;
  tcp_multi_path?: boolean;
  disable_tcp_keep_alive?: boolean;
  tcp_keep_alive?: Duration;
  tcp_keep_alive_interval?: Duration;
  udp_fragment?: boolean;
  domain_resolver?: DomainResolver;
  network_strategy?: "default" | "fallback" | "hybrid";
  network_type?: InterfaceType | InterfaceType[];
  fallback_network_type?: InterfaceType | InterfaceType[];
  fallback_delay?: Duration;
}
export interface AnyTLSUser {
  name?: string;
  password?: string;
}
export interface DialerOptions {
  detour?: string;
  bind_interface?: string;
  inet4_bind_address?: string;
  inet6_bind_address?: string;
  bind_address_no_port?: boolean;
  protect_path?: string;
  routing_mark?: number | string;
  reuse_addr?: boolean;
  netns?: string;
  connect_timeout?: Duration;
  tcp_fast_open?: boolean;
  tcp_multi_path?: boolean;
  disable_tcp_keep_alive?: boolean;
  tcp_keep_alive?: Duration;
  tcp_keep_alive_interval?: Duration;
  udp_fragment?: boolean;
  domain_resolver?: DomainResolver;
  network_strategy?: "default" | "fallback" | "hybrid";
  network_type?: InterfaceType | InterfaceType[];
  fallback_network_type?: InterfaceType | InterfaceType[];
  fallback_delay?: Duration;
}
export interface HysteriaUser {
  name?: string;
  auth?: string | number[];
  auth_str?: string;
}
export interface Hysteria2User {
  name?: string;
  password?: string;
}
export interface Hysteria2InboundRealm {
  server_url?: string;
  token?: string;
  realm_id?: string;
  stun_servers?: string | string[];
  ip_version?: 0 | 4 | 6;
  port_mapping?: Hysteria2RealmPortMapping;
  http_client?: HTTPClientReference;
  stun_domain_resolver?: DomainResolver;
}
export interface Hysteria2RealmPortMapping {
  enabled?: boolean;
  timeout?: Duration;
  lifetime?: Duration;
}
export interface ShadowsocksUser {
  name?: string;
  password?: string;
}
export interface ShadowsocksDestination {
  name?: string;
  password?: string;
  server?: string;
  server_port?: number;
}
export interface InboundMultiplexOptions {
  enabled?: boolean;
  padding?: boolean;
  brutal?: BrutalOptions;
}
export interface BrutalOptions {
  enabled?: boolean;
  up_mbps?: number;
  down_mbps?: number;
}
export interface ShadowTLSUser {
  name?: string;
  password?: string;
}
export interface ShadowTLSHandshakeOptions {
  server?: string;
  server_port?: number;
  detour?: string;
  bind_interface?: string;
  inet4_bind_address?: string;
  inet6_bind_address?: string;
  bind_address_no_port?: boolean;
  protect_path?: string;
  routing_mark?: number | string;
  reuse_addr?: boolean;
  netns?: string;
  connect_timeout?: Duration;
  tcp_fast_open?: boolean;
  tcp_multi_path?: boolean;
  disable_tcp_keep_alive?: boolean;
  tcp_keep_alive?: Duration;
  tcp_keep_alive_interval?: Duration;
  udp_fragment?: boolean;
  domain_resolver?: DomainResolver;
  network_strategy?: "default" | "fallback" | "hybrid";
  network_type?: InterfaceType | InterfaceType[];
  fallback_network_type?: InterfaceType | InterfaceType[];
  fallback_delay?: Duration;
}
export interface SnellUser {
  name?: string;
  userkey?: string;
}
export interface TrojanUser {
  name?: string;
  password?: string;
}
export interface ServerOptions {
  server?: string;
  server_port?: number;
}
export interface TUICUser {
  name?: string;
  uuid?: string;
  password?: string;
}
export interface TunPlatformOptions {
  http_proxy?: HTTPProxyOptions;
}
export interface HTTPProxyOptions {
  enabled?: boolean;
  server?: string;
  server_port?: number;
  bypass_domain?: string | string[];
  match_domain?: string | string[];
}
export interface VLESSUser {
  name?: string;
  uuid?: string;
  flow?: string;
}
export interface VMessUser {
  name?: string;
  uuid?: string;
  alterId?: number;
}
export interface Hysteria2Realm {
  server_url?: string;
  token?: string;
  realm_id?: string;
  stun_servers?: string | string[];
  ip_version?: 0 | 4 | 6;
  port_mapping?: Hysteria2RealmPortMapping;
  http_client?: HTTPClientReference;
}
export interface OutboundMultiplexOptions {
  enabled?: boolean;
  protocol?: "h2mux" | "smux" | "yamux";
  max_connections?: number;
  min_streams?: number;
  max_streams?: number;
  padding?: boolean;
  brutal?: BrutalOptions;
}
export interface RouteOptions {
  rules?: Rule[];
  rule_set?: RuleSet[];
  final?: string;
  find_process?: boolean;
  find_neighbor?: boolean;
  dhcp_lease_files?: string | string[];
  auto_detect_interface?: boolean;
  override_android_vpn?: boolean;
  default_interface?: string;
  default_mark?: number | string;
  default_domain_resolver?: DomainResolver;
  default_network_strategy?: "default" | "fallback" | "hybrid";
  default_network_type?: InterfaceType | InterfaceType[];
  default_fallback_network_type?: InterfaceType | InterfaceType[];
  default_fallback_delay?: Duration;
  default_http_client?: string;
}
export interface CCMUser {
  name?: string;
  token?: string;
}
export interface DERPMeshOptions {
  server?: string;
  server_port?: number;
  host?: string;
  tls?: OutboundTLSOptions;
  detour?: string;
  bind_interface?: string;
  inet4_bind_address?: string;
  inet6_bind_address?: string;
  bind_address_no_port?: boolean;
  protect_path?: string;
  routing_mark?: number | string;
  reuse_addr?: boolean;
  netns?: string;
  connect_timeout?: Duration;
  tcp_fast_open?: boolean;
  tcp_multi_path?: boolean;
  disable_tcp_keep_alive?: boolean;
  tcp_keep_alive?: Duration;
  tcp_keep_alive_interval?: Duration;
  udp_fragment?: boolean;
  domain_resolver?: DomainResolver;
  network_strategy?: "default" | "fallback" | "hybrid";
  network_type?: InterfaceType | InterfaceType[];
  fallback_network_type?: InterfaceType | InterfaceType[];
  fallback_delay?: Duration;
}
export interface HysteriaRealmUser {
  name?: string;
  token?: string;
  max_realms?: number;
}
export interface OCMUser {
  name?: string;
  token?: string;
}
export interface USBIPDeviceMatch {
  bus_id?: string;
  vendor_id?: number;
  product_id?: number;
  serial?: string;
}
export interface ExperimentalOptions {
  cache_file?: CacheFileOptions;
  clash_api?: ClashAPIOptions;
  v2ray_api?: V2RayAPIOptions;
  debug?: DebugOptions;
}
export interface CacheFileOptions {
  enabled?: boolean;
  path?: string;
  cache_id?: string;
  store_fakeip?: boolean;
  rdrc_timeout?: Duration;
  store_dns?: boolean;
}
export interface ClashAPIOptions {
  external_controller?: string;
  external_ui?: string;
  external_ui_download_url?: string;
  external_ui_download_detour?: string;
  secret?: string;
  default_mode?: string;
  access_control_allow_origin?: string | string[];
  access_control_allow_private_network?: boolean;
}
export interface V2RayAPIOptions {
  listen?: string;
  stats?: V2RayStatsServiceOptions;
}
export interface V2RayStatsServiceOptions {
  enabled?: boolean;
  inbounds?: string[];
  outbounds?: string[];
  users?: string[];
}
export interface DebugOptions {
  listen?: string;
  gc_percent?: number;
  max_stack?: number;
  max_threads?: number;
  panic_on_fault?: boolean;
  trace_back?: string;
  memory_limit?: number | string;
  oom_killer?: boolean;
}
