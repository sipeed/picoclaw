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
const NodeTypeEnum_js_1 = __importDefault(require("../node/NodeTypeEnum.cjs"));
/**
 * Attribute node interface.
 *
 * Reference: https://developer.mozilla.org/en-US/docs/Web/API/Attr.
 */
class Attr extends Node_js_1.default {
    [PropertySymbol.nodeType] = NodeTypeEnum_js_1.default.attributeNode;
    [PropertySymbol.namespaceURI] = null;
    [PropertySymbol.name] = null;
    [PropertySymbol.localName] = null;
    [PropertySymbol.prefix] = null;
    [PropertySymbol.value] = null;
    [PropertySymbol.specified] = true;
    [PropertySymbol.ownerElement] = null;
    /**
     * Returns specified.
     *
     * @returns Specified.
     */
    get specified() {
        return this[PropertySymbol.specified];
    }
    /**
     * Returns owner element.
     *
     * @returns Owner element.
     */
    get ownerElement() {
        return this[PropertySymbol.ownerElement];
    }
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value() {
        return this[PropertySymbol.value];
    }
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value) {
        this[PropertySymbol.value] = value;
    }
    /**
     * Returns name.
     *
     * @returns Name.
     */
    get name() {
        return this[PropertySymbol.name];
    }
    /**
     * Returns local name.
     *
     * @returns Local name.
     */
    get localName() {
        return this[PropertySymbol.localName];
    }
    /**
     * Returns prefix.
     *
     * @returns Prefix.
     */
    get prefix() {
        return this[PropertySymbol.prefix];
    }
    /**
     * @override
     */
    get textContent() {
        return this[PropertySymbol.value];
    }
    /**
     * Returns namespace URI.
     *
     * @returns Namespace URI.
     */
    get namespaceURI() {
        return this[PropertySymbol.namespaceURI];
    }
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep = false) {
        const clone = super[PropertySymbol.cloneNode](deep);
        clone[PropertySymbol.namespaceURI] = this[PropertySymbol.namespaceURI];
        clone[PropertySymbol.name] = this[PropertySymbol.name];
        clone[PropertySymbol.localName] = this[PropertySymbol.localName];
        clone[PropertySymbol.prefix] = this[PropertySymbol.prefix];
        clone[PropertySymbol.value] = this[PropertySymbol.value];
        clone[PropertySymbol.specified] = this[PropertySymbol.specified];
        return clone;
    }
}
exports.default = Attr;
//# sourceMappingURL=Attr.cjs.map