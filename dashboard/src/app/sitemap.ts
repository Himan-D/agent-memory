import { MetadataRoute } from 'next';

const SITE_URL = 'https://dashboard.hystersis.ai';

export default function sitemap(): MetadataRoute.Sitemap {
  const now = new Date();

  const publicPages: MetadataRoute.Sitemap = [
    {
      url: `${SITE_URL}/demo`,
      lastModified: now,
      changeFrequency: 'monthly',
      priority: 1.0,
    },
    {
      url: `${SITE_URL}/playground`,
      lastModified: now,
      changeFrequency: 'monthly',
      priority: 0.9,
    },
    {
      url: `${SITE_URL}/auth/signin`,
      lastModified: now,
      changeFrequency: 'monthly',
      priority: 0.5,
    },
    {
      url: `${SITE_URL}/auth/signup`,
      lastModified: now,
      changeFrequency: 'monthly',
      priority: 0.5,
    },
  ];

  return publicPages;
}
