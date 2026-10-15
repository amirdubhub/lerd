import { describe, it, expect } from 'vitest';
import { editorSurfaces, keywordRule, semanticRules } from './editor-theme';

describe('editor theme', () => {
  it('paints the editor on the theme surfaces', () => {
    const vars: Record<string, string> = { '--lerd-card': '#1e2030', '--color-gray-50': '#eff1f5' };
    expect(editorSurfaces((n) => vars[n] ?? '')).toEqual({ dark: '#1e2030', light: '#eff1f5' });
  });

  it('falls back to the built-in surfaces when a value is missing or not hex', () => {
    const vars: Record<string, string> = { '--color-gray-50': 'oklch(98.5% 0 0)' };
    expect(editorSurfaces((n) => vars[n] ?? '')).toEqual({ dark: '#161616', light: '#f9fafb' });
  });

  it('colours keywords with the accent', () => {
    expect(keywordRule('#8aadf4')).toEqual({ token: 'keyword', foreground: '8aadf4' });
  });

  it('colours the classes, calls and members phpantom names', () => {
    for (const dark of [true, false]) {
      const tokens = semanticRules(dark).map((r) => r.token);
      expect(tokens).toEqual(expect.arrayContaining(['class', 'method', 'function', 'property', 'parameter']));
    }
  });
});
