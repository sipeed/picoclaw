#!/bin/bash
# If JOB_TYPE is set, delegate to the job entrypoint script
if [ -n "$JOB_TYPE" ]; then
    exec /bin/bash /home/picoclaw/entrypoint-job.sh
fi
# Otherwise run picoclaw normally (gateway, agent, etc.)
exec picoclaw "$@"
