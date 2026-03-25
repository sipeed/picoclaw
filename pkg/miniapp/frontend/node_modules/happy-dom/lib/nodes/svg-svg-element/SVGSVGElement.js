import SVGGraphicsElement from '../svg-graphics-element/SVGGraphicsElement.js';
import SVGRect from '../../svg/SVGRect.js';
import SVGPoint from '../../svg/SVGPoint.js';
import SVGLength from '../../svg/SVGLength.js';
import SVGAngle from '../../svg/SVGAngle.js';
import SVGNumber from '../../svg/SVGNumber.js';
import SVGTransform from '../../svg/SVGTransform.js';
import SVGAnimatedRect from '../../svg/SVGAnimatedRect.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedPreserveAspectRatio from '../../svg/SVGAnimatedPreserveAspectRatio.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import NodeList from '../node/NodeList.js';
import SVGMatrix from '../../svg/SVGMatrix.js';
import ParentNodeUtility from '../parent-node/ParentNodeUtility.js';
/**
 * SVGSVGElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGSVGElement
 */
export default class SVGSVGElement extends SVGGraphicsElement {
    // Internal properties
    [PropertySymbol.preserveAspectRatio] = null;
    [PropertySymbol.x] = null;
    [PropertySymbol.y] = null;
    [PropertySymbol.width] = null;
    [PropertySymbol.height] = null;
    [PropertySymbol.currentScale] = 1;
    [PropertySymbol.viewBox] = null;
    // Events
    onafterprint = null;
    onbeforeprint = null;
    onbeforeunload = null;
    ongamepadconnected = null;
    ongamepaddisconnected = null;
    onhashchange = null;
    onlanguagechange = null;
    onmessage = null;
    onmessageerror = null;
    onoffline = null;
    ononline = null;
    onpagehide = null;
    onpageshow = null;
    onpopstate = null;
    onrejectionhandled = null;
    onstorage = null;
    onunhandledrejection = null;
    onunload = null;
    /**
     * Returns preserve aspect ratio.
     *
     * @returns Preserve aspect ratio.
     */
    get preserveAspectRatio() {
        if (!this[PropertySymbol.preserveAspectRatio]) {
            this[PropertySymbol.preserveAspectRatio] = new SVGAnimatedPreserveAspectRatio(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('preserveAspectRatio'),
                setAttribute: (value) => this.setAttribute('preserveAspectRatio', value)
            });
        }
        return this[PropertySymbol.preserveAspectRatio];
    }
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height() {
        if (!this[PropertySymbol.height]) {
            this[PropertySymbol.height] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('height'),
                setAttribute: (value) => this.setAttribute('height', value)
            });
        }
        return this[PropertySymbol.height];
    }
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width() {
        if (!this[PropertySymbol.width]) {
            this[PropertySymbol.width] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('width'),
                setAttribute: (value) => this.setAttribute('width', value)
            });
        }
        return this[PropertySymbol.width];
    }
    /**
     * Returns x position.
     *
     * @returns X position.
     */
    get x() {
        if (!this[PropertySymbol.x]) {
            this[PropertySymbol.x] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('x'),
                setAttribute: (value) => this.setAttribute('x', value)
            });
        }
        return this[PropertySymbol.x];
    }
    /**
     * Returns y position.
     *
     * @returns Y position.
     */
    get y() {
        if (!this[PropertySymbol.y]) {
            this[PropertySymbol.y] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('y'),
                setAttribute: (value) => this.setAttribute('y', value)
            });
        }
        return this[PropertySymbol.y];
    }
    /**
     * Returns currentScale.
     *
     * @returns CurrentScale.
     */
    get currentScale() {
        return this[PropertySymbol.currentScale];
    }
    /**
     * Sets currentScale.
     *
     * @param currentScale CurrentScale.
     */
    set currentScale(currentScale) {
        const parsed = typeof currentScale !== 'number' ? parseFloat(String(currentScale)) : currentScale;
        if (isNaN(parsed)) {
            throw this[PropertySymbol.window].TypeError(`Failed to set the 'currentScale' property on 'SVGSVGElement': The provided float value is non-finite.`);
        }
        if (parsed < 1) {
            return;
        }
        this[PropertySymbol.currentScale] = parsed;
    }
    /**
     * Returns current translate.
     *
     * @returns SVG point.
     */
    get currentTranslate() {
        return new SVGPoint(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
    }
    /**
     * Returns view box.
     *
     * @returns View box.
     */
    get viewBox() {
        if (!this[PropertySymbol.viewBox]) {
            this[PropertySymbol.viewBox] = new SVGAnimatedRect(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('viewBox'),
                setAttribute: (value) => this.setAttribute('viewBox', value)
            });
        }
        return this[PropertySymbol.viewBox];
    }
    /**
     * Pauses animation.
     */
    pauseAnimations() { }
    /**
     * Unpauses animation.
     */
    unpauseAnimations() { }
    /**
     * Returns "true" if animation is paused.
     *
     * @returns "true" if animation is paused.
     */
    animationsPaused() {
        return false;
    }
    /**
     * Returns the current time in seconds relative to the start time for the current SVG document fragment.
     *
     * @returns Current time in seconds.
     */
    getCurrentTime() {
        return 0;
    }
    /**
     * Sets current time.
     *
     * @param _seconds Seconds.
     */
    setCurrentTime(_seconds) { }
    /**
     * Returns intersection list.
     *
     * @param _rect SVG Rect.
     * @param _element SVG Element.
     * @returns Intersection list.
     */
    getIntersectionList(_rect, _element) {
        return new NodeList(PropertySymbol.illegalConstructor, []);
    }
    /**
     * Returns enclousure list.
     *
     * @param _rect SVG Rect.
     * @param _element SVG Element.
     * @returns Enclousure list.
     */
    getEnclosureList(_rect, _element) {
        return new NodeList(PropertySymbol.illegalConstructor, []);
    }
    /**
     * Returns true if the rendered content of the given element intersects the supplied rectangle.
     *
     * @param _element SVG Element.
     * @param _rect SVG Rect.
     * @returns Intersection state.
     */
    checkIntersection(_element, _rect) {
        return false;
    }
    /**
     * Returns true if the rendered content of the given element is entirely contained within the supplied rectangle.
     *
     * @param _element SVG Element.
     * @param _rect SVG Rect.
     * @returns Enclousure state.
     */
    checkEnclosure(_element, _rect) {
        return false;
    }
    /**
     * Unselects any selected objects, including any selections of text strings and type-in bars.
     */
    deselectAll() { }
    /**
     * Returns a number.
     *
     * @returns Number.
     */
    createSVGNumber() {
        return new SVGNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
    }
    /**
     * Returns a length.
     *
     * @returns Length.
     */
    createSVGLength() {
        return new SVGLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
    }
    /**
     * Returns a angle.
     *
     * @returns Angle.
     */
    createSVGAngle() {
        return new SVGAngle(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
    }
    /**
     * Returns a point.
     *
     * @returns Point.
     */
    createSVGPoint() {
        return new SVGPoint(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
    }
    /**
     * Returns a matrix.
     *
     * @returns Matrix.
     */
    createSVGMatrix() {
        return new SVGMatrix(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
    }
    /**
     * Returns a rect.
     *
     * @returns Rect.
     */
    createSVGRect() {
        return new SVGRect(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
    }
    /**
     * Returns a transform.
     *
     * @returns Transform.
     */
    createSVGTransform() {
        return new SVGTransform(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
    }
    /**
     * Returns a transform from a matrix.
     *
     * @param matrix Matrix.
     */
    createSVGTransformFromMatrix(matrix) {
        const transform = new SVGTransform(PropertySymbol.illegalConstructor, this[PropertySymbol.window]);
        transform.setMatrix(matrix);
        return transform;
    }
    /**
     * Returns an elements by class name.
     *
     * @param className Tag name.
     * @returns Matching element.
     */
    getElementsByClassName(className) {
        return ParentNodeUtility.getElementsByClassName(this, className);
    }
    /**
     * Returns an elements by tag name.
     *
     * @param tagName Tag name.
     * @returns Matching element.
     */
    getElementsByTagName(tagName) {
        return ParentNodeUtility.getElementsByTagName(this, tagName);
    }
    /**
     * Returns an elements by tag name and namespace.
     *
     * @param namespaceURI Namespace URI.
     * @param tagName Tag name.
     * @returns Matching element.
     */
    getElementsByTagNameNS(namespaceURI, tagName) {
        return ParentNodeUtility.getElementsByTagNameNS(this, namespaceURI, tagName);
    }
    /**
     * Returns an element by ID.
     *
     * @param id ID.
     * @returns Matching element.
     */
    getElementById(id) {
        return ParentNodeUtility.getElementById(this, id);
    }
    /**
     * @override
     */
    [PropertySymbol.cloneNode](deep = false) {
        return super[PropertySymbol.cloneNode](deep);
    }
}
//# sourceMappingURL=SVGSVGElement.js.map