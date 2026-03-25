import CharacterData from '../character-data/CharacterData.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import NodeTypeEnum from '../node/NodeTypeEnum.js';
/**
 * Comment node.
 */
export default class Comment extends CharacterData {
    [PropertySymbol.nodeType]: NodeTypeEnum;
    cloneNode: (deep?: boolean) => Comment;
    /**
     * Node name.
     *
     * @returns Node name.
     */
    get nodeName(): string;
    /**
     * Converts to string.
     *
     * @returns String.
     */
    toString(): string;
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep?: boolean): Comment;
}
//# sourceMappingURL=Comment.d.ts.map