/** @type {import('next').NextConfig} */
const nextConfig = {
  eslint: {
    ignoreDuringBuilds: true,
  },
  typescript: {
    ignoreBuildErrors: true,
  },
  reactStrictMode: true,
  experimental: {
    optimizePackageImports: [
      'lucide-react',
      'recharts',
      '@tanstack/react-query',
      'sonner',
      'date-fns',
    ],
  },
  compiler: {
    removeConsole: process.env.NODE_ENV === 'production',
  },
  images: {
    remotePatterns: [
      {
        protocol: 'https',
        hostname: 'images.unsplash.com',
      },
    ],
    formats: ['image/avif', 'image/webp'],
  },
  webpack(config, { isServer }) {
    if (!isServer) {
      config.optimization = config.optimization || {};
      config.optimization.splitChunks = {
        ...config.optimization.splitChunks,
        cacheGroups: {
          ...config.optimization.splitChunks?.cacheGroups,
          recharts: {
            test: /[\\/]node_modules[\\/](recharts|d3-*|victory|react-vis)[\\/]/,
            name: 'recharts-vendor',
            chunks: 'all',
            priority: 20,
          },
          forceGraph: {
            test: /[\\/]node_modules[\\/](react-force-graph|force-graph|three)[\\/]/,
            name: 'force-graph-vendor',
            chunks: 'all',
            priority: 20,
          },
        },
      };
    }
    return config;
  },
};

export default nextConfig;
