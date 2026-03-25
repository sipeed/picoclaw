import EventTarget from './EventTarget.cjs';
/**
 * Message port.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/MessagePort
 */
export default abstract class MessagePort extends EventTarget {
    /**
     * Sends a message from the port, and optionally, transfers ownership of objects to other browsing contexts.
     *
     * @param _message Message.
     * @param _transerList Transfer list.
     */
    postMessage(_message: unknown, _transerList: unknown[]): void;
    /**
     * Starts the sending of messages queued on the port.
     */
    start(): void;
    /**
     * Disconnects the port, so it is no longer active. This stops the flow of messages to that port.
     */
    close(): void;
}
//# sourceMappingURL=MessagePort.d.ts.map