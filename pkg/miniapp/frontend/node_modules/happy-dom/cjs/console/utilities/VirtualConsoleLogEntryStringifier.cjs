"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const VirtualConsoleLogTypeEnum_js_1 = __importDefault(require("../enums/VirtualConsoleLogTypeEnum.cjs"));
/**
 * Virtual console utility.
 */
class VirtualConsoleLogEntryStringifier {
    /**
     * Stringifies a log entry.
     *
     * @param logEntry Log entry.
     * @returns Stringified message.
     */
    static toString(logEntry) {
        if (this.isLogEntryCollapsed(logEntry)) {
            return '';
        }
        const tabbing = this.getLogEntryGroupTabbing(logEntry);
        let output = tabbing;
        for (const part of logEntry.message) {
            output += output !== '' && output !== tabbing ? ' ' : '';
            if (typeof part === 'object' &&
                (part === null || part.constructor.name === 'Object' || Array.isArray(part))) {
                try {
                    output += JSON.stringify(part);
                }
                catch (error) {
                    output += new Error('Failed to JSON stringify object in log entry.').stack.replace(/\n    at/gm, '\n    ' + tabbing + 'at');
                }
            }
            else if (typeof part === 'object' && part['message'] && part['stack']) {
                output += part['stack'].replace(/\n    at/gm, '\n    ' + tabbing + 'at');
            }
            else {
                output += this.getLogEntryIcon(logEntry) + String(part);
            }
        }
        return output + '\n';
    }
    /**
     * Gets the log entry icon.
     *
     * @param logEntry Log entry.
     * @returns Icon.
     */
    static getLogEntryIcon(logEntry) {
        switch (logEntry.type) {
            case VirtualConsoleLogTypeEnum_js_1.default.group:
                return '▼ ';
            case VirtualConsoleLogTypeEnum_js_1.default.groupCollapsed:
                return '▶ ';
        }
        return '';
    }
    /**
     * Gets the log entry group tabbing.
     *
     * @param logEntry Log entry.
     * @returns Tabbing.
     */
    static getLogEntryGroupTabbing(logEntry) {
        let tabs = '';
        let group = logEntry.type === VirtualConsoleLogTypeEnum_js_1.default.group ||
            logEntry.type === VirtualConsoleLogTypeEnum_js_1.default.groupCollapsed
            ? logEntry.group?.parent
            : logEntry.group;
        while (group) {
            tabs += '  ';
            group = group.parent;
        }
        return tabs;
    }
    /**
     * Checks if the log entry content is collapsed.
     *
     * @param logEntry Log entry.
     * @returns True if collapsed.
     */
    static isLogEntryCollapsed(logEntry) {
        let group = logEntry.type === VirtualConsoleLogTypeEnum_js_1.default.group ||
            logEntry.type === VirtualConsoleLogTypeEnum_js_1.default.groupCollapsed
            ? logEntry.group?.parent
            : logEntry.group;
        while (group) {
            if (group.collapsed) {
                return true;
            }
            group = group.parent;
        }
        return false;
    }
}
exports.default = VirtualConsoleLogEntryStringifier;
//# sourceMappingURL=VirtualConsoleLogEntryStringifier.cjs.map