/** @type {import('next').NextConfig} */
const nextConfig = {
  experimental: {
    externalDir: true,
  },
  reactStrictMode: true,
  serverExternalPackages: ['@campaign-lottery/payment-module'],
};

export default nextConfig;
