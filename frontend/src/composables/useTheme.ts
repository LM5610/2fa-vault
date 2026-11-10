import { ref } from 'vue';

export type Theme = 'dark' | 'light';

const theme = ref<Theme>((localStorage.getItem('theme_preference') as Theme) || 'dark');

export function useTheme() {
  function applyTheme(newTheme: Theme) {
    theme.value = newTheme;
    localStorage.setItem('theme_preference', newTheme);
    const root = document.documentElement;
    if (newTheme === 'dark') {
      root.classList.add('dark');
    } else {
      root.classList.remove('dark');
    }
  }

  function toggleTheme() {
    applyTheme(theme.value === 'dark' ? 'light' : 'dark');
  }

  // Initial apply
  applyTheme(theme.value);

  return {
    theme,
    applyTheme,
    toggleTheme,
  };
}
