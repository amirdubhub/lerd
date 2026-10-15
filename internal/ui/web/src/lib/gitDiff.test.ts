import { describe, it, expect } from 'vitest';
import { diffLines, diffStats } from './gitDiff';

describe('diffLines', () => {
  it('drops the file header and classifies each line', () => {
    const patch = [
      'diff --git a/a.php b/a.php',
      'index 1..2 100644',
      '--- a/a.php',
      '+++ b/a.php',
      '@@ -1,2 +1,2 @@',
      ' keep',
      '-old',
      '+new',
      '\\ No newline at end of file',
      ''
    ].join('\n');
    expect(diffLines(patch).map((l) => l.kind)).toEqual(['hunk', 'context', 'del', 'add', 'meta']);
  });

  // "--- a/x" inside a hunk is a removed line that starts with "--", not a header.
  it('keeps hunk lines that look like headers', () => {
    const patch = '--- a/x\n+++ b/x\n@@ -1 +1 @@\n--- old dashes\n+++ new pluses\n';
    expect(diffLines(patch).map(({ kind, text }) => ({ kind, text }))).toEqual([
      { kind: 'hunk', text: '@@ -1 +1 @@' },
      { kind: 'del', text: '--- old dashes' },
      { kind: 'add', text: '+++ new pluses' }
    ]);
  });

  it('shows the header when there is no hunk, as for a binary file', () => {
    const patch = 'diff --git a/logo.png b/logo.png\nBinary files a/logo.png and b/logo.png differ\n';
    expect(diffLines(patch).map((l) => l.kind)).toEqual(['meta', 'meta']);
  });
});

describe('diffLines numbering', () => {
  it('numbers each side from the hunk header and restarts at the next hunk', () => {
    const patch = '@@ -12,3 +12,3 @@ class A\n keep\n-old\n+new\n@@ -40 +40,2 @@\n tail\n+added\n';
    expect(diffLines(patch).map((l) => [l.old, l.new])).toEqual([
      [null, null],
      [12, 12],
      [13, null],
      [null, 13],
      [null, null],
      [40, 40],
      [null, 41]
    ]);
  });
});

describe('diffStats', () => {
  it('counts added and removed lines, not headers that look like them', () => {
    const patch = '--- a/x\n+++ b/x\n@@ -1,2 +1,2 @@\n--- gone\n+one\n+two\n context\n';
    expect(diffStats(diffLines(patch))).toEqual({ added: 2, removed: 1 });
  });
});
