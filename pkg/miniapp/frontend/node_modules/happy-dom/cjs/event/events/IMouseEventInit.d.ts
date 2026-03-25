import EventTarget from '../EventTarget.cjs';
import IUIEventInit from '../IUIEventInit.cjs';
export default interface IMouseEventInit extends IUIEventInit {
    screenX?: number;
    screenY?: number;
    clientX?: number;
    clientY?: number;
    ctrlKey?: boolean;
    shiftKey?: boolean;
    altKey?: boolean;
    metaKey?: boolean;
    movementX?: number;
    movementY?: number;
    button?: number;
    buttons?: number;
    relatedTarget?: EventTarget;
    region?: string;
}
//# sourceMappingURL=IMouseEventInit.d.ts.map