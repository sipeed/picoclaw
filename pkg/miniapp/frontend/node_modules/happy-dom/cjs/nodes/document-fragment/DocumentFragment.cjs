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
const Node_js_1 = __importDefault(require("../node/Node.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const QuerySelector_js_1 = __importDefault(require("../../query-selector/QuerySelector.cjs"));
const ParentNodeUtility_js_1 = __importDefault(require("../parent-node/ParentNodeUtility.cjs"));
const HTMLCollection_js_1 = __importDefault(require("../element/HTMLCollection.cjs"));
const NodeTypeEnum_js_1 = __importDefault(require("../node/NodeTypeEnum.cjs"));
/**
 * DocumentFragment.
 */
class DocumentFragment extends Node_js_1.default {
    [PropertySymbol.children] = null;
    [PropertySymbol.rootNode] = this;
    [PropertySymbol.nodeType] = NodeTypeEnum_js_1.default.documentFragmentNode;
    /**
     * Returns the document fragment children.
     */
    get children() {
        if (!this[PropertySymbol.children]) {
            const elements = this[PropertySymbol.elementArray];
            this[PropertySymbol.children] = new HTMLCollection_js_1.default(PropertySymbol.illegalConstructor, () => elements);
        }
        return this[PropertySymbol.children];
    }
    /**
     * Last element child.
     *
     * @returns Element.
     */
    get childElementCount() {
        return this[PropertySymbol.elementArray].length;
    }
    /**
     * First element child.
     *
     * @returns Element.
     */
    get firstElementChild() {
        return this[PropertySymbol.elementArray][0] ?? null;
    }
    /**
     * Last element child.
     *
     * @returns Element.
     */
    get lastElementChild() {
        const children = this[PropertySymbol.elementArray];
        return children[children.length - 1] ?? null;
    }
    /**
     * Get text value of children.
     *
     * @returns Text content.
     */
    get textContent() {
        let result = '';
        for (const childNode of this[PropertySymbol.nodeArray]) {
            if (childNode[PropertySymbol.nodeType] === NodeTypeEnum_js_1.default.elementNode ||
                childNode[PropertySymbol.nodeType] === NodeTypeEnum_js_1.default.textNode) {
                result += childNode.textContent;
            }
        }
        return result;
    }
    /**
     * Sets text content.
     *
     * @param textContent Text content.
     */
    set textContent(textContent) {
        const childNodes = this[PropertySymbol.nodeArray];
        while (childNodes.length) {
            this.removeChild(childNodes[0]);
        }
        if (textContent) {
            this.appendChild(this[PropertySymbol.ownerDocument].createTextNode(textContent));
        }
    }
    /**
     * Inserts a set of Node objects or DOMString objects after the last child of the ParentNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param nodes List of Node or DOMString.
     */
    append(...nodes) {
        ParentNodeUtility_js_1.default.append(this, ...nodes);
    }
    /**
     * Inserts a set of Node objects or DOMString objects before the first child of the ParentNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param nodes List of Node or DOMString.
     */
    prepend(...nodes) {
        ParentNodeUtility_js_1.default.prepend(this, ...nodes);
    }
    /**
     * Replaces the existing children of a node with a specified new set of children.
     *
     * @param nodes List of Node or DOMString.
     */
    replaceChildren(...nodes) {
        ParentNodeUtility_js_1.default.replaceChildren(this, ...nodes);
    }
    /**
     * Query CSS selector to find matching elments.
     *
     * @param selector CSS selector.
     * @returns Matching elements.
     */
    querySelectorAll(selector) {
        return QuerySelector_js_1.default.querySelectorAll(this, selector);
    }
    /**
     * Query CSS Selector to find a matching element.
     *
     * @param selector CSS selector.
     * @returns Matching element.
     */
    querySelector(selector) {
        return QuerySelector_js_1.default.querySelector(this, selector);
    }
    /**
     * Returns an element by ID.
     *
     * @param id ID.
     * @returns Matching element.
     */
    getElementById(id) {
        return ParentNodeUtility_js_1.default.getElementById(this, id);
    }
}
exports.default = DocumentFragment;
//# sourceMappingURL=DocumentFragment.cjs.map