const { runPageInspection } = require('./lib/shared');
runPageInspection({ path: '/logs', name: 'Logs', needsLogin: true });
