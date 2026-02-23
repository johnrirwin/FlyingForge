import { describe, expect, it } from 'vitest'
import { stripHtmlToText } from './textUtils'

describe('stripHtmlToText', () => {
  it('returns empty string for undefined or empty input', () => {
    expect(stripHtmlToText()).toBe('')
    expect(stripHtmlToText('')).toBe('')
  })

  it('removes html tags and normalizes whitespace', () => {
    expect(stripHtmlToText('<p>Hello <strong>world</strong></p>\n<div>Again</div>')).toBe('Hello world Again')
  })

  it('decodes html entities', () => {
    expect(stripHtmlToText('News &amp; updates&nbsp;today')).toBe('News & updates today')
  })

  it('strips tags that were entity-encoded in source text', () => {
    expect(stripHtmlToText('&lt;img src=x onerror=alert(1)&gt; Safe &amp; sound')).toBe('Safe & sound')
  })
})
