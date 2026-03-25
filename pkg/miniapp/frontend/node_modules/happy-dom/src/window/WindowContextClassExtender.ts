import BrowserWindow from './BrowserWindow.js';
import * as PropertySymbol from '../PropertySymbol.js';

import DocumentImplementation from '../nodes/document/Document.js';
import HTMLDocumentImplementation from '../nodes/html-document/HTMLDocument.js';
import XMLDocumentImplementation from '../nodes/xml-document/XMLDocument.js';
import DocumentFragmentImplementation from '../nodes/document-fragment/DocumentFragment.js';
import TextImplementation from '../nodes/text/Text.js';
import CommentImplementation from '../nodes/comment/Comment.js';
import ImageImplementation from '../nodes/html-image-element/Image.js';
import AudioImplementation from '../nodes/html-audio-element/Audio.js';
import NodeIteratorImplementation from '../tree-walker/NodeIterator.js';
import TreeWalkerImplementation from '../tree-walker/TreeWalker.js';
import MutationObserverImplementation from '../mutation-observer/MutationObserver.js';
import MessagePortImplementation from '../event/MessagePort.js';
import DataTransferImplementation from '../event/DataTransfer.js';
import DataTransferItemImplementation from '../event/DataTransferItem.js';
import DataTransferItemListImplementation from '../event/DataTransferItemList.js';
import XMLSerializerImplementation from '../xml-serializer/XMLSerializer.js';
import CSSStyleSheetImplementation from '../css/CSSStyleSheet.js';
import DOMExceptionImplementation from '../exception/DOMException.js';
import CSSUnitValueImplementation from '../css/CSSUnitValue.js';
import SelectionImplementation from '../selection/Selection.js';
import HeadersImplementation from '../fetch/Headers.js';
import RequestImplementation from '../fetch/Request.js';
import ResponseImplementation from '../fetch/Response.js';
import EventTargetImplementation from '../event/EventTarget.js';
import XMLHttpRequestUploadImplementation from '../xml-http-request/XMLHttpRequestUpload.js';
import XMLHttpRequestEventTargetImplementation from '../xml-http-request/XMLHttpRequestEventTarget.js';
import AbortControllerImplementation from '../fetch/AbortController.js';
import AbortSignalImplementation from '../fetch/AbortSignal.js';
import FormDataImplementation from '../form-data/FormData.js';
import PermissionsImplementation from '../permissions/Permissions.js';
import PermissionStatusImplementation from '../permissions/PermissionStatus.js';
import ClipboardItemImplementation from '../clipboard/ClipboardItem.js';
import XMLHttpRequestImplementation from '../xml-http-request/XMLHttpRequest.js';
import DOMParserImplementation from '../dom-parser/DOMParser.js';
import RangeImplementation from '../range/Range.js';
import VTTCueImplementation from '../nodes/html-media-element/VTTCue.js';
import TextTrackImplementation from '../nodes/html-media-element/TextTrack.js';
import TextTrackListImplementation from '../nodes/html-media-element/TextTrackList.js';
import TextTrackCueImplementation from '../nodes/html-media-element/TextTrackCue.js';
import RemotePlaybackImplementation from '../nodes/html-media-element/RemotePlayback.js';
import FileReaderImplementation from '../file/FileReader.js';
import MediaStreamImplementation from '../nodes/html-media-element/MediaStream.js';
import MediaStreamTrackImplementation from '../nodes/html-media-element/MediaStreamTrack.js';
import CanvasCaptureMediaStreamTrackImplementation from '../nodes/html-canvas-element/CanvasCaptureMediaStreamTrack.js';
import NamedNodeMapImplementation from '../nodes/element/NamedNodeMap.js';

/**
 * Extends classes with a "window" property, so that they internally can access it's Window context.
 *
 * By using WindowBrowserContext, the classes can get access to their Browser context, for accessing settings or navigating the browser.
 */
export default class WindowContextClassExtender {
	/**
	 * Extends classes with a "window" property.
	 *
	 * @param window Window.
	 */
	public static extendClasses(window: BrowserWindow): void {
		/* eslint-disable jsdoc/require-jsdoc */

		// Document
		class Document extends DocumentImplementation {}
		Document.prototype[PropertySymbol.window] = window;
		(<typeof Document>window.Document) = Document;

		// HTMLDocument
		class HTMLDocument extends HTMLDocumentImplementation {}
		HTMLDocument.prototype[PropertySymbol.window] = window;
		(<typeof HTMLDocument>window.HTMLDocument) = HTMLDocument;

		// XMLDocument
		class XMLDocument extends XMLDocumentImplementation {}
		XMLDocument.prototype[PropertySymbol.window] = window;
		(<typeof XMLDocument>window.XMLDocument) = XMLDocument;

		// DocumentFragment
		class DocumentFragment extends DocumentFragmentImplementation {}
		DocumentFragment.prototype[PropertySymbol.window] = window;
		(<typeof DocumentFragment>window.DocumentFragment) = DocumentFragment;

		// Text
		class Text extends TextImplementation {}
		Text.prototype[PropertySymbol.window] = window;
		(<typeof Text>window.Text) = Text;

		// Comment
		class Comment extends CommentImplementation {}
		Comment.prototype[PropertySymbol.window] = window;
		(<typeof Comment>window.Comment) = Comment;

		// Image
		class Image extends ImageImplementation {}
		Image.prototype[PropertySymbol.window] = window;
		(<typeof Image>window.Image) = Image;

		// Audio
		class Audio extends AudioImplementation {}
		Audio.prototype[PropertySymbol.window] = window;
		(<typeof Audio>window.Audio) = Audio;

		// NodeIterator
		class NodeIterator extends NodeIteratorImplementation {}
		NodeIterator.prototype[PropertySymbol.window] = window;
		(<typeof NodeIterator>window.NodeIterator) = NodeIterator;

		// TreeWalker
		class TreeWalker extends TreeWalkerImplementation {}
		TreeWalker.prototype[PropertySymbol.window] = window;
		(<typeof TreeWalker>window.TreeWalker) = TreeWalker;

		// MutationObserver
		class MutationObserver extends MutationObserverImplementation {}
		MutationObserver.prototype[PropertySymbol.window] = window;
		(<typeof MutationObserver>window.MutationObserver) = MutationObserver;

		// MessagePort
		class MessagePort extends MessagePortImplementation {}
		MessagePort.prototype[PropertySymbol.window] = window;
		(<typeof MessagePort>window.MessagePort) = MessagePort;

		// DataTransfer
		class DataTransfer extends DataTransferImplementation {}
		DataTransfer.prototype[PropertySymbol.window] = window;
		(<typeof DataTransfer>window.DataTransfer) = DataTransfer;

		// DataTransferItem
		class DataTransferItem extends DataTransferItemImplementation {}
		DataTransferItem.prototype[PropertySymbol.window] = window;
		(<typeof DataTransferItem>window.DataTransferItem) = DataTransferItem;

		// DataTransferItemList
		class DataTransferItemList extends DataTransferItemListImplementation {}
		DataTransferItemList.prototype[PropertySymbol.window] = window;
		(<typeof DataTransferItemList>window.DataTransferItemList) = DataTransferItemList;

		// XMLSerializer
		class XMLSerializer extends XMLSerializerImplementation {}
		XMLSerializer.prototype[PropertySymbol.window] = window;
		(<typeof XMLSerializer>window.XMLSerializer) = XMLSerializer;

		// CSSStyleSheet
		class CSSStyleSheet extends CSSStyleSheetImplementation {}
		CSSStyleSheet.prototype[PropertySymbol.window] = window;
		(<typeof CSSStyleSheet>window.CSSStyleSheet) = CSSStyleSheet;

		// DOMException
		class DOMException extends DOMExceptionImplementation {}
		(<typeof DOMException>window.DOMException) = DOMException;

		// CSSUnitValue
		class CSSUnitValue extends CSSUnitValueImplementation {}
		CSSUnitValue.prototype[PropertySymbol.window] = window;
		(<typeof CSSUnitValue>window.CSSUnitValue) = CSSUnitValue;

		// Selection
		class Selection extends SelectionImplementation {}
		Selection.prototype[PropertySymbol.window] = window;
		(<typeof Selection>window.Selection) = Selection;

		// Headers
		class Headers extends HeadersImplementation {}
		Headers.prototype[PropertySymbol.window] = window;
		(<typeof Headers>window.Headers) = Headers;

		// Request
		class Request extends RequestImplementation {}
		Request.prototype[PropertySymbol.window] = window;
		(<typeof Request>window.Request) = Request;

		// Response
		class Response extends ResponseImplementation {}
		Response.prototype[PropertySymbol.window] = window;
		Response[PropertySymbol.window] = window;
		(<typeof Response>window.Response) = Response;

		// XMLHttpRequestEventTarget
		class EventTarget extends EventTargetImplementation {}
		EventTarget.prototype[PropertySymbol.window] = window;
		(<typeof EventTarget>window.EventTarget) = EventTarget;

		// XMLHttpRequestUpload
		class XMLHttpRequestUpload extends XMLHttpRequestUploadImplementation {}
		XMLHttpRequestUpload.prototype[PropertySymbol.window] = window;
		(<typeof XMLHttpRequestUpload>window.XMLHttpRequestUpload) = XMLHttpRequestUpload;

		// XMLHttpRequestEventTarget
		class XMLHttpRequestEventTarget extends XMLHttpRequestEventTargetImplementation {}
		XMLHttpRequestEventTarget.prototype[PropertySymbol.window] = window;
		(<typeof XMLHttpRequestEventTarget>window.XMLHttpRequestEventTarget) =
			XMLHttpRequestEventTarget;

		// AbortController
		class AbortController extends AbortControllerImplementation {}
		AbortController.prototype[PropertySymbol.window] = window;
		(<typeof AbortController>window.AbortController) = AbortController;

		// AbortSignal
		class AbortSignal extends AbortSignalImplementation {}
		AbortSignal.prototype[PropertySymbol.window] = window;
		AbortSignal[PropertySymbol.window] = window;
		(<typeof AbortSignal>window.AbortSignal) = AbortSignal;

		// FormData
		class FormData extends FormDataImplementation {}
		FormData.prototype[PropertySymbol.window] = window;
		(<typeof FormData>window.FormData) = FormData;

		// Permissions
		class Permissions extends PermissionsImplementation {}
		Permissions.prototype[PropertySymbol.window] = window;
		(<typeof Permissions>window.Permissions) = Permissions;

		// PermissionStatus
		class PermissionStatus extends PermissionStatusImplementation {}
		PermissionStatus.prototype[PropertySymbol.window] = window;
		(<typeof PermissionStatus>window.PermissionStatus) = PermissionStatus;

		// ClipboardItem
		class ClipboardItem extends ClipboardItemImplementation {}
		ClipboardItem.prototype[PropertySymbol.window] = window;
		(<typeof ClipboardItem>window.ClipboardItem) = ClipboardItem;

		// XMLHttpRequest
		class XMLHttpRequest extends XMLHttpRequestImplementation {}
		XMLHttpRequest.prototype[PropertySymbol.window] = window;
		(<typeof XMLHttpRequest>window.XMLHttpRequest) = XMLHttpRequest;

		// DOMParser
		class DOMParser extends DOMParserImplementation {}
		DOMParser.prototype[PropertySymbol.window] = window;
		(<typeof DOMParser>window.DOMParser) = DOMParser;

		// Range
		class Range extends RangeImplementation {}
		Range.prototype[PropertySymbol.window] = window;
		(<typeof Range>window.Range) = Range;

		// VTTCue
		class VTTCue extends VTTCueImplementation {}
		VTTCue.prototype[PropertySymbol.window] = window;
		(<typeof VTTCue>window.VTTCue) = VTTCue;

		// TextTrack
		class TextTrack extends TextTrackImplementation {}
		TextTrack.prototype[PropertySymbol.window] = window;
		(<typeof TextTrack>window.TextTrack) = TextTrack;

		// TextTrackList
		class TextTrackList extends TextTrackListImplementation {}
		TextTrackList.prototype[PropertySymbol.window] = window;
		(<typeof TextTrackList>window.TextTrackList) = TextTrackList;

		// TextTrackCue
		class TextTrackCue extends TextTrackCueImplementation {}
		TextTrackCue.prototype[PropertySymbol.window] = window;
		(<typeof TextTrackCue>window.TextTrackCue) = TextTrackCue;

		// RemotePlayback
		class RemotePlayback extends RemotePlaybackImplementation {}
		RemotePlayback.prototype[PropertySymbol.window] = window;
		(<typeof RemotePlayback>window.RemotePlayback) = RemotePlayback;

		// FileReader
		class FileReader extends FileReaderImplementation {}
		FileReader.prototype[PropertySymbol.window] = window;
		(<typeof FileReader>window.FileReader) = FileReader;

		// MediaStream
		class MediaStream extends MediaStreamImplementation {}
		MediaStream.prototype[PropertySymbol.window] = window;
		(<typeof MediaStream>window.MediaStream) = MediaStream;

		// MediaStreamTrack
		class MediaStreamTrack extends MediaStreamTrackImplementation {}
		MediaStreamTrack.prototype[PropertySymbol.window] = window;
		(<typeof MediaStreamTrack>window.MediaStreamTrack) = MediaStreamTrack;

		// MediaStreamTrack
		class CanvasCaptureMediaStreamTrack extends CanvasCaptureMediaStreamTrackImplementation {}
		CanvasCaptureMediaStreamTrack.prototype[PropertySymbol.window] = window;
		(<typeof CanvasCaptureMediaStreamTrack>window.CanvasCaptureMediaStreamTrack) =
			CanvasCaptureMediaStreamTrack;

		// NamedNodeMap
		class NamedNodeMap extends NamedNodeMapImplementation {}
		NamedNodeMap.prototype[PropertySymbol.window] = window;
		(<typeof NamedNodeMap>window.NamedNodeMap) = NamedNodeMap;

		/* eslint-enable jsdoc/require-jsdoc */
	}
}
