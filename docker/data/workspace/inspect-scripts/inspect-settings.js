const { runPageInspection } = require('./lib/shared');
runPageInspection({ path: '/settings', name: 'Settings', needsLogin: true });
