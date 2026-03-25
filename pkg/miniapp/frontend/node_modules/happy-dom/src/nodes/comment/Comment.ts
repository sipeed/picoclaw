import CharacterData from '../character-data/CharacterData.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';

/**
 * Comment node.
 */
export default class Comment extends CharacterData {
	public [PropertySymbol.nodeType] = NodeTypeEnum.commentNode;
	public declare cloneNode: (deep?: boolean) => Comment;

	/**
	 * Node name.
	 *
	 * @returns Node name.
	 */
	public get nodeName(): string {
		return '#comment';
	}

	/**
	 * Converts to string.
	 *
	 * @returns String.
	 */
	public toString(): string {
		return '[object Comment]';
	}

	/**
	 * @override
	 */
	public override [PropertySymbol.cloneNode](deep = false): Comment {
		return <Comment>super[PropertySymbol.cloneNode](deep);
	}
}
