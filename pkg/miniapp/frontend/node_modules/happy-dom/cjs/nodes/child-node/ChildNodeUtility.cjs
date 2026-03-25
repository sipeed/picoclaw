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
const DOMException_js_1 = __importDefault(require("../../exception/DOMException.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const Node_js_1 = __importDefault(require("../node/Node.cjs"));
/**
 * Child node utility.
 */
class ChildNodeUtility {
    /**
     * Removes the node from its parent.
     *
     * @param childNode Child node.
     */
    static remove(childNode) {
        if (childNode[PropertySymbol.parentNode]) {
            childNode[PropertySymbol.parentNode].removeChild(childNode);
        }
    }
    /**
     * The Node.replaceWith() method replaces this Node in the children list of its parent with a set of Node or DOMString objects.
     *
     * @param childNode Child node.
     * @param nodes List of Node or DOMString.
     */
    static replaceWith(childNode, ...nodes) {
        const parent = childNode[PropertySymbol.parentNode];
        if (!parent) {
            throw new DOMException_js_1.default('This element has no parent node.');
        }
        for (const node of nodes) {
            if (node instanceof Node_js_1.default) {
                parent.insertBefore(node, childNode);
            }
            else {
                parent.insertBefore(parent[PropertySymbol.ownerDocument].createTextNode(String(node)), childNode);
            }
        }
        parent.removeChild(childNode);
    }
    /**
     * Inserts a set of Node or DOMString objects in the children list of this ChildNode's parent, just before this ChildNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param childNode Child node.
     * @param nodes List of Node or DOMString.
     */
    static before(childNode, ...nodes) {
        const parent = childNode[PropertySymbol.parentNode];
        if (!parent) {
            return;
        }
        for (const node of nodes) {
            if (node instanceof Node_js_1.default) {
                parent.insertBefore(node, childNode);
            }
            else {
                parent.insertBefore(parent[PropertySymbol.ownerDocument].createTextNode(String(node)), childNode);
            }
        }
    }
    /**
     * Inserts a set of Node or DOMString objects in the children list of this ChildNode's parent, just after this ChildNode. DOMString objects are inserted as equivalent Text nodes.
     *
     * @param childNode Child node.
     * @param nodes List of Node or DOMString.
     */
    static after(childNode, ...nodes) {
        const parent = childNode[PropertySymbol.parentNode];
        if (!parent) {
            return;
        }
        const nextSibling = childNode.nextSibling;
        for (const node of nodes) {
            const insertedNode = node instanceof Node_js_1.default
                ? node
                : parent[PropertySymbol.ownerDocument].createTextNode(String(node));
            if (!nextSibling) {
                parent.appendChild(insertedNode);
            }
            else {
                parent.insertBefore(insertedNode, nextSibling);
            }
        }
    }
}
exports.default = ChildNodeUtility;
//# sourceMappingURL=ChildNodeUtility.cjs.map