import PermissionNameEnum from './PermissionNameEnum.js';
/**
 * Permissions API.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/Permissions.
 */
export default class Permissions {
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
        if (!Object.values(PermissionNameEnum).includes(permissionDescriptor.name)) {
            throw new Error(`Failed to execute 'query' on 'Permissions': Failed to read the 'name' property from 'PermissionDescriptor': The provided value '${permissionDescriptor.name}' is not a valid enum value of type PermissionName.`);
        }
        this.#permissionStatus[permissionDescriptor.name] = new this.#window.PermissionStatus('granted');
        return this.#permissionStatus[permissionDescriptor.name];
    }
}
//# sourceMappingURL=Permissions.js.map