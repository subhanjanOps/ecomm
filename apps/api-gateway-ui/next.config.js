/** @type {import('next').NextConfig} */
// Prefer IPv4 loopback by default to avoid IPv6 (::1) connection attempts that
// can fail when the target binds only to IPv4. When running in Docker, set
// `NEXT_PUBLIC_API_BASE` to the gateway service address (e.g. `http://api-gateway:8080`).
const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://api-gateway:8080'

const nextConfig = {
  reactStrictMode: true,
  async rewrites() {
    return [
      { source: '/admin/:path*', destination: `${API_BASE}/admin/:path*` },
      { source: '/api/:path*', destination: `${API_BASE}/api/:path*` },
    ]
  },
}

module.exports = nextConfig
