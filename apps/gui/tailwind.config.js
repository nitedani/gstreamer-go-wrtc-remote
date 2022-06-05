const { join } = require('path');
const { cwd } = require('process');

module.exports = {
  content: [
    join(cwd(), 'apps', 'gui') + '/src/**/*.{js,jsx,ts,tsx}',
    join(cwd(), 'apps', 'gui') + '/src/public/index.html',
  ],
  theme: {
    extend: {},
  },
  plugins: [],
};
