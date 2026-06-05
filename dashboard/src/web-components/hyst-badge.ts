import { LitElement, html, css } from 'lit';

export class HystBadge extends LitElement {
  static styles = css`
    :host {
      display: inline-flex;
    }
    .badge {
      display: inline-flex;
      align-items: center;
      border-radius: 9999px;
      padding: 0.125rem 0.625rem;
      font-size: 0.75rem;
      font-weight: 500;
      line-height: 1.25rem;
      white-space: nowrap;
    }
    .badge--default {
      background: hsl(var(--primary, 240 5.9% 10%));
      color: hsl(var(--primary-foreground, 0 0% 98%));
    }
    .badge--secondary {
      background: hsl(var(--secondary, 240 4.8% 95.9%));
      color: hsl(var(--secondary-foreground, 240 5.9% 10%));
    }
    .badge--outline {
      background: transparent;
      color: hsl(var(--foreground, 240 10% 3.9%));
      border: 1px solid hsl(var(--border, 240 5.9% 90%));
    }
    .badge--destructive {
      background: hsl(var(--destructive, 0 84.2% 60.2%));
      color: hsl(var(--destructive-foreground, 0 0% 98%));
    }
    .badge--success {
      background: #dcfce7;
      color: #166534;
    }
    .badge--warning {
      background: #fef3c7;
      color: #92400e;
    }
  `;

  static properties = {
    variant: { type: String },
  };

  variant: string;

  constructor() {
    super();
    this.variant = 'default';
  }

  render() {
    return html`
      <span class="badge badge--${this.variant}" part="badge">
        <slot></slot>
      </span>
    `;
  }
}

customElements.define('hyst-badge', HystBadge);
