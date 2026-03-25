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
const NodeList_js_1 = __importDefault(require("../node/NodeList.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
/**
 * Utility for finding labels associated with a form element.
 */
class HTMLLabelElementUtility {
    /**
     * Returns label elements for a form element.
     *
     * @param element Element to get labels for.
     * @returns Label elements.
     */
    static getAssociatedLabelElements(element) {
        const id = element.id;
        let labels;
        if (id && element[PropertySymbol.isConnected]) {
            const rootNode = element[PropertySymbol.rootNode] ||
                element[PropertySymbol.ownerDocument];
            labels = (rootNode.querySelectorAll(`label[for="${id}"]`)[PropertySymbol.items]);
        }
        else {
            labels = [];
        }
        let parent = element[PropertySymbol.parentNode];
        while (parent) {
            if (parent['tagName'] === 'LABEL') {
                labels.push(parent);
                break;
            }
            parent = parent[PropertySymbol.parentNode];
        }
        return new NodeList_js_1.default(PropertySymbol.illegalConstructor, labels);
    }
}
exports.default = HTMLLabelElementUtility;
//# sourceMappingURL=HTMLLabelElementUtility.cjs.map