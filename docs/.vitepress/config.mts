import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'sing-controller & WebUI',
  description: 'sing-box 现代化配置管理服务与 WebUI 控制面板文档',
  lang: 'zh-CN',
  base: '/',
  cleanUrls: true,
  lastUpdated: true,
  ignoreDeadLinks: true,

  head: [
    ['meta', { name: 'theme-color', content: '#3eaf7c' }],
  ],

  themeConfig: {
    siteTitle: 'sing-controller',
    logo: '/logo.svg',

    nav: [
      { text: '快速指南', link: '/guide/quickstart' },
      { text: '核心架构', link: '/architecture/overview' },
      { text: '配置参考', link: '/config/controller-settings' },
      { text: 'API 文档', link: '/api/index' },
      { text: 'WebUI 开发', link: '/webui/ui-guide' },
      { text: '更新日志', link: '/changelog' },
    ],

    sidebar: {
      '/guide/': [
        {
          text: '入门指南',
          items: [
            { text: '快速启动', link: '/guide/quickstart' },
            { text: 'Linux systemd 双服务部署', link: '/guide/deployment' },
            { text: 'Windows 桌面与托管', link: '/guide/windows-managed' },
            { text: 'OpenWrt 部署', link: '/guide/openwrt' },
          ],
        },
      ],
      '/architecture/': [
        {
          text: '系统架构',
          items: [
            { text: '设计理念与协议原则', link: '/architecture/overview' },
            { text: '内核托管与重载机制', link: '/architecture/supervisor' },
            { text: '分布式集群架构', link: '/architecture/cluster' },
            { text: '集群需求评审', link: '/architecture/requirement-review' },
            { text: '数据库与状态迁移', link: '/architecture/database' },
          ],
        },
      ],
      '/config/': [
        {
          text: '配置手册',
          items: [
            { text: 'Controller 配置 (controller.json)', link: '/config/controller-settings' },
            { text: '原子保存与校验管线', link: '/config/validation' },
          ],
        },
      ],
      '/api/': [
        {
          text: '接口体系 (OpenAPI)',
          items: [
            { text: 'API 概览与设计规范', link: '/api/index' },
            { text: '在线交互文档 (Scalar / Swagger)', link: '/api/interactive' },
            { text: 'RESTful 核心接口', link: '/api/rest' },
          ],
        },
      ],
      '/webui/': [
        {
          text: 'WebUI 开发与规范',
          items: [
            { text: 'UI 规范与页面拼装指南', link: '/webui/ui-guide' },
            { text: '代码编辑器 (CodeMirror) 集成', link: '/webui/editor' },
            { text: '前端技术栈与协议', link: '/webui/tech-stack' },
          ],
        },
      ],
    },

    search: {
      provider: 'local',
      options: {
        translations: {
          button: {
            buttonText: '搜索文档',
            buttonAriaLabel: '搜索文档',
          },
          modal: {
            noResultsText: '无法找到相关结果',
            resetButtonTitle: '清除查询条件',
            footer: {
              selectText: '选择',
              navigateText: '切换',
              closeText: '关闭',
            },
          },
        },
      },
    },

    outline: {
      level: [2, 3],
      label: '本页导航',
    },

    docFooter: {
      prev: '上一篇',
      next: '下一篇',
    },

    darkModeSwitchLabel: '深浅模式切换',
    returnToTopLabel: '返回顶部',
    sidebarMenuLabel: '目录',
  },
})
