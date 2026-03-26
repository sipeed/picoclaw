const { runPageInspection } = require('./lib/shared');
runPageInspection({ path: '/organization', name: 'Organization', needsLogin: true });
