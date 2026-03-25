import EventTarget from './EventTarget.js';
import ITouchInit from './ITouchInit.js';
/**
 *
 */
export default class Touch {
    readonly identifier: number;
    readonly target: EventTarget;
    readonly clientX: number;
    readonly clientY: number;
    readonly screenX: number;
    readonly screenY: number;
    readonly pageX: number;
    readonly pageY: number;
    readonly radiusX: number;
    readonly radiusY: number;
    readonly rotationAngle: number;
    readonly force: number;
    /**
     * Constructor.
     *
     * @param [touchInit] Touch init.
     */
    constructor(touchInit: ITouchInit);
}
//# sourceMappingURL=Touch.d.ts.map