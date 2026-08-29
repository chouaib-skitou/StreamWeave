const fs = require('node:fs');

const version = process.argv[2];

if (!version || !/^\d+\.\d+\.\d+$/.test(version)) {
  throw new Error(`Invalid release version: ${version ?? '<missing>'}`);
}

fs.writeFileSync('VERSION', `${version}\n`, 'utf8');
