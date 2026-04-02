const { runPageInspection } = require('./lib/shared');
runPageInspection({ path: '/change-email', name: 'Change Email', needsLogin: true });
