// The editor wears the chosen lerd theme: its background is the theme's
// surface and keywords take the accent. Themes carry no syntax colours, so the
// rest of the tokens use VS Code's Dark+ and Light+ tones.

const HEX = /^#[0-9a-f]{6}$/i;

function hexOr(v: string, fallback: string): string {
  const t = v.trim();
  return HEX.test(t) ? t : fallback;
}

// Light mode only moves --color-gray-50 when the theme declares a light page,
// otherwise Tailwind's own (not hex) stands and the stock grey is used.
export function editorSurfaces(read: (name: string) => string): { dark: string; light: string } {
  return {
    dark: hexOr(read('--lerd-card'), '#161616'),
    light: hexOr(read('--color-gray-50'), '#f9fafb')
  };
}

export function keywordRule(accent: string): { token: string; foreground: string } {
  return { token: 'keyword', foreground: accent.replace('#', '') };
}

// phpantom's full token stream names classes, calls and members, which Monaco's
// stock themes have no rules for, so without these they render as plain text.
const SEMANTIC = {
  dark: { type: '4ec9b0', call: 'dcdcaa', member: '9cdcfe', namespace: '4ec9b0' },
  light: { type: '267f99', call: '795e26', member: '001080', namespace: '267f99' }
};

export function semanticRules(dark: boolean): { token: string; foreground: string }[] {
  const c = dark ? SEMANTIC.dark : SEMANTIC.light;
  return [
    ...['class', 'interface', 'enum', 'type', 'typeParameter'].map((token) => ({ token, foreground: c.type })),
    ...['function', 'method'].map((token) => ({ token, foreground: c.call })),
    ...['property', 'parameter', 'enumMember'].map((token) => ({ token, foreground: c.member })),
    { token: 'namespace', foreground: c.namespace }
  ];
}
