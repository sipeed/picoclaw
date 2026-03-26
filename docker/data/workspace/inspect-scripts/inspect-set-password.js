const { runPageInspection } = require('./lib/shared');
runPageInspection({ path: '/auth/set-password', name: 'Set Password Page', needsLogin: false });
