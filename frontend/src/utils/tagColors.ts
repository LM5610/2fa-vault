// Deterministic Modern Color System for Tags

export interface TagColorSet {
  name: string;
  badge: string;
  activeBadge: string;
  gradientBar: string;
  dot: string;
  text: string;
}

const PALETTES: TagColorSet[] = [
  // 0. Blue (Work / Office)
  {
    name: 'blue',
    badge: 'bg-blue-50 dark:bg-blue-950/40 text-blue-600 dark:text-blue-400 border border-blue-200/80 dark:border-blue-800/60',
    activeBadge: 'bg-blue-600 text-white shadow-sm shadow-blue-500/20',
    gradientBar: 'from-blue-500 to-indigo-500',
    dot: 'bg-blue-500',
    text: 'text-blue-500',
  },
  // 1. Amber (Crypto / Finance)
  {
    name: 'amber',
    badge: 'bg-amber-50 dark:bg-amber-950/40 text-amber-700 dark:text-amber-400 border border-amber-200/80 dark:border-amber-800/60',
    activeBadge: 'bg-amber-600 text-white shadow-sm shadow-amber-500/20',
    gradientBar: 'from-amber-500 to-yellow-500',
    dot: 'bg-amber-500',
    text: 'text-amber-500',
  },
  // 2. Emerald (Social / Chat)
  {
    name: 'emerald',
    badge: 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400 border border-emerald-200/80 dark:border-emerald-800/60',
    activeBadge: 'bg-emerald-600 text-white shadow-sm shadow-emerald-500/20',
    gradientBar: 'from-emerald-500 to-teal-500',
    dot: 'bg-emerald-500',
    text: 'text-emerald-500',
  },
  // 3. Purple (Gaming / Media)
  {
    name: 'purple',
    badge: 'bg-purple-50 dark:bg-purple-950/40 text-purple-600 dark:text-purple-400 border border-purple-200/80 dark:border-purple-800/60',
    activeBadge: 'bg-purple-600 text-white shadow-sm shadow-purple-500/20',
    gradientBar: 'from-purple-500 to-pink-500',
    dot: 'bg-purple-500',
    text: 'text-purple-500',
  },
  // 4. Cyan (Dev / Tech)
  {
    name: 'cyan',
    badge: 'bg-cyan-50 dark:bg-cyan-950/40 text-cyan-600 dark:text-cyan-400 border border-cyan-200/80 dark:border-cyan-800/60',
    activeBadge: 'bg-cyan-600 text-white shadow-sm shadow-cyan-500/20',
    gradientBar: 'from-cyan-500 to-sky-500',
    dot: 'bg-cyan-500',
    text: 'text-cyan-500',
  },
  // 5. Rose (Email / Personal)
  {
    name: 'rose',
    badge: 'bg-rose-50 dark:bg-rose-950/40 text-rose-600 dark:text-rose-400 border border-rose-200/80 dark:border-rose-800/60',
    activeBadge: 'bg-rose-600 text-white shadow-sm shadow-rose-500/20',
    gradientBar: 'from-rose-500 to-pink-500',
    dot: 'bg-rose-500',
    text: 'text-rose-500',
  },
  // 6. Orange (Cloud / Server)
  {
    name: 'orange',
    badge: 'bg-orange-50 dark:bg-orange-950/40 text-orange-600 dark:text-orange-400 border border-orange-200/80 dark:border-orange-800/60',
    activeBadge: 'bg-orange-600 text-white shadow-sm shadow-orange-500/20',
    gradientBar: 'from-orange-500 to-amber-500',
    dot: 'bg-orange-500',
    text: 'text-orange-500',
  },
  // 7. Indigo (Default / Secondary)
  {
    name: 'indigo',
    badge: 'bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 border border-indigo-200/80 dark:border-indigo-800/60',
    activeBadge: 'bg-indigo-600 text-white shadow-sm shadow-indigo-500/20',
    gradientBar: 'from-indigo-500 to-violet-500',
    dot: 'bg-indigo-500',
    text: 'text-indigo-500',
  },
];

// Keywords mapping to preferred color
const KEYWORD_MAP: Record<string, number> = {
  work: 0,
  office: 0,
  job: 0,
  company: 0,
  crypto: 1,
  btc: 1,
  eth: 1,
  finance: 1,
  binance: 1,
  bank: 1,
  social: 2,
  chat: 2,
  wechat: 2,
  qq: 2,
  discord: 3,
  game: 3,
  steam: 3,
  epic: 3,
  dev: 4,
  code: 4,
  github: 4,
  git: 4,
  tech: 4,
  email: 5,
  mail: 5,
  gmail: 5,
  personal: 5,
  cloud: 6,
  vps: 6,
  server: 6,
  aws: 6,
};

/**
 * Returns a deterministic color palette set for a given tag
 */
export function getTagColor(tag: string): TagColorSet {
  const norm = (tag || '').toLowerCase().trim();

  // Keyword match
  for (const [kw, idx] of Object.entries(KEYWORD_MAP)) {
    if (norm.includes(kw)) {
      return PALETTES[idx];
    }
  }

  // Deterministic string hash
  let hash = 0;
  for (let i = 0; i < norm.length; i++) {
    hash = (hash << 5) - hash + norm.charCodeAt(i);
    hash |= 0;
  }
  const index = Math.abs(hash) % PALETTES.length;
  return PALETTES[index];
}

/**
 * Returns the accent top gradient bar class for an account based on its first tag
 */
export function getCardTagGradient(tags?: string[]): string | null {
  if (!tags || tags.length === 0) return null;
  return getTagColor(tags[0]).gradientBar;
}
