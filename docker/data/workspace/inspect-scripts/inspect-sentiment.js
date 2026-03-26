const { runPageInspection } = require('./lib/shared');
runPageInspection({ path: '/sentiment', name: 'Sentiment Dashboard', needsLogin: true });
