"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * Virtual console log level.
 *
 * @see https://console.spec.whatwg.org/#loglevel-severity
 */
var VirtualConsoleLogLevelEnum;
(function (VirtualConsoleLogLevelEnum) {
    VirtualConsoleLogLevelEnum[VirtualConsoleLogLevelEnum["log"] = 0] = "log";
    VirtualConsoleLogLevelEnum[VirtualConsoleLogLevelEnum["info"] = 1] = "info";
    VirtualConsoleLogLevelEnum[VirtualConsoleLogLevelEnum["warn"] = 2] = "warn";
    VirtualConsoleLogLevelEnum[VirtualConsoleLogLevelEnum["error"] = 3] = "error";
})(VirtualConsoleLogLevelEnum || (VirtualConsoleLogLevelEnum = {}));
exports.default = VirtualConsoleLogLevelEnum;
//# sourceMappingURL=VirtualConsoleLogLevelEnum.cjs.map