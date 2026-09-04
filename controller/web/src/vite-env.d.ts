/// <reference types="vite/client" />

// .grammar 文件由 @lezer/generator 的 rollup 插件编译，导出编译后的 LRParser
declare module '*.grammar' {
  import type { LRParser } from '@lezer/lr'
  export const parser: LRParser
}

declare module 'virtual:recent-changes' {
  export interface CommitInfo {
    hash: string
    message: string
    date: string
    author: string
  }
  export interface BuildInfo {
    version: string
    branch: string
    commit: string
    buildTime: string
    latestChanges: CommitInfo[]
  }
  export const buildInfo: BuildInfo
}
