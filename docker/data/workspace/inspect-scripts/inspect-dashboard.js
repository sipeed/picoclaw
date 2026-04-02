const { runPageInspection } = require('./lib/shared');
runPageInspection({ path: '/', name: 'Dashboard', needsLogin: true });
