const { runPageInspection } = require('./lib/shared');

runPageInspection({
  path: '/add-ons/webchat-widget/install',
  name: 'Add-Ons Webchat Widget Install',
  needsLogin: true,
});
