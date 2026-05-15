declare namespace React {
  namespace JSX {
    interface IntrinsicElements {
      'hyst-stats-card': React.DetailedHTMLProps<
        React.HTMLAttributes<HTMLElement> & {
          title?: string;
          value?: string;
          description?: string;
          'icon-svg'?: string;
          'trend-value'?: string;
          'trend-positive'?: string;
        },
        HTMLElement
      >;
      'hyst-skeleton': React.DetailedHTMLProps<
        React.HTMLAttributes<HTMLElement> & {
          width?: string;
          height?: string;
          circle?: boolean;
        },
        HTMLElement
      >;
      'hyst-badge': React.DetailedHTMLProps<
        React.HTMLAttributes<HTMLElement> & {
          variant?: 'default' | 'secondary' | 'outline' | 'destructive' | 'success' | 'warning';
        },
        HTMLElement
      >;
      'hyst-progress-bar': React.DetailedHTMLProps<
        React.HTMLAttributes<HTMLElement> & {
          label?: string;
          value?: number;
          max?: number;
          color?: 'green' | 'blue' | 'purple' | 'amber' | 'red';
        },
        HTMLElement
      >;
      'hyst-theme-toggle': React.DetailedHTMLProps<
        React.HTMLAttributes<HTMLElement> & {
          theme?: 'light' | 'dark' | 'system';
        },
        HTMLElement
      >;
    }
  }
}
