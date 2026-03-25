"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const vm_1 = require("vm");
/**
 * Browser frame script evaluator.
 */
class BrowserFrameScriptEvaluator {
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
        script = typeof script === 'string' ? new vm_1.Script(script) : script;
        return script.runInContext(frame.window);
    }
}
exports.default = BrowserFrameScriptEvaluator;
//# sourceMappingURL=BrowserFrameScriptEvaluator.cjs.map