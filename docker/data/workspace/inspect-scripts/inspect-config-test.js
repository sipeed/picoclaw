const { runPageInspection } = require('./lib/shared');
runPageInspection({ path: '/config-test', name: 'Config Test', needsLogin: true });
