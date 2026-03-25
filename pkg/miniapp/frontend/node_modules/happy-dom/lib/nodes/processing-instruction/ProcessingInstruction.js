import CharacterData from '../character-data/CharacterData.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * Processing instruction node interface.
 *
 * Reference: https://developer.mozilla.org/en-US/docs/Web/API/ProcessingInstruction.
 */
export default class ProcessingInstruction extends CharacterData {
    [PropertySymbol.nodeType] = NodeTypeEnum.processingInstructionNode;
    [PropertySymbol.target];
    /**
     * Returns target.
     *
     * @returns Target.
     */
    get target() {
        return this[PropertySymbol.target];
    }
}
//# sourceMappingURL=ProcessingInstruction.js.map