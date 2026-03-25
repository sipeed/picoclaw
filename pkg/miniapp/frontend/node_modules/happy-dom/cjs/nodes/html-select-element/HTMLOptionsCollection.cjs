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
const HTMLCollection_js_1 = __importDefault(require("../element/HTMLCollection.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const QuerySelector_js_1 = __importDefault(require("../../query-selector/QuerySelector.cjs"));
/**
 * HTML Options Collection.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLOptionsCollection.
 */
class HTMLOptionsCollection extends HTMLCollection_js_1.default {
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param ownerElement Select element.
     */
    constructor(illegalConstructorSymbol, ownerElement) {
        super(illegalConstructorSymbol, () => (QuerySelector_js_1.default.querySelectorAll(ownerElement, 'option')[PropertySymbol.items]));
        this[PropertySymbol.ownerElement] = ownerElement;
    }
    /**
     * Returns selectedIndex.
     *
     * @returns SelectedIndex.
     */
    get selectedIndex() {
        return this[PropertySymbol.ownerElement].selectedIndex;
    }
    /**
     * Sets selectedIndex.
     *
     * @param selectedIndex SelectedIndex.
     */
    set selectedIndex(selectedIndex) {
        this[PropertySymbol.ownerElement].selectedIndex = selectedIndex;
    }
    /**
     *
     * @param element
     * @param before
     */
    add(element, before) {
        this[PropertySymbol.ownerElement].add(element, before);
    }
    /**
     * Removes indexed element from collection.
     *
     * @param index Index.
     */
    remove(index) {
        this[PropertySymbol.ownerElement].remove(index);
    }
}
exports.default = HTMLOptionsCollection;
//# sourceMappingURL=HTMLOptionsCollection.cjs.map