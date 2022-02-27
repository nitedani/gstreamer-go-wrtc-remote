const { join } = require('path');
const { cwd } = require('process');

module.exports = {
  content: [
    join(cwd(), 'apps', 'webapp') + '/src/**/*.{js,jsx,ts,tsx}',
    join(cwd(), 'apps', 'webapp') + '/src/public/index.html',
  ],
  theme: {
    extend: {},
  },
  plugins: [],
};
