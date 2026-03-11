# Debugging PicoClaw

PicoClaw performs multiple complex interactions under the hood for every single request it receives—from routing messages and evaluating complexity, to executing tools and adapting to model failures. Being able to see exactly what is happening is crucial, not just for troubleshooting potential issues, but also for truly understanding how the agent operates.
## Starting PicoClaw in Debug Mode

To get detailed information about what the agent is doing (LLM requests, tool calls, message routing), you can start the PicoClaw gateway with the debug flag:

```bash
picoclaw gateway --debug
# or
picoclaw gateway -d
```

In this mode, the system will format the logs extensively and display previews of system prompts and tool execution results.

## Disabling Log Truncation (Full Logs)

By default, PicoClaw truncates very long strings (such as the *System Prompt* or large JSON output results) in the debug logs to keep the console readable.

If you need to inspect the complete output of a command or the exact payload sent to the LLM model, you can use the `--no-truncate` flag.

**Note:** This flag *only* works when combined with the `--debug` mode.

```bash
picoclaw gateway --debug --no-truncate

```

When this flag is active, the global truncation function is disabled. This is extremely useful for:

* Verifying the exact syntax of the messages sent to the provider.
* Reading the complete output of tools like `exec`, `web_fetch`, or `read_file`.
* Debugging the session history saved in memory.
