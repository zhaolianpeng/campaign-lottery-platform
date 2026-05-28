/** @type {import('next').NextConfig} */
const nextConfig = {
  experimental: {
    externalDir: true,
  },
  reactStrictMode: true,
  transpilePackages: ['@campaign-lottery/payment-module'],
};

export default nextConfig;
