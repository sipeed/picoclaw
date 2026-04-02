const { runPageInspection } = require('./lib/shared');

runPageInspection({
  path: '/add-ons/webchat-widget',
  name: 'Add-Ons Webchat Widget',
  needsLogin: true,
});
