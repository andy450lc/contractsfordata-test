import { describe, expect, it } from 'vitest'

import { documentFilename } from './api'

describe('documentFilename', () => {
  it.each([
    [null, 'agreement.docx'],
    ['attachment; filename="Client SOW.docx"', 'Client SOW.docx'],
    ['attachment; filename=Client.pdf', 'Client.docx'],
    ["attachment; filename*=UTF-8''..%2F..%2FClient%20SOW.docx", 'Client SOW.docx'],
    ["attachment; filename*=UTF-8''%E0%A4%A", '_E0_A4_A.docx'],
    ['attachment; filename="..."', 'agreement.docx'],
  ])('turns %s into %s', (header, expected) => {
    expect(documentFilename(header)).toBe(expected)
  })
})
