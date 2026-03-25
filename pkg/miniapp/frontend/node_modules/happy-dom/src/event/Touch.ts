import EventTarget from './EventTarget.js';
import ITouchInit from './ITouchInit.js';

/**
 *
 */
export default class Touch {
	public readonly identifier: number;
	public readonly target: EventTarget;
	public readonly clientX: number;
	public readonly clientY: number;
	public readonly screenX: number;
	public readonly screenY: number;
	public readonly pageX: number;
	public readonly pageY: number;
	public readonly radiusX: number;
	public readonly radiusY: number;
	public readonly rotationAngle: number;
	public readonly force: number;

	/**
	 * Constructor.
	 *
	 * @param [touchInit] Touch init.
	 */
	constructor(touchInit: ITouchInit) {
		this.identifier = touchInit.identifier;
		this.target = touchInit.target;
		this.clientX = touchInit.clientX ?? 0;
		this.clientY = touchInit.clientY ?? 0;
		this.screenX = touchInit.screenX ?? 0;
		this.screenY = touchInit.screenY ?? 0;
		this.pageX = touchInit.pageX ?? 0;
		this.pageY = touchInit.pageY ?? 0;
		this.radiusX = touchInit.radiusX ?? 0;
		this.radiusY = touchInit.radiusY ?? 0;
		this.rotationAngle = touchInit.rotationAngle ?? 0;
		this.force = touchInit.force ?? 0;
	}
}
