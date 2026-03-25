"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const NodeFilter_js_1 = __importDefault(require("./NodeFilter.cjs"));
exports.default = {
    /* ELEMENT_NODE */
    1: NodeFilter_js_1.default.SHOW_ELEMENT,
    /* ATTRIBUTE_NODE */
    2: NodeFilter_js_1.default.SHOW_ATTRIBUTE,
    /* TEXT_NODE */
    3: NodeFilter_js_1.default.SHOW_TEXT,
    /* CDATA_SECTION_NODE */
    4: NodeFilter_js_1.default.SHOW_CDATA_SECTION,
    /* ENTITY_REFERENCE_NODE */
    5: NodeFilter_js_1.default.SHOW_ENTITY_REFERENCE,
    /* ENTITY_NODE */
    6: NodeFilter_js_1.default.SHOW_PROCESSING_INSTRUCTION,
    /* PROCESSING_INSTRUCTION_NODE */
    7: NodeFilter_js_1.default.SHOW_PROCESSING_INSTRUCTION,
    /* COMMENT_NODE */
    8: NodeFilter_js_1.default.SHOW_COMMENT,
    /* DOCUMENT_NODE */
    9: NodeFilter_js_1.default.SHOW_DOCUMENT,
    /* DOCUMENT_TYPE_NODE */
    10: NodeFilter_js_1.default.SHOW_DOCUMENT_TYPE,
    /* DOCUMENT_FRAGMENT_NODE */
    11: NodeFilter_js_1.default.SHOW_DOCUMENT_FRAGMENT,
    /* NOTATION_NODE */
    12: NodeFilter_js_1.default.SHOW_NOTATION
};
//# sourceMappingURL=NodeFilterMask.cjs.map