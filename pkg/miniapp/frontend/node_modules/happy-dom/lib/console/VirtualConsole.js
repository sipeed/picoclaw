import VirtualConsoleLogLevelEnum from './enums/VirtualConsoleLogLevelEnum.js';
import VirtualConsoleLogTypeEnum from './enums/VirtualConsoleLogTypeEnum.js';
/**
 * Virtual Console.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/Console
 */
export default class VirtualConsole {
    // This is needed as the interface for the NodeJS Console also have a reference to the ConsoleConstructor class as a property for some reason.
    // This is not part of the browser specs.
    Console;
    #printer;
    #count = {};
    #time = {};
    #groupID = 0;
    #groups = [];
    /**
     * Constructor.
     *
     * @param printer Console printer.
     */
    constructor(printer) {
        this.#printer = printer;
    }
    /**
     * Writes an error message to the console if the assertion is false. If the assertion is true, nothing happens.
     *
     * @param assertion Assertion.
     * @param message Message.
     * @param args Arguments.
     */
    assert(assertion, message, ...args) {
        if (!assertion) {
            this.#printer.print({
                type: VirtualConsoleLogTypeEnum.assert,
                level: VirtualConsoleLogLevelEnum.error,
                message: ['Assertion failed:', ...(message ? [message, ...args] : args)],
                group: this.#groups[this.#groups.length - 1] || null
            });
        }
    }
    /**
     * Clears the console.
     */
    clear() {
        this.#printer.clear();
    }
    /**
     * Logs the number of times that this particular call to count() has been called.
     *
     * @param [label='default'] Label.
     */
    count(label = 'default') {
        if (!this.#count[label]) {
            this.#count[label] = 0;
        }
        this.#count[label]++;
        this.#printer.print({
            type: VirtualConsoleLogTypeEnum.count,
            level: VirtualConsoleLogLevelEnum.info,
            message: [`${label}: ${this.#count[label]}`],
            group: this.#groups[this.#groups.length - 1] || null
        });
    }
    /**
     * Resets the counter.
     *
     * @param [label='default'] Label.
     */
    countReset(label = 'default') {
        delete this.#count[label];
        this.#printer.print({
            type: VirtualConsoleLogTypeEnum.countReset,
            level: VirtualConsoleLogLevelEnum.warn,
            message: [`${label}: 0`],
            group: this.#groups[this.#groups.length - 1] || null
        });
    }
    /**
     * Outputs a message to the web console at the "debug" log level.
     *
     * @param message Message.
     * @param args Arguments.
     */
    debug(message, ...args) {
        this.#printer.print({
            type: VirtualConsoleLogTypeEnum.debug,
            level: VirtualConsoleLogLevelEnum.log,
            message: message ? [message, ...args] : args,
            group: this.#groups[this.#groups.length - 1] || null
        });
    }
    /**
     * Displays an interactive list of the properties of the specified JavaScript object.
     *
     * @param data Data.
     */
    dir(data) {
        this.#printer.print({
            type: VirtualConsoleLogTypeEnum.dir,
            level: VirtualConsoleLogLevelEnum.log,
            message: [data],
            group: this.#groups[this.#groups.length - 1] || null
        });
    }
    /**
     * Displays an interactive tree of the descendant elements of the specified XML/HTML element.
     *
     * @param data Data.
     */
    dirxml(data) {
        this.#printer.print({
            type: VirtualConsoleLogTypeEnum.dirxml,
            level: VirtualConsoleLogLevelEnum.log,
            message: [data],
            group: this.#groups[this.#groups.length - 1] || null
        });
    }
    /**
     * Outputs an error message to the console.
     *
     * @param message Message.
     * @param args Arguments.
     */
    error(message, ...args) {
        this.#printer.print({
            type: VirtualConsoleLogTypeEnum.error,
            level: VirtualConsoleLogLevelEnum.error,
            message: message ? [message, ...args] : args,
            group: this.#groups[this.#groups.length - 1] || null
        });
    }
    /**
     * Alias for error().
     *
     * @deprecated
     * @alias error()
     * @param args Arguments.
     */
    exception(...args) {
        this.error(...args);
    }
    /**
     * Creates a new inline group in the console, causing any subsequent console messages to be indented by an additional level, until console.groupEnd() is called.
     *
     * @param [label] Label.
     */
    group(label) {
        this.#groupID++;
        const group = {
            id: this.#groupID,
            label: label || 'default',
            collapsed: false,
            parent: this.#groups[this.#groups.length - 1] || null
        };
        this.#groups.push(group);
        this.#printer.print({
            type: VirtualConsoleLogTypeEnum.group,
            level: VirtualConsoleLogLevelEnum.log,
            message: [label || 'default'],
            group
        });
    }
    /**
     * Creates a new inline group in the console, but prints it as collapsed, requiring the use of a disclosure button to expand it.
     *
     * @param [label] Label.
     */
    groupCollapsed(label) {
        this.#groupID++;
        const group = {
            id: this.#groupID,
            label: label || 'default',
            collapsed: true,
            parent: this.#groups[this.#groups.length - 1] || null
        };
        this.#groups.push(group);
        this.#printer.print({
            type: VirtualConsoleLogTypeEnum.groupCollapsed,
            level: VirtualConsoleLogLevelEnum.log,
            message: [label || 'default'],
            group
        });
    }
    /**
     * Exits the current inline group in the console.
     */
    groupEnd() {
        if (this.#groups.length === 0) {
            return;
        }
        this.#groups.pop();
    }
    /**
     * Outputs an informational message to the console.
     *
     * @param message Message.
     * @param args Arguments.
     */
    info(message, ...args) {
        this.#printer.print({
            type: VirtualConsoleLogTypeEnum.info,
            level: VirtualConsoleLogLevelEnum.info,
            message: message ? [message, ...args] : args,
            group: this.#groups[this.#groups.length - 1] || null
        });
    }
    /**
     * Outputs a message to the console.
     *
     * @param message Message.
     * @param args Arguments.
     */
    log(message, ...args) {
        this.#printer.print({
            type: VirtualConsoleLogTypeEnum.log,
            level: VirtualConsoleLogLevelEnum.log,
            message: message ? [message, ...args] : args,
            group: this.#groups[this.#groups.length - 1] || null
        });
    }
    /**
     * Starts recording a performance profile.
     *
     * TODO: Implement this.
     */
    profile() {
        throw new Error('Method not implemented.');
    }
    /**
     * Stops recording a performance profile.
     *
     * TODO: Implement this.
     */
    profileEnd() {
        throw new Error('Method not implemented.');
    }
    /**
     * Displays tabular data as a table.
     *
     * @param data Data.
     */
    table(data) {
        this.#printer.print({
            type: VirtualConsoleLogTypeEnum.table,
            level: VirtualConsoleLogLevelEnum.log,
            message: [data],
            group: this.#groups[this.#groups.length - 1] || null
        });
    }
    /**
     * Starts a timer you can use to track how long an operation takes.
     *
     * @param [label=default] Label.
     */
    time(label = 'default') {
        this.#time[label] = performance.now();
    }
    /**
     * Stops a timer that was previously started by calling console.time().
     * The method logs the elapsed time in milliseconds.
     *
     * @param [label=default] Label.
     */
    timeEnd(label = 'default') {
        const time = this.#time[label];
        if (time) {
            const duration = performance.now() - time;
            this.#printer.print({
                type: VirtualConsoleLogTypeEnum.timeEnd,
                level: VirtualConsoleLogLevelEnum.info,
                message: [`${label}: ${duration}ms - timer ended`],
                group: this.#groups[this.#groups.length - 1] || null
            });
        }
    }
    /**
     * Logs the current value of a timer that was previously started by calling console.time().
     * The method logs the elapsed time in milliseconds.
     *
     * @param [label=default] Label.
     * @param [args] Arguments.
     */
    timeLog(label = 'default', ...args) {
        const time = this.#time[label];
        if (time) {
            const duration = performance.now() - time;
            this.#printer.print({
                type: VirtualConsoleLogTypeEnum.timeLog,
                level: VirtualConsoleLogLevelEnum.info,
                message: [`${label}: ${duration}ms`, ...args],
                group: this.#groups[this.#groups.length - 1] || null
            });
        }
    }
    /**
     * Adds a single marker to the browser's Performance tool.
     *
     * TODO: Implement this.
     */
    timeStamp() {
        throw new Error('Method not implemented.');
    }
    /**
     * Outputs a stack trace to the console.
     *
     * @param message Message.
     * @param args Arguments.
     */
    trace(message, ...args) {
        this.#printer.print({
            type: VirtualConsoleLogTypeEnum.trace,
            level: VirtualConsoleLogLevelEnum.log,
            message: [
                ...(message ? [message, ...args] : args),
                new Error('stack').stack.replace('Error: stack', '')
            ],
            group: this.#groups[this.#groups.length - 1] || null
        });
    }
    /**
     * Outputs a warning message to the console.
     *
     * @param message Message.
     * @param args Arguments.
     */
    warn(message, ...args) {
        this.#printer.print({
            type: VirtualConsoleLogTypeEnum.warn,
            level: VirtualConsoleLogLevelEnum.warn,
            message: message ? [message, ...args] : args,
            group: this.#groups[this.#groups.length - 1] || null
        });
    }
}
//# sourceMappingURL=VirtualConsole.js.map