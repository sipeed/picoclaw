import CharacterData from '../character-data/CharacterData.cjs';
import NodeTypeEnum from '../node/NodeTypeEnum.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
/**
 * Processing instruction node interface.
 *
 * Reference: https://developer.mozilla.org/en-US/docs/Web/API/ProcessingInstruction.
 */
export default class ProcessingInstruction extends CharacterData {
    [PropertySymbol.nodeType]: NodeTypeEnum;
    [PropertySymbol.target]: string;
    /**
     * Returns target.
     *
     * @returns Target.
     */
    get target(): string;
}
//# sourceMappingURL=ProcessingInstruction.d.ts.map