const { runPageInspection } = require('./lib/shared');
runPageInspection({ path: '/change-password', name: 'Change Password', needsLogin: true });
