import { LitElement, html, css } from 'lit';

export class HystProgressBar extends LitElement {
  static styles = css`
    :host {
      display: block;
    }
    .bar-container {
      display: flex;
      flex-direction: column;
      gap: 0.375rem;
    }
    .bar-label {
      display: flex;
      align-items: center;
      justify-content: space-between;
    }
    .bar-label-text {
      font-size: 0.875rem;
      font-weight: 500;
      color: hsl(var(--foreground, 240 10% 3.9%));
    }
    .bar-label-value {
      font-size: 0.875rem;
      color: hsl(var(--muted-foreground, 240 3.8% 46.1%));
    }
    .bar-track {
      height: 0.5rem;
      width: 100%;
      border-radius: 9999px;
      background: hsl(var(--muted, 240 4.8% 95.9%));
      overflow: hidden;
    }
    .bar-fill {
      height: 100%;
      border-radius: 9999px;
      transition: width 0.3s ease;
    }
    .bar-fill.green { background: #22c55e; }
    .bar-fill.blue { background: #3b82f6; }
    .bar-fill.purple { background: #8b5cf6; }
    .bar-fill.amber { background: #f59e0b; }
    .bar-fill.red { background: #ef4444; }
  `;

  static properties = {
    label: { type: String },
    value: { type: Number },
    max: { type: Number },
    color: { type: String },
  };

  label: string;
  value: number;
  max: number;
  color: string;

  constructor() {
    super();
    this.label = '';
    this.value = 0;
    this.max = 100;
    this.color = 'blue';
  }

  render() {
    const percent = Math.min(Math.max((this.value / this.max) * 100, 0), 100);
    return html`
      <div class="bar-container" part="container">
        <div class="bar-label">
          <span class="bar-label-text">${this.label}</span>
          <span class="bar-label-value">${Math.round(percent)}%</span>
        </div>
        <div class="bar-track">
          <div class="bar-fill ${this.color}" style="width: ${percent}%"></div>
        </div>
      </div>
    `;
  }
}

customElements.define('hyst-progress-bar', HystProgressBar);
