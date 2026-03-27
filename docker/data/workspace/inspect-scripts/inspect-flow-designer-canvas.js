const { runPageInspection } = require('./lib/shared');

const flowId = 206;

if (!flowId) {
  console.error('Missing FLOW_ID env var. Example: FLOW_ID=123');
  process.exit(1);
}

runPageInspection({
  path: `/flow-designer/${flowId}`,
  name: `Flow Designer Canvas ${flowId}`,
  needsLogin: true,
});
