const { runPageInspection } = require('./lib/shared');
runPageInspection({ path: '/profile', name: 'Profile', needsLogin: true });
