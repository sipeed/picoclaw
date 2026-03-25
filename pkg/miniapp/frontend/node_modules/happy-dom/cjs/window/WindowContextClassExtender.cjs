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
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const Document_js_1 = __importDefault(require("../nodes/document/Document.cjs"));
const HTMLDocument_js_1 = __importDefault(require("../nodes/html-document/HTMLDocument.cjs"));
const XMLDocument_js_1 = __importDefault(require("../nodes/xml-document/XMLDocument.cjs"));
const DocumentFragment_js_1 = __importDefault(require("../nodes/document-fragment/DocumentFragment.cjs"));
const Text_js_1 = __importDefault(require("../nodes/text/Text.cjs"));
const Comment_js_1 = __importDefault(require("../nodes/comment/Comment.cjs"));
const Image_js_1 = __importDefault(require("../nodes/html-image-element/Image.cjs"));
const Audio_js_1 = __importDefault(require("../nodes/html-audio-element/Audio.cjs"));
const NodeIterator_js_1 = __importDefault(require("../tree-walker/NodeIterator.cjs"));
const TreeWalker_js_1 = __importDefault(require("../tree-walker/TreeWalker.cjs"));
const MutationObserver_js_1 = __importDefault(require("../mutation-observer/MutationObserver.cjs"));
const MessagePort_js_1 = __importDefault(require("../event/MessagePort.cjs"));
const DataTransfer_js_1 = __importDefault(require("../event/DataTransfer.cjs"));
const DataTransferItem_js_1 = __importDefault(require("../event/DataTransferItem.cjs"));
const DataTransferItemList_js_1 = __importDefault(require("../event/DataTransferItemList.cjs"));
const XMLSerializer_js_1 = __importDefault(require("../xml-serializer/XMLSerializer.cjs"));
const CSSStyleSheet_js_1 = __importDefault(require("../css/CSSStyleSheet.cjs"));
const DOMException_js_1 = __importDefault(require("../exception/DOMException.cjs"));
const CSSUnitValue_js_1 = __importDefault(require("../css/CSSUnitValue.cjs"));
const Selection_js_1 = __importDefault(require("../selection/Selection.cjs"));
const Headers_js_1 = __importDefault(require("../fetch/Headers.cjs"));
const Request_js_1 = __importDefault(require("../fetch/Request.cjs"));
const Response_js_1 = __importDefault(require("../fetch/Response.cjs"));
const EventTarget_js_1 = __importDefault(require("../event/EventTarget.cjs"));
const XMLHttpRequestUpload_js_1 = __importDefault(require("../xml-http-request/XMLHttpRequestUpload.cjs"));
const XMLHttpRequestEventTarget_js_1 = __importDefault(require("../xml-http-request/XMLHttpRequestEventTarget.cjs"));
const AbortController_js_1 = __importDefault(require("../fetch/AbortController.cjs"));
const AbortSignal_js_1 = __importDefault(require("../fetch/AbortSignal.cjs"));
const FormData_js_1 = __importDefault(require("../form-data/FormData.cjs"));
const Permissions_js_1 = __importDefault(require("../permissions/Permissions.cjs"));
const PermissionStatus_js_1 = __importDefault(require("../permissions/PermissionStatus.cjs"));
const ClipboardItem_js_1 = __importDefault(require("../clipboard/ClipboardItem.cjs"));
const XMLHttpRequest_js_1 = __importDefault(require("../xml-http-request/XMLHttpRequest.cjs"));
const DOMParser_js_1 = __importDefault(require("../dom-parser/DOMParser.cjs"));
const Range_js_1 = __importDefault(require("../range/Range.cjs"));
const VTTCue_js_1 = __importDefault(require("../nodes/html-media-element/VTTCue.cjs"));
const TextTrack_js_1 = __importDefault(require("../nodes/html-media-element/TextTrack.cjs"));
const TextTrackList_js_1 = __importDefault(require("../nodes/html-media-element/TextTrackList.cjs"));
const TextTrackCue_js_1 = __importDefault(require("../nodes/html-media-element/TextTrackCue.cjs"));
const RemotePlayback_js_1 = __importDefault(require("../nodes/html-media-element/RemotePlayback.cjs"));
const FileReader_js_1 = __importDefault(require("../file/FileReader.cjs"));
const MediaStream_js_1 = __importDefault(require("../nodes/html-media-element/MediaStream.cjs"));
const MediaStreamTrack_js_1 = __importDefault(require("../nodes/html-media-element/MediaStreamTrack.cjs"));
const CanvasCaptureMediaStreamTrack_js_1 = __importDefault(require("../nodes/html-canvas-element/CanvasCaptureMediaStreamTrack.cjs"));
const NamedNodeMap_js_1 = __importDefault(require("../nodes/element/NamedNodeMap.cjs"));
/**
 * Extends classes with a "window" property, so that they internally can access it's Window context.
 *
 * By using WindowBrowserContext, the classes can get access to their Browser context, for accessing settings or navigating the browser.
 */
class WindowContextClassExtender {
    /**
     * Extends classes with a "window" property.
     *
     * @param window Window.
     */
    static extendClasses(window) {
        /* eslint-disable jsdoc/require-jsdoc */
        // Document
        class Document extends Document_js_1.default {
        }
        Document.prototype[PropertySymbol.window] = window;
        window.Document = Document;
        // HTMLDocument
        class HTMLDocument extends HTMLDocument_js_1.default {
        }
        HTMLDocument.prototype[PropertySymbol.window] = window;
        window.HTMLDocument = HTMLDocument;
        // XMLDocument
        class XMLDocument extends XMLDocument_js_1.default {
        }
        XMLDocument.prototype[PropertySymbol.window] = window;
        window.XMLDocument = XMLDocument;
        // DocumentFragment
        class DocumentFragment extends DocumentFragment_js_1.default {
        }
        DocumentFragment.prototype[PropertySymbol.window] = window;
        window.DocumentFragment = DocumentFragment;
        // Text
        class Text extends Text_js_1.default {
        }
        Text.prototype[PropertySymbol.window] = window;
        window.Text = Text;
        // Comment
        class Comment extends Comment_js_1.default {
        }
        Comment.prototype[PropertySymbol.window] = window;
        window.Comment = Comment;
        // Image
        class Image extends Image_js_1.default {
        }
        Image.prototype[PropertySymbol.window] = window;
        window.Image = Image;
        // Audio
        class Audio extends Audio_js_1.default {
        }
        Audio.prototype[PropertySymbol.window] = window;
        window.Audio = Audio;
        // NodeIterator
        class NodeIterator extends NodeIterator_js_1.default {
        }
        NodeIterator.prototype[PropertySymbol.window] = window;
        window.NodeIterator = NodeIterator;
        // TreeWalker
        class TreeWalker extends TreeWalker_js_1.default {
        }
        TreeWalker.prototype[PropertySymbol.window] = window;
        window.TreeWalker = TreeWalker;
        // MutationObserver
        class MutationObserver extends MutationObserver_js_1.default {
        }
        MutationObserver.prototype[PropertySymbol.window] = window;
        window.MutationObserver = MutationObserver;
        // MessagePort
        class MessagePort extends MessagePort_js_1.default {
        }
        MessagePort.prototype[PropertySymbol.window] = window;
        window.MessagePort = MessagePort;
        // DataTransfer
        class DataTransfer extends DataTransfer_js_1.default {
        }
        DataTransfer.prototype[PropertySymbol.window] = window;
        window.DataTransfer = DataTransfer;
        // DataTransferItem
        class DataTransferItem extends DataTransferItem_js_1.default {
        }
        DataTransferItem.prototype[PropertySymbol.window] = window;
        window.DataTransferItem = DataTransferItem;
        // DataTransferItemList
        class DataTransferItemList extends DataTransferItemList_js_1.default {
        }
        DataTransferItemList.prototype[PropertySymbol.window] = window;
        window.DataTransferItemList = DataTransferItemList;
        // XMLSerializer
        class XMLSerializer extends XMLSerializer_js_1.default {
        }
        XMLSerializer.prototype[PropertySymbol.window] = window;
        window.XMLSerializer = XMLSerializer;
        // CSSStyleSheet
        class CSSStyleSheet extends CSSStyleSheet_js_1.default {
        }
        CSSStyleSheet.prototype[PropertySymbol.window] = window;
        window.CSSStyleSheet = CSSStyleSheet;
        // DOMException
        class DOMException extends DOMException_js_1.default {
        }
        window.DOMException = DOMException;
        // CSSUnitValue
        class CSSUnitValue extends CSSUnitValue_js_1.default {
        }
        CSSUnitValue.prototype[PropertySymbol.window] = window;
        window.CSSUnitValue = CSSUnitValue;
        // Selection
        class Selection extends Selection_js_1.default {
        }
        Selection.prototype[PropertySymbol.window] = window;
        window.Selection = Selection;
        // Headers
        class Headers extends Headers_js_1.default {
        }
        Headers.prototype[PropertySymbol.window] = window;
        window.Headers = Headers;
        // Request
        class Request extends Request_js_1.default {
        }
        Request.prototype[PropertySymbol.window] = window;
        window.Request = Request;
        // Response
        class Response extends Response_js_1.default {
        }
        Response.prototype[PropertySymbol.window] = window;
        Response[PropertySymbol.window] = window;
        window.Response = Response;
        // XMLHttpRequestEventTarget
        class EventTarget extends EventTarget_js_1.default {
        }
        EventTarget.prototype[PropertySymbol.window] = window;
        window.EventTarget = EventTarget;
        // XMLHttpRequestUpload
        class XMLHttpRequestUpload extends XMLHttpRequestUpload_js_1.default {
        }
        XMLHttpRequestUpload.prototype[PropertySymbol.window] = window;
        window.XMLHttpRequestUpload = XMLHttpRequestUpload;
        // XMLHttpRequestEventTarget
        class XMLHttpRequestEventTarget extends XMLHttpRequestEventTarget_js_1.default {
        }
        XMLHttpRequestEventTarget.prototype[PropertySymbol.window] = window;
        window.XMLHttpRequestEventTarget =
            XMLHttpRequestEventTarget;
        // AbortController
        class AbortController extends AbortController_js_1.default {
        }
        AbortController.prototype[PropertySymbol.window] = window;
        window.AbortController = AbortController;
        // AbortSignal
        class AbortSignal extends AbortSignal_js_1.default {
        }
        AbortSignal.prototype[PropertySymbol.window] = window;
        AbortSignal[PropertySymbol.window] = window;
        window.AbortSignal = AbortSignal;
        // FormData
        class FormData extends FormData_js_1.default {
        }
        FormData.prototype[PropertySymbol.window] = window;
        window.FormData = FormData;
        // Permissions
        class Permissions extends Permissions_js_1.default {
        }
        Permissions.prototype[PropertySymbol.window] = window;
        window.Permissions = Permissions;
        // PermissionStatus
        class PermissionStatus extends PermissionStatus_js_1.default {
        }
        PermissionStatus.prototype[PropertySymbol.window] = window;
        window.PermissionStatus = PermissionStatus;
        // ClipboardItem
        class ClipboardItem extends ClipboardItem_js_1.default {
        }
        ClipboardItem.prototype[PropertySymbol.window] = window;
        window.ClipboardItem = ClipboardItem;
        // XMLHttpRequest
        class XMLHttpRequest extends XMLHttpRequest_js_1.default {
        }
        XMLHttpRequest.prototype[PropertySymbol.window] = window;
        window.XMLHttpRequest = XMLHttpRequest;
        // DOMParser
        class DOMParser extends DOMParser_js_1.default {
        }
        DOMParser.prototype[PropertySymbol.window] = window;
        window.DOMParser = DOMParser;
        // Range
        class Range extends Range_js_1.default {
        }
        Range.prototype[PropertySymbol.window] = window;
        window.Range = Range;
        // VTTCue
        class VTTCue extends VTTCue_js_1.default {
        }
        VTTCue.prototype[PropertySymbol.window] = window;
        window.VTTCue = VTTCue;
        // TextTrack
        class TextTrack extends TextTrack_js_1.default {
        }
        TextTrack.prototype[PropertySymbol.window] = window;
        window.TextTrack = TextTrack;
        // TextTrackList
        class TextTrackList extends TextTrackList_js_1.default {
        }
        TextTrackList.prototype[PropertySymbol.window] = window;
        window.TextTrackList = TextTrackList;
        // TextTrackCue
        class TextTrackCue extends TextTrackCue_js_1.default {
        }
        TextTrackCue.prototype[PropertySymbol.window] = window;
        window.TextTrackCue = TextTrackCue;
        // RemotePlayback
        class RemotePlayback extends RemotePlayback_js_1.default {
        }
        RemotePlayback.prototype[PropertySymbol.window] = window;
        window.RemotePlayback = RemotePlayback;
        // FileReader
        class FileReader extends FileReader_js_1.default {
        }
        FileReader.prototype[PropertySymbol.window] = window;
        window.FileReader = FileReader;
        // MediaStream
        class MediaStream extends MediaStream_js_1.default {
        }
        MediaStream.prototype[PropertySymbol.window] = window;
        window.MediaStream = MediaStream;
        // MediaStreamTrack
        class MediaStreamTrack extends MediaStreamTrack_js_1.default {
        }
        MediaStreamTrack.prototype[PropertySymbol.window] = window;
        window.MediaStreamTrack = MediaStreamTrack;
        // MediaStreamTrack
        class CanvasCaptureMediaStreamTrack extends CanvasCaptureMediaStreamTrack_js_1.default {
        }
        CanvasCaptureMediaStreamTrack.prototype[PropertySymbol.window] = window;
        window.CanvasCaptureMediaStreamTrack =
            CanvasCaptureMediaStreamTrack;
        // NamedNodeMap
        class NamedNodeMap extends NamedNodeMap_js_1.default {
        }
        NamedNodeMap.prototype[PropertySymbol.window] = window;
        window.NamedNodeMap = NamedNodeMap;
        /* eslint-enable jsdoc/require-jsdoc */
    }
}
exports.default = WindowContextClassExtender;
//# sourceMappingURL=WindowContextClassExtender.cjs.map