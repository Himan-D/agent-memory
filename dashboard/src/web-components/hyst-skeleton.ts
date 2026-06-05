import { LitElement, html, css } from 'lit';

export class HystSkeleton extends LitElement {
  static styles = css`
    :host {
      display: block;
    }
    .skeleton {
      border-radius: var(--radius, 0.375rem);
      background: linear-gradient(
        90deg,
        hsl(var(--muted, 240 4.8% 95.9%)) 25%,
        hsl(var(--muted, 240 4.8% 95.9%) / 0.5) 50%,
        hsl(var(--muted, 240 4.8% 95.9%)) 75%
      );
      background-size: 200% 100%;
      animation: shimmer 1.5s ease-in-out infinite;
    }
    @keyframes shimmer {
      0% { background-position: 200% 0; }
      100% { background-position: -200% 0; }
    }
  `;

  static properties = {
    width: { type: String },
    height: { type: String },
    circle: { type: Boolean },
  };

  width: string;
  height: string;
  circle: boolean;

  constructor() {
    super();
    this.width = '100%';
    this.height = '1rem';
    this.circle = false;
  }

  render() {
    const radius = this.circle ? 'border-radius: 50%;' : '';
    return html`
      <div
        class="skeleton"
        style="width: ${this.width}; height: ${this.height}; ${radius}"
        part="skeleton"
      ></div>
    `;
  }
}

customElements.define('hyst-skeleton', HystSkeleton);
