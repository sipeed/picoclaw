const { runPageInspection } = require('./lib/shared');

const addonType = "twilio";

if (!addonType) {
  console.error('Missing ADDON_TYPE env var. Example: ADDON_TYPE=slack');
  process.exit(1);
}

runPageInspection({
  path: `/add-ons/${addonType}`,
  name: `Add-Ons ${addonType} Detail`,
  needsLogin: true,
});
