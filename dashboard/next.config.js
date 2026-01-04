/** @type {import('next').NextConfig} */
const nextConfig = {
  webpack: (config, { isServer }) => {
    // Handle sql.js WebAssembly
    config.experiments = {
      ...config.experiments,
      asyncWebAssembly: true,
    };

    // For sql.js to work in Node.js environment
    if (isServer) {
      config.externals.push({
        'sql.js': 'commonjs sql.js',
      });
    }

    return config;
  },
}

module.exports = nextConfig
