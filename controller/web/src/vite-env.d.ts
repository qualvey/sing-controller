/// <reference types="vite/client" />

// .grammar 文件由 @lezer/generator 的 rollup 插件编译，导出编译后的 LRParser
declare module '*.grammar' {
  import type { LRParser } from '@lezer/lr'
  export const parser: LRParser
}
