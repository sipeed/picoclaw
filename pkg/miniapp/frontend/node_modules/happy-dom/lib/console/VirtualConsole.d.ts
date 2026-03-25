import IVirtualConsolePrinter from './IVirtualConsolePrinter.js';
import { ConsoleConstructor } from 'console';
/**
 * Virtual Console.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/Console
 */
export default class VirtualConsole implements Console {
    #private;
    Console: ConsoleConstructor;
    /**
     * Constructor.
     *
     * @param printer Console printer.
     */
    constructor(printer: IVirtualConsolePrinter);
    /**
     * Writes an error message to the console if the assertion is false. If the assertion is true, nothing happens.
     *
     * @param assertion Assertion.
     * @param message Message.
     * @param args Arguments.
     */
    assert(assertion: boolean, message?: any, ...args: Array<object | string>): void;
    /**
     * Clears the console.
     */
    clear(): void;
    /**
     * Logs the number of times that this particular call to count() has been called.
     *
     * @param [label='default'] Label.
     */
    count(label?: string): void;
    /**
     * Resets the counter.
     *
     * @param [label='default'] Label.
     */
    countReset(label?: string): void;
    /**
     * Outputs a message to the web console at the "debug" log level.
     *
     * @param message Message.
     * @param args Arguments.
     */
    debug(message?: any, ...args: Array<object | string>): void;
    /**
     * Displays an interactive list of the properties of the specified JavaScript object.
     *
     * @param data Data.
     */
    dir(data: any): void;
    /**
     * Displays an interactive tree of the descendant elements of the specified XML/HTML element.
     *
     * @param data Data.
     */
    dirxml(data: any[]): void;
    /**
     * Outputs an error message to the console.
     *
     * @param message Message.
     * @param args Arguments.
     */
    error(message?: any, ...args: Array<object | string>): void;
    /**
     * Alias for error().
     *
     * @deprecated
     * @alias error()
     * @param args Arguments.
     */
    exception(...args: Array<object | string>): void;
    /**
     * Creates a new inline group in the console, causing any subsequent console messages to be indented by an additional level, until console.groupEnd() is called.
     *
     * @param [label] Label.
     */
    group(label?: string): void;
    /**
     * Creates a new inline group in the console, but prints it as collapsed, requiring the use of a disclosure button to expand it.
     *
     * @param [label] Label.
     */
    groupCollapsed(label?: string): void;
    /**
     * Exits the current inline group in the console.
     */
    groupEnd(): void;
    /**
     * Outputs an informational message to the console.
     *
     * @param message Message.
     * @param args Arguments.
     */
    info(message?: any, ...args: Array<object | string>): void;
    /**
     * Outputs a message to the console.
     *
     * @param message Message.
     * @param args Arguments.
     */
    log(message?: any, ...args: Array<object | string>): void;
    /**
     * Starts recording a performance profile.
     *
     * TODO: Implement this.
     */
    profile(): void;
    /**
     * Stops recording a performance profile.
     *
     * TODO: Implement this.
     */
    profileEnd(): void;
    /**
     * Displays tabular data as a table.
     *
     * @param data Data.
     */
    table(data: {
        [key: string]: number | string | boolean;
    } | string[]): void;
    /**
     * Starts a timer you can use to track how long an operation takes.
     *
     * @param [label=default] Label.
     */
    time(label?: string): void;
    /**
     * Stops a timer that was previously started by calling console.time().
     * The method logs the elapsed time in milliseconds.
     *
     * @param [label=default] Label.
     */
    timeEnd(label?: string): void;
    /**
     * Logs the current value of a timer that was previously started by calling console.time().
     * The method logs the elapsed time in milliseconds.
     *
     * @param [label=default] Label.
     * @param [args] Arguments.
     */
    timeLog(label?: string, ...args: Array<object | string>): void;
    /**
     * Adds a single marker to the browser's Performance tool.
     *
     * TODO: Implement this.
     */
    timeStamp(): void;
    /**
     * Outputs a stack trace to the console.
     *
     * @param message Message.
     * @param args Arguments.
     */
    trace(message?: any, ...args: Array<object | string>): void;
    /**
     * Outputs a warning message to the console.
     *
     * @param message Message.
     * @param args Arguments.
     */
    warn(message?: any, ...args: Array<object | string>): void;
}
//# sourceMappingURL=VirtualConsole.d.ts.map