import EventTarget from './EventTarget.js';
export default interface ITouchInit {
    identifier: number;
    target: EventTarget;
    clientX?: number;
    clientY?: number;
    screenX?: number;
    screenY?: number;
    pageX?: number;
    pageY?: number;
    radiusX?: number;
    radiusY?: number;
    rotationAngle?: number;
    force?: number;
}
//# sourceMappingURL=ITouchInit.d.ts.map