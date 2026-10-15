import { apiJson } from './api';

export type GitDiffSection = 'staged' | 'unstaged' | 'untracked';

export interface GitChangedFile {
  path: string;
  // git's letter: A, M, D, T, U, or ? for untracked.
  status: string;
}

export type GitChanges = Record<GitDiffSection, GitChangedFile[]>;

export interface GitPatch {
  patch: string;
  truncated: boolean;
}

export const SECTIONS: GitDiffSection[] = ['staged', 'unstaged', 'untracked'];

function query(domain: string, path: string): string {
  return `domain=${encodeURIComponent(domain)}&path=${encodeURIComponent(path)}`;
}

export function loadGitChanges(domain: string, path: string): Promise<GitChanges> {
  return apiJson<GitChanges>(`/api/sites/git-diff?${query(domain, path)}`);
}

export function loadGitPatch(
  domain: string,
  path: string,
  section: GitDiffSection,
  file: string,
  ignoreSpace = false
): Promise<GitPatch> {
  const space = ignoreSpace ? '&ignore_space=1' : '';
  return apiJson<GitPatch>(
    `/api/sites/git-diff?${query(domain, path)}&section=${section}&file=${encodeURIComponent(file)}${space}`
  );
}

export type DiffLineKind = 'add' | 'del' | 'hunk' | 'meta' | 'context';

export interface DiffLine {
  kind: DiffLineKind;
  text: string;
  // Line numbers in the old and new file; null where the line is not on that side.
  old: number | null;
  new: number | null;
}

// diffLines drops git's file header (diff --git, index, ---/+++), which the file
// list already says, and keeps it only when there is no hunk, as for a binary file.
export function diffLines(patch: string): DiffLine[] {
  const lines = patch.replace(/\n$/, '').split('\n');
  const first = lines.findIndex((l) => l.startsWith('@@'));
  if (first < 0) return lines.filter((l) => l !== '').map((text) => ({ kind: 'meta', text, old: null, new: null }));
  let oldNo = 0;
  let newNo = 0;
  return lines.slice(first).map((text) => {
    const kind = kindOf(text);
    if (kind === 'hunk') {
      const m = /^@@ -(\d+)(?:,\d+)? \+(\d+)/.exec(text);
      oldNo = m ? Number(m[1]) : 0;
      newNo = m ? Number(m[2]) : 0;
      return { kind, text, old: null, new: null };
    }
    if (kind === 'add') return { kind, text, old: null, new: newNo++ };
    if (kind === 'del') return { kind, text, old: oldNo++, new: null };
    if (kind === 'context') return { kind, text, old: oldNo++, new: newNo++ };
    return { kind, text, old: null, new: null };
  });
}

export function diffStats(lines: DiffLine[]): { added: number; removed: number } {
  return {
    added: lines.filter((l) => l.kind === 'add').length,
    removed: lines.filter((l) => l.kind === 'del').length
  };
}

function kindOf(line: string): DiffLineKind {
  if (line.startsWith('@@')) return 'hunk';
  if (line.startsWith('+')) return 'add';
  if (line.startsWith('-')) return 'del';
  if (line.startsWith('\\')) return 'meta';
  return 'context';
}
