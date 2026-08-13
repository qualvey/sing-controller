// Tailwind 化回归：指示条对齐（10 场景）+ 折叠态逻辑
import puppeteer from 'puppeteer-core'

const EDGE = 'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe'
const browser = await puppeteer.launch({
  executablePath: EDGE,
  headless: true,
  args: ['--disable-gpu', '--no-sandbox']
})

const page = await browser.newPage()
await page.setViewport({ width: 900, height: 480 })
await page.goto('http://localhost:5199/dns', { waitUntil: 'networkidle0' })
await new Promise((r) => setTimeout(r, 1000))

const sample = () =>
  page.evaluate(() => {
    const nav = document.querySelector('aside .overflow-y-auto')
    const item = document.querySelector('.nav-item.active')
    const ind = document.querySelector('.sidebar-tab-indicator')
    if (!nav || !item || !ind) return null
    const ir = item.getBoundingClientRect()
    const dr = ind.getBoundingClientRect()
    return {
      label: item.textContent.trim(),
      scrollTop: nav.scrollTop.toFixed(1),
      dy: (dr.top - ir.top).toFixed(1),
      dx: (dr.left - ir.left).toFixed(1),
      dh: (dr.height - ir.height).toFixed(1),
      dw: (dr.width - ir.width).toFixed(1)
    }
  })

const check = async (name) => console.log(`  ${name}:`, JSON.stringify(await sample()))

console.log('1) 初始'); await check('初始')
console.log('2) 真实滚轮到底'); await page.mouse.move(100, 300); await page.mouse.wheel({ deltaY: 200 }); await new Promise((r) => setTimeout(r, 900)); await check('滚动后')
console.log('3) 点击 Settings'); await page.click('a[href="/settings"]'); await new Promise((r) => setTimeout(r, 900)); await check('点击后')
console.log('4) 点击 Inbounds'); await page.click('a[href="/inbounds"]'); await new Promise((r) => setTimeout(r, 900)); await check('点击后')
console.log('5) 回滚中段'); await page.mouse.wheel({ deltaY: -60 }); await new Promise((r) => setTimeout(r, 900)); await check('回滚后')
console.log('6) 程序化滚动到底'); await page.evaluate(() => { const nav = document.querySelector('aside .overflow-y-auto'); nav.scrollTop = nav.scrollHeight; nav.dispatchEvent(new Event('scroll')) }); await new Promise((r) => setTimeout(r, 400)); await check('滚动后')

const align = await page.evaluate(() => {
  const item = document.querySelector('.nav-item')
  const icon = item.querySelector('svg')
  const label = item.querySelector('span')
  const iconR = icon.getBoundingClientRect()
  const labelL = label.getBoundingClientRect().left
  return { justify: getComputedStyle(item).justifyContent, gap: (labelL - iconR.right).toFixed(1) }
})
console.log('7) 桌面菜单(应左对齐, gap≈10px):', JSON.stringify(align))

await page.setViewport({ width: 400, height: 600 })
await new Promise((r) => setTimeout(r, 500))
const mob = await page.evaluate(() => {
  const item = document.querySelector('.nav-item')
  const icon = item.querySelector('svg')
  const itemRect = item.getBoundingClientRect()
  const iconRect = icon.getBoundingClientRect()
  return {
    justify: getComputedStyle(item).justifyContent,
    iconCenterDelta: ((iconRect.left + iconRect.width / 2) - (itemRect.left + itemRect.width / 2)).toFixed(1),
    labelDisplay: getComputedStyle(item.querySelector('span')).display
  }
})
console.log('8) 移动折叠(应居中, icon偏移≈0, label隐藏):', JSON.stringify(mob))

await page.evaluate(() => document.querySelector('.icon-btn').click())
await new Promise((r) => setTimeout(r, 500))
const mobOpen = await page.evaluate(() => {
  const item = document.querySelector('.nav-item')
  const icon = item.querySelector('svg')
  const label = item.querySelector('span')
  const itemRect = item.getBoundingClientRect()
  const iconRect = icon.getBoundingClientRect()
  const ind = document.querySelector('.sidebar-tab-indicator')
  const itemA = document.querySelector('.nav-item.active')
  const ir = itemA.getBoundingClientRect()
  const dr = ind.getBoundingClientRect()
  return {
    justify: getComputedStyle(item).justifyContent,
    iconLeftDelta: (iconRect.left - itemRect.left).toFixed(1),
    labelVisible: getComputedStyle(label).display !== 'none',
    indDy: (dr.top - ir.top).toFixed(1)
  }
})
console.log('9) 移动展开(应左对齐, icon左边距≈12px, label可见, 指示条对齐):', JSON.stringify(mobOpen))

const p2 = await browser.newPage()
await p2.setViewport({ width: 900, height: 700 })
await p2.goto('http://localhost:5199/proxies', { waitUntil: 'networkidle0' })
await new Promise((r) => setTimeout(r, 1000))
const s2 = await p2.evaluate(() => {
  const item = document.querySelector('.nav-item.active')
  const ind = document.querySelector('.sidebar-tab-indicator')
  const ir = item.getBoundingClientRect()
  const dr = ind.getBoundingClientRect()
  return { label: item.textContent.trim(), dy: (dr.top - ir.top).toFixed(1), dx: (dr.left - ir.left).toFixed(1) }
})
console.log('10) 深链 /proxies h=700:', JSON.stringify(s2))
await p2.close()

await browser.close()
console.log('\ndone')
