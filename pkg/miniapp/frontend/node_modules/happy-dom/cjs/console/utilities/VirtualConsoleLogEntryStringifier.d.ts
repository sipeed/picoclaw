import IVirtualConsoleLogEntry from '../IVirtualConsoleLogEntry.cjs';
/**
 * Virtual console utility.
 */
export default class VirtualConsoleLogEntryStringifier {
    /**
     * Stringifies a log entry.
     *
     * @param logEntry Log entry.
     * @returns Stringified message.
     */
    static toString(logEntry: IVirtualConsoleLogEntry): string;
    /**
     * Gets the log entry icon.
     *
     * @param logEntry Log entry.
     * @returns Icon.
     */
    private static getLogEntryIcon;
    /**
     * Gets the log entry group tabbing.
     *
     * @param logEntry Log entry.
     * @returns Tabbing.
     */
    private static getLogEntryGroupTabbing;
    /**
     * Checks if the log entry content is collapsed.
     *
     * @param logEntry Log entry.
     * @returns True if collapsed.
     */
    private static isLogEntryCollapsed;
}
//# sourceMappingURL=VirtualConsoleLogEntryStringifier.d.ts.map