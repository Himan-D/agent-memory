import { LitElement, html, css } from 'lit';

export class HystStatsCard extends LitElement {
  static styles = css`
    :host {
      display: block;
      --card-bg: hsl(var(--card, 0 0% 100%));
      --card-fg: hsl(var(--card-foreground, 240 10% 3.9%));
      --muted-fg: hsl(var(--muted-foreground, 240 3.8% 46.1%));
      --primary: hsl(var(--primary, 240 5.9% 10%));
      --border-color: hsl(var(--border, 240 5.9% 90%));
      --radius: var(--radius, 0.75rem);
    }
    .card {
      background: var(--card-bg);
      border: 1px solid var(--border-color);
      border-radius: var(--radius);
      padding: 1.5rem;
      transition: all 0.3s ease;
    }
    .card:hover {
      box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1);
      transform: translateY(-2px);
    }
    .header {
      display: flex;
      align-items: flex-start;
      justify-content: space-between;
    }
    .info {
      display: flex;
      flex-direction: column;
      gap: 0.5rem;
    }
    .title {
      font-size: 0.875rem;
      font-weight: 500;
      color: var(--muted-fg);
      margin: 0;
    }
    .value {
      font-size: 1.875rem;
      font-weight: 700;
      color: var(--card-fg);
      margin: 0;
      line-height: 1;
    }
    .description {
      font-size: 0.75rem;
      color: var(--muted-fg);
      margin: 0;
    }
    .trend {
      display: flex;
      align-items: center;
      gap: 0.25rem;
      font-size: 0.75rem;
      font-weight: 500;
    }
    .trend.positive { color: #16a34a; }
    .trend.negative { color: #dc2626; }
    .trend-label { color: var(--muted-fg); }
    .icon-box {
      border-radius: 0.5rem;
      padding: 0.75rem;
      background: color-mix(in srgb, var(--primary) 10%, transparent);
    }
    .icon-box svg {
      width: 1.5rem;
      height: 1.5rem;
      color: var(--primary);
    }
  `;

  static properties = {
    title: { type: String },
    value: { type: String },
    description: { type: String },
    trendValue: { type: String, attribute: 'trend-value' },
    trendPositive: { type: Boolean, attribute: 'trend-positive' },
    iconSvg: { type: String, attribute: 'icon-svg' },
  };

  title: string;
  value: string;
  description: string;
  trendValue: string;
  trendPositive: boolean;
  iconSvg: string;

  constructor() {
    super();
    this.title = '';
    this.value = '0';
    this.description = '';
    this.trendValue = '';
    this.trendPositive = true;
    this.iconSvg = '';
  }

  render() {
    const trendClass = this.trendPositive ? 'positive' : 'negative';
    const prefix = this.trendPositive ? '+' : '';
    return html`
      <div class="card" part="card">
        <div class="header">
          <div class="info">
            <p class="title">${this.title}</p>
            <p class="value">${this.value}</p>
            ${this.description ? html`<p class="description">${this.description}</p>` : ''}
            ${this.trendValue ? html`
              <div class="trend ${trendClass}">
                <span>${prefix}${this.trendValue}%</span>
                <span class="trend-label">vs last month</span>
              </div>
            ` : ''}
          </div>
          ${this.iconSvg ? html`
            <div class="icon-box"></div>
          ` : ''}
        </div>
      </div>
    `;
  }

  updated() {
    if (this.iconSvg) {
      const box = this.shadowRoot?.querySelector('.icon-box');
      if (box && !box.innerHTML) {
        const template = document.createElement('template');
        template.innerHTML = this.iconSvg;
        box.appendChild(template.content.cloneNode(true));
      }
    }
  }
}

customElements.define('hyst-stats-card', HystStatsCard);
