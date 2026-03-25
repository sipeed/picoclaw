import { Script } from 'vm';
/**
 * Browser frame script evaluator.
 */
export default class BrowserFrameScriptEvaluator {
    /**
     * Evaluates code or a VM Script in the frame's context.
     *
     * @param frame Frame.
     * @param script Script.
     * @returns Result.
     */
    static evaluate(frame, script) {
        if (!frame.window) {
            throw new Error('The frame has been destroyed, the "window" property is not set.');
        }
        script = typeof script === 'string' ? new Script(script) : script;
        return script.runInContext(frame.window);
    }
}
//# sourceMappingURL=BrowserFrameScriptEvaluator.js.map