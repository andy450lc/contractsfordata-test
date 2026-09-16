import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

function source(relativePath: string) {
  return readFileSync(new URL(relativePath, import.meta.url), 'utf8')
}

describe('marketing template entry points', () => {
  it('mounts the delivery dialog globally', () => {
    expect(source('../layouts/Base.astro')).toContain('<TemplateModal />')
  })

  it('gates desktop, mobile, hero, inline, footer, and error-page triggers', () => {
    expect(source('./SiteHeader.astro').match(/data-open-template/g)).toHaveLength(2)
    expect(source('../pages/index.astro').match(/data-open-template/g)).toHaveLength(2)
    expect(source('./SiteFooter.astro')).toContain('data-open-template')
    expect(source('../pages/404.astro')).toContain('data-open-template')
  })

  it('contains no static PDF or anchor bypass', () => {
    const files = [
      './SiteHeader.astro',
      './SiteFooter.astro',
      './TemplateModal.astro',
      '../pages/index.astro',
      '../pages/404.astro',
    ].map(source)

    for (const file of files) {
      expect(file).not.toMatch(/template\.pdf|TEMPLATE_PDF/i)
      expect(file).not.toMatch(/<a[^>]*data-open-template/i)
    }
  })
})
