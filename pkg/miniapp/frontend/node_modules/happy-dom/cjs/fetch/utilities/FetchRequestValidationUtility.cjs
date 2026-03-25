"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const DOMException_js_1 = __importDefault(require("../../exception/DOMException.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../../exception/DOMExceptionNameEnum.cjs"));
const VALID_REFERRER_POLICIES = [
    '',
    'no-referrer',
    'no-referrer-when-downgrade',
    'same-origin',
    'origin',
    'strict-origin',
    'origin-when-cross-origin',
    'strict-origin-when-cross-origin',
    'unsafe-url'
];
const VALID_REDIRECTS = ['error', 'manual', 'follow'];
const SUPPORTED_SCHEMAS = ['data:', 'http:', 'https:'];
const FORBIDDEN_REQUEST_METHODS = ['TRACE', 'TRACK', 'CONNECT'];
const REQUEST_METHOD_REGEXP = /^[A-Z]+$/;
/**
 * Fetch request validation utility.
 */
class FetchRequestValidationUtility {
    /**
     * Validates request method.
     *
     * @throws DOMException
     * @param request Request.
     */
    static validateMethod(request) {
        if (!request.method || FORBIDDEN_REQUEST_METHODS.includes(request.method)) {
            throw new DOMException_js_1.default(`'${request.method || ''}' is not a valid HTTP method.`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        if (!REQUEST_METHOD_REGEXP.test(request.method)) {
            throw new DOMException_js_1.default(`'${request.method}' HTTP method is unsupported.`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
    }
    /**
     * Validates request body.
     *
     * @throws DOMException
     * @param request Request.
     */
    static validateBody(request) {
        if (request.body && (request.method === 'GET' || request.method === 'HEAD')) {
            throw new DOMException_js_1.default(`Request with GET/HEAD method cannot have body.`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
    }
    /**
     * Validates request URL.
     *
     * @throws DOMException
     * @param url URL.
     */
    static validateURL(url) {
        if (url.username !== '' || url.password !== '') {
            throw new DOMException_js_1.default(`${url} is an url with embedded credentials.`, DOMExceptionNameEnum_js_1.default.notSupportedError);
        }
    }
    /**
     * Validates request referrer policy.
     *
     * @throws DOMException
     * @param referrerPolicy Referrer policy.
     */
    static validateReferrerPolicy(referrerPolicy) {
        if (!VALID_REFERRER_POLICIES.includes(referrerPolicy)) {
            throw new DOMException_js_1.default(`Invalid referrer policy "${referrerPolicy}".`, DOMExceptionNameEnum_js_1.default.syntaxError);
        }
    }
    /**
     * Validates request redirect.
     *
     * @throws DOMException
     * @param redirect Redirect.
     */
    static validateRedirect(redirect) {
        if (!VALID_REDIRECTS.includes(redirect)) {
            throw new DOMException_js_1.default(`Invalid redirect "${redirect}".`, DOMExceptionNameEnum_js_1.default.syntaxError);
        }
    }
    /**
     * Validates request redirect.
     *
     * @throws DOMException
     * @param request
     * @param redirect Redirect.
     */
    static validateSchema(request) {
        if (!SUPPORTED_SCHEMAS.includes(request[PropertySymbol.url].protocol)) {
            throw new DOMException_js_1.default(`Failed to fetch from "${request.url}": URL scheme "${request[PropertySymbol.url].protocol.replace(/:$/, '')}" is not supported.`, DOMExceptionNameEnum_js_1.default.notSupportedError);
        }
    }
}
exports.default = FetchRequestValidationUtility;
//# sourceMappingURL=FetchRequestValidationUtility.cjs.map