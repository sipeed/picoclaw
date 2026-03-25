"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const PermissionNameEnum_js_1 = __importDefault(require("./PermissionNameEnum.cjs"));
/**
 * Permissions API.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/Permissions.
 */
class Permissions {
    #permissionStatus = {};
    #window;
    /**
     * Constructor.
     *
     * @param window Window.
     */
    constructor(window) {
        if (!window?.document) {
            new TypeError('Invalid constructor');
        }
        this.#window = window;
    }
    /**
     * Returns scroll restoration.
     *
     * @param permissionDescriptor Permission descriptor.
     * @param permissionDescriptor.name Permission name.
     * @param [permissionDescriptor.userVisibleOnly] User visible only.
     * @param [permissionDescriptor.sysex] Sysex.
     * @returns Permission status.
     */
    async query(permissionDescriptor) {
        if (this.#permissionStatus[permissionDescriptor.name]) {
            return this.#permissionStatus[permissionDescriptor.name];
        }
        if (!Object.values(PermissionNameEnum_js_1.default).includes(permissionDescriptor.name)) {
            throw new Error(`Failed to execute 'query' on 'Permissions': Failed to read the 'name' property from 'PermissionDescriptor': The provided value '${permissionDescriptor.name}' is not a valid enum value of type PermissionName.`);
        }
        this.#permissionStatus[permissionDescriptor.name] = new this.#window.PermissionStatus('granted');
        return this.#permissionStatus[permissionDescriptor.name];
    }
}
exports.default = Permissions;
//# sourceMappingURL=Permissions.cjs.map