const { runPageInspection } = require('./lib/shared');

const addonType = process.env.ADDON_TYPE;

if (!addonType) {
  console.error('Missing ADDON_TYPE env var. Example: ADDON_TYPE=slack');
  process.exit(1);
}

runPageInspection({
  path: `/add-ons/${addonType}/install`,
  name: `Add-Ons ${addonType} Install`,
  needsLogin: true,
});
