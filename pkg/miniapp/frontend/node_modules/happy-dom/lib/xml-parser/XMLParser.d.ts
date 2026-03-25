import BrowserWindow from '../window/BrowserWindow.js';
import XMLDocument from '../nodes/xml-document/XMLDocument.js';
/**
 * XML parser.
 */
export default class XMLParser {
    private window;
    private rootNode;
    private nodeStack;
    private tagNameStack;
    private defaultNamespaceStack;
    private namespacePrefixStack;
    private startTagIndex;
    private markupRegExp;
    private lastIndex;
    private errorIndex;
    private nextElement;
    private nextTagName;
    private currentNode;
    private readState;
    private errorMessage;
    /**
     * Constructor.
     *
     * @param window Window.
     * @param [options] Options.
     * @param [options.mode] Mode. Defaults to "htmlFragment".
     * @param [options.evaluateScripts] Set to "true" to enable script execution
     */
    constructor(window: BrowserWindow);
    /**
     * Parses XML and returns an XML document containing nodes found.
     *
     * @param xml XML string.
     * @returns XML document.
     */
    parse(xml: string): XMLDocument;
    /**
     * Parses plain text.
     *
     * @param text Text.
     */
    private parsePlainText;
    /**
     * Parses processing instruction.
     *
     * @param text Text.
     */
    private parseProcessingInstruction;
    /**
     * Parses comment.
     *
     * @param comment Comment.
     */
    private parseComment;
    /**
     * Parses document type.
     *
     * @param text Text.
     */
    private parseDocumentType;
    /**
     * Parses start tag.
     *
     * @param tagName Tag name.
     */
    private parseStartTag;
    /**
     * Parses end of start tag.
     *
     * @param attributeString Attribute string.
     * @param isSelfClosed Is self closed.
     */
    private parseEndOfStartTag;
    /**
     * Parses end tag.
     *
     * @param tagName Tag name.
     * @returns True if the end tag was parsed, false otherwise.
     */
    private parseEndTag;
    /**
     * Parses XML document error.
     *
     * @param readXML XML that has been read.
     * @param errorMessage Error message.
     */
    private parseError;
    /**
     * Removes overflowing text nodes in the current node.
     *
     * This needs to be done for some errors.
     */
    private removeOverflowingTextNodes;
    /**
     * Returns document type.
     *
     * @param value Value.
     * @returns Document type.
     */
    private getDocumentType;
}
//# sourceMappingURL=XMLParser.d.ts.map