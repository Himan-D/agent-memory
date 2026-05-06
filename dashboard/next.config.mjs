/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  images: {
    remotePatterns: [
      {
        protocol: 'https',
        hostname: 'images.unsplash.com',
      },
    ],
  },
  // For production, use standalone output or run as server
  // output: 'standalone', // Uncomment for standalone build
};

export default nextConfig;
