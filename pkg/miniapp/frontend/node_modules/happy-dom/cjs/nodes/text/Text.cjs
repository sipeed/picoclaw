"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const CharacterData_js_1 = __importDefault(require("../character-data/CharacterData.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../../exception/DOMExceptionNameEnum.cjs"));
const NodeTypeEnum_js_1 = __importDefault(require("../node/NodeTypeEnum.cjs"));
/**
 * Text node.
 */
class Text extends CharacterData_js_1.default {
    [PropertySymbol.nodeType] = NodeTypeEnum_js_1.default.textNode;
    [PropertySymbol.textAreaNode] = null;
    [PropertySymbol.styleNode] = null;
    /**
     * Node name.
     *
     * @returns Node name.
     */
    get nodeName() {
        return '#text';
    }
    /**
     * @override
     */
    get data() {
        return this[PropertySymbol.data];
    }
    /**
     * @override
     */
    set data(data) {
        super.data = data;
        if (this[PropertySymbol.textAreaNode]) {
            this[PropertySymbol.textAreaNode][PropertySymbol.resetSelection]();
        }
        if (this[PropertySymbol.styleNode]) {
            this[PropertySymbol.styleNode][PropertySymbol.updateSheet]();
        }
    }
    /**
     * Breaks the Text node into two nodes at the specified offset, keeping both nodes in the tree as siblings.
     *
     * @see https://dom.spec.whatwg.org/#dom-text-splittext
     * @param offset Offset.
     * @returns New text node.
     */
    splitText(offset) {
        const length = this[PropertySymbol.data].length;
        if (offset < 0 || offset > length) {
            throw new this[PropertySymbol.window].DOMException('The index is not in the allowed range.', DOMExceptionNameEnum_js_1.default.indexSizeError);
        }
        const count = length - offset;
        const newData = this.substringData(offset, count);
        const newNode = this[PropertySymbol.ownerDocument].createTextNode(newData);
        if (this[PropertySymbol.parentNode] !== null) {
            this[PropertySymbol.parentNode].insertBefore(newNode, this.nextSibling);
        }
        this.replaceData(offset, count, '');
        return newNode;
    }
    /**
     * Converts to string.
     *
     * @returns String.
     */
    toString() {
        return '[object Text]';
    }
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep = false) {
        return super[PropertySymbol.cloneNode](deep);
    }
    /**
     * @override
     */
    [PropertySymbol.connectedToNode]() {
        super[PropertySymbol.connectedToNode]();
        if (this[PropertySymbol.textAreaNode]) {
            this[PropertySymbol.textAreaNode][PropertySymbol.resetSelection]();
        }
        if (this[PropertySymbol.styleNode] && this[PropertySymbol.data]) {
            this[PropertySymbol.styleNode][PropertySymbol.updateSheet]();
        }
    }
    /**
     * @override
     */
    [PropertySymbol.disconnectedFromNode]() {
        if (this[PropertySymbol.textAreaNode]) {
            this[PropertySymbol.textAreaNode][PropertySymbol.resetSelection]();
        }
        if (this[PropertySymbol.styleNode] && this[PropertySymbol.data]) {
            this[PropertySymbol.styleNode][PropertySymbol.updateSheet]();
        }
        super[PropertySymbol.disconnectedFromNode]();
    }
}
exports.default = Text;
//# sourceMappingURL=Text.cjs.map