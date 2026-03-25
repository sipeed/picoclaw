/**
 * VTTRegion.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/VTTRegion
 */
export default class VTTRegion {
	public id: string = '';
	public width: number = 100;
	public lines: number = 3;
	public regionAnchorX: number = 0;
	public regionAnchorY: number = 100;
	public viewportAnchorX: number = 0;
	public viewportAnchorY: number = 100;
	public scroll: string = 'none';
}
