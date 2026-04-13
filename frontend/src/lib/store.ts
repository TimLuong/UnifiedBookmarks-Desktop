import { writable, derived } from 'svelte/store';

// ── Settings (persisted to localStorage) ──────────────

export type ThemeMode = 'dark' | 'light' | 'sand';
export type FontSize = 'small' | 'medium' | 'large';

export interface AppSettings {
  theme: ThemeMode;
  fontSize: FontSize;
}

const defaultSettings: AppSettings = { theme: 'dark', fontSize: 'small' };

function loadSettings(): AppSettings {
  try {
    const raw = localStorage.getItem('ub-settings');
    if (raw) return { ...defaultSettings, ...JSON.parse(raw) };
  } catch {}
  return { ...defaultSettings };
}

function createSettingsStore() {
  const { subscribe, set, update } = writable<AppSettings>(loadSettings());
  return {
    subscribe,
    set(val: AppSettings) {
      localStorage.setItem('ub-settings', JSON.stringify(val));
      set(val);
    },
    update(fn: (s: AppSettings) => AppSettings) {
      update(s => {
        const next = fn(s);
        localStorage.setItem('ub-settings', JSON.stringify(next));
        return next;
      });
    },
  };
}

export const settings = createSettingsStore();

// Derived convenience stores
export const theme = derived(settings, s => s.theme);
export const fontSize = derived(settings, s => s.fontSize);

export interface Bookmark {
  id: string;
  title: string;
  url: string;
  folderPath: string;
  browser: string;
  profileDir: string;
  displayName: string;
  dateAdded: string;
  category: string;
  confidence: number;
  paraType: string;
  paraContext: string;
  tags: string[];
}

export interface Profile {
  browser: string;
  browserLabel: string;
  profileDir: string;
  displayName: string;
  bookmarksPath: string;
  hasBookmarks: boolean;
  userDataDir: string;
}

export interface Snapshot {
  id: string;
  timestamp: string;
  browser: string;
  profile: string;
  count: number;
  filePath: string;
  sizeBytes: number;
}

export const bookmarks = writable<Bookmark[]>([]);
export const profiles = writable<Profile[]>([]);
export const view = writable<'list' | 'cards'>('list');
export const activePara = writable<string | null>(null);
export const activeTag = writable<string | null>(null);
export const activeCat = writable<string | null>(null);
export const activeFolder = writable<string | null>(null);      // folder path filter
export const searchQuery = writable('');
export const sortBy = writable<'date-desc' | 'date-asc' | 'name-asc' | 'name-desc' | 'site-asc' | 'site-desc'>('date-desc');
export const showInList = writable({ cover: true, title: true, note: true, description: true, tags: true, bookmarksInfo: true });
export const loading = writable('');
export const statusMsg = writable('');
export const currentTab = writable<'bookmarks' | 'restore'>('bookmarks');
export const showConsole = writable(false);
export const showSettings = writable(false);
export const showPrompts = writable(false);

// ── Prompt Profiles ─────────────────────────────────────────

export interface PromptProfile {
  id: string;
  name: string;
  description: string;
  content: string;
  isBuiltin: boolean;
}

export interface LogEntry {
  time: string;
  level: 'info' | 'warn' | 'error' | 'success' | 'debug';
  message: string;
}

export const consoleLogs = writable<LogEntry[]>([]);

export function addLog(level: LogEntry['level'], message: string) {
  const time = new Date().toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' });
  consoleLogs.update(logs => [...logs, { time, level, message }]);
}

// ── Sync Preview types ────────────────────────────────

export interface ProfileDiff {
  browser: string;
  browserLabel: string;
  profileDir: string;
  displayName: string;
  beforeCount: number;
  afterCount: number;
  added: number;
  removed: number;
  unchanged: number;
  addedSample: Bookmark[];
  removedSample: Bookmark[];
}

export interface SyncPreviewResult {
  diffs: ProfileDiff[];
  totalBefore: number;
  totalAfter: number;
}

// ── AI Config types ───────────────────────────────────

export interface AIConfig {
  endpoint: string;
  apiKey: string;
  model: string;
  systemPrompt?: string;
}

export interface TestResult {
  ok: boolean;
  message: string;
  model: string;
}

// ── Folder Tree types ─────────────────────────────────

export interface FolderNode {
  name: string;
  path: string;
  count: number;
  urls: string[];       // direct bookmark URLs at this exact folder level
  children: FolderNode[];
}

export interface ProfileTree {
  browser: string;
  browserLabel: string;
  profileDir: string;
  displayName: string;
  totalCount: number;
  roots: FolderNode[];
}

export const folderTree = writable<ProfileTree[]>([]);

// null = show all profiles; Set of "browser__profileDir" = only those
export const activeProfiles = writable<Set<string> | null>(null);

// Set of URLs belonging to the currently selected sidebar folder; null = no folder filter
export const activeFolderUrls = writable<Set<string> | null>(null);

// Open a bookmark URL in the in-app browser panel
export const openInApp = writable<{ url: string; title: string } | null>(null);

// ── Pipeline step tracking ────────────────────────────
// 0=fresh, 1=scanned, 2=collected, 3=analyzed, 4=synced
export const pipelineStep = writable<0 | 1 | 2 | 3 | 4>(0);

// ── Cross-profile unique-URL toggle ──────────────────
// true = show only one entry per URL (first occurrence wins)
export const uniqueUrlsOnly = writable<boolean>(false);
