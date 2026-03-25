import NodeFilter from './NodeFilter.js';
export default {
    /* ELEMENT_NODE */
    1: NodeFilter.SHOW_ELEMENT,
    /* ATTRIBUTE_NODE */
    2: NodeFilter.SHOW_ATTRIBUTE,
    /* TEXT_NODE */
    3: NodeFilter.SHOW_TEXT,
    /* CDATA_SECTION_NODE */
    4: NodeFilter.SHOW_CDATA_SECTION,
    /* ENTITY_REFERENCE_NODE */
    5: NodeFilter.SHOW_ENTITY_REFERENCE,
    /* ENTITY_NODE */
    6: NodeFilter.SHOW_PROCESSING_INSTRUCTION,
    /* PROCESSING_INSTRUCTION_NODE */
    7: NodeFilter.SHOW_PROCESSING_INSTRUCTION,
    /* COMMENT_NODE */
    8: NodeFilter.SHOW_COMMENT,
    /* DOCUMENT_NODE */
    9: NodeFilter.SHOW_DOCUMENT,
    /* DOCUMENT_TYPE_NODE */
    10: NodeFilter.SHOW_DOCUMENT_TYPE,
    /* DOCUMENT_FRAGMENT_NODE */
    11: NodeFilter.SHOW_DOCUMENT_FRAGMENT,
    /* NOTATION_NODE */
    12: NodeFilter.SHOW_NOTATION
};
//# sourceMappingURL=NodeFilterMask.js.map