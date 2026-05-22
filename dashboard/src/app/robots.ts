import { MetadataRoute } from 'next';

const SITE_URL = 'https://dashboard.hystersis.ai';

export default function robots(): MetadataRoute.Robots {
  return {
    rules: [
      {
        userAgent: '*',
        allow: ['/demo', '/playground', '/auth/signin', '/auth/signup'],
        disallow: [
          '/api/',
          '/admin/',
          '/memories',
          '/entities',
          '/sessions',
          '/agents',
          '/groups',
          '/projects',
          '/skills',
          '/chains',
          '/webhooks',
          '/api-keys',
          '/alerts',
          '/users',
          '/analytics',
          '/notifications',
          '/settings',
          '/search',
          '/documents',
          '/billing',
        ],
      },
    ],
    sitemap: `${SITE_URL}/sitemap.xml`,
  };
}
