import CharacterData from '../character-data/CharacterData.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
import * as PropertySymbol from '../../PropertySymbol.js';

/**
 * Processing instruction node interface.
 *
 * Reference: https://developer.mozilla.org/en-US/docs/Web/API/ProcessingInstruction.
 */
export default class ProcessingInstruction extends CharacterData {
	public [PropertySymbol.nodeType] = NodeTypeEnum.processingInstructionNode;
	public [PropertySymbol.target]: string;

	/**
	 * Returns target.
	 *
	 * @returns Target.
	 */
	public get target(): string {
		return this[PropertySymbol.target];
	}
}
