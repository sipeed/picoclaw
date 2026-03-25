import IUIEventInit from '../IUIEventInit.js';
import Touch from '../Touch.js';
export default interface ITouchEventInit extends IUIEventInit {
    touches?: Touch[] | null;
    targetTouches?: Touch[] | null;
    changedTouches?: Touch[] | null;
    ctrlKey?: boolean | null;
    shiftKey?: boolean | null;
    altKey?: boolean | null;
    metaKey?: boolean | null;
}
//# sourceMappingURL=ITouchEventInit.d.ts.map