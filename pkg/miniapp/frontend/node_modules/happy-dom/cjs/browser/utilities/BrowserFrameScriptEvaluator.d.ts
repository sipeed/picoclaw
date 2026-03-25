import IBrowserFrame from '../types/IBrowserFrame.cjs';
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
    static evaluate(frame: IBrowserFrame, script: string | Script): any;
}
//# sourceMappingURL=BrowserFrameScriptEvaluator.d.ts.map