const { runPageInspection } = require('./lib/shared');
runPageInspection({ path: '/add-ons', name: 'Add-Ons', needsLogin: true });
