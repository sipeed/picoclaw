"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var BrowserErrorCaptureEnum;
(function (BrowserErrorCaptureEnum) {
    /** Happy DOM use try and catch when evaluating code, but will not be able to catch all errors and Promise rejections. This will decrease performance as using try and catch makes the execution significally slower. This is the default setting. */
    BrowserErrorCaptureEnum["tryAndCatch"] = "tryAndCatch";
    /** Happy DOM will add an event listener to the Node.js process to catch all errors and Promise rejections. This will not work in Jest and Vitest as it conflicts with their error listeners. */
    BrowserErrorCaptureEnum["processLevel"] = "processLevel";
    /** Error capturing is disabled. Errors and Promise rejections will be thrown. */
    BrowserErrorCaptureEnum["disabled"] = "disabled";
})(BrowserErrorCaptureEnum || (BrowserErrorCaptureEnum = {}));
exports.default = BrowserErrorCaptureEnum;
//# sourceMappingURL=BrowserErrorCaptureEnum.cjs.map