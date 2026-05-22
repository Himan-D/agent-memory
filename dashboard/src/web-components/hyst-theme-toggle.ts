import { LitElement, html, css } from 'lit';

export class HystThemeToggle extends LitElement {
  static styles = css`
    :host {
      display: inline-flex;
    }
    button {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: 2.25rem;
      height: 2.25rem;
      border-radius: 0.5rem;
      border: 1px solid hsl(var(--border, 240 5.9% 90%));
      background: transparent;
      cursor: pointer;
      color: hsl(var(--muted-foreground, 240 3.8% 46.1%));
      transition: all 0.2s ease;
    }
    button:hover {
      background: hsl(var(--accent, 240 4.8% 95.9%));
      color: hsl(var(--foreground, 240 10% 3.9%));
    }
    svg {
      width: 1.125rem;
      height: 1.125rem;
    }
  `;

  static properties = {
    theme: { type: String },
  };

  theme: string;

  constructor() {
    super();
    this.theme = 'system';
  }

  _isDark() {
    if (this.theme === 'dark') return true;
    if (this.theme === 'light') return false;
    return typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches;
  }

  render() {
    const dark = this._isDark();
    return html`
      <button @click=${this._toggle} aria-label="Toggle theme" part="button">
        ${dark ? html`
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <circle cx="12" cy="12" r="5"/>
            <line x1="12" y1="1" x2="12" y2="3"/>
            <line x1="12" y1="21" x2="12" y2="23"/>
            <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/>
            <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/>
            <line x1="1" y1="12" x2="3" y2="12"/>
            <line x1="21" y1="12" x2="23" y2="12"/>
            <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/>
            <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>
          </svg>
        ` : html`
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
          </svg>
        `}
      </button>
    `;
  }

  _toggle() {
    const current = document.documentElement.classList.contains('dark') ? 'dark' : 'light';
    const next = current === 'dark' ? 'light' : 'dark';
    this.dispatchEvent(new CustomEvent('theme-change', {
      detail: { theme: next },
      bubbles: true,
      composed: true,
    }));
  }
}

customElements.define('hyst-theme-toggle', HystThemeToggle);
