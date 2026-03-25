"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 *
 */
class Touch {
    identifier;
    target;
    clientX;
    clientY;
    screenX;
    screenY;
    pageX;
    pageY;
    radiusX;
    radiusY;
    rotationAngle;
    force;
    /**
     * Constructor.
     *
     * @param [touchInit] Touch init.
     */
    constructor(touchInit) {
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
exports.default = Touch;
//# sourceMappingURL=Touch.cjs.map