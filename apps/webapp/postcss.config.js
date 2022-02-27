const { join } = require('path');
const { cwd } = require('process');

module.exports = {
  plugins: {
    tailwindcss: {
      config: join(cwd(), 'apps', 'webapp', 'tailwind.config.js'),
    },
    autoprefixer: {},
  },
};
