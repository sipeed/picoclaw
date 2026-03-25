import SVGGeometryElement from '../svg-geometry-element/SVGGeometryElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGPointList from '../../svg/SVGPointList.js';

/**
 * SVG Polyline Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGPolylineElement
 */
export default class SVGPolylineElement extends SVGGeometryElement {
	// Internal properties
	public [PropertySymbol.animatedPoints]: SVGPointList | null = null;
	public [PropertySymbol.points]: SVGPointList | null = null;

	/**
	 * Returns animated points.
	 *
	 * @returns Animated points.
	 */
	public get animatedPoints(): SVGPointList {
		if (!this[PropertySymbol.animatedPoints]) {
			this[PropertySymbol.animatedPoints] = new SVGPointList(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					readOnly: true,
					getAttribute: () => this.getAttribute('points'),
					setAttribute: () => {}
				}
			);
		}
		return this[PropertySymbol.animatedPoints];
	}

	/**
	 * Returns points.
	 *
	 * @returns Points.
	 */
	public get points(): SVGPointList {
		if (!this[PropertySymbol.points]) {
			this[PropertySymbol.points] = new SVGPointList(
				PropertySymbol.illegalConstructor,
				this[PropertySymbol.window],
				{
					getAttribute: () => this.getAttribute('points'),
					setAttribute: (value) => this.setAttribute('points', value)
				}
			);
		}
		return this[PropertySymbol.points];
	}
}
