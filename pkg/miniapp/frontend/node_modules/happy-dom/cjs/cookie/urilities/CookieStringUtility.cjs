"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const CookieSameSiteEnum_js_1 = __importDefault(require("../enums/CookieSameSiteEnum.cjs"));
const DefaultCookie_js_1 = __importDefault(require("../DefaultCookie.cjs"));
/**
 * Cookie string.
 */
class CookieStringUtility {
    /**
     * Returns cookie.
     *
     * @param originURL Origin URL.
     * @param cookieString Cookie string.
     * @returns Cookie.
     */
    static stringToCookie(originURL, cookieString) {
        const parts = cookieString.split(';');
        const part = parts.shift();
        const index = part.indexOf('=');
        const key = index !== -1 ? part.slice(0, index).trim() : part.trim();
        const value = index !== -1 ? part.slice(index + 1).trim() : null;
        const cookie = Object.assign({}, DefaultCookie_js_1.default, {
            // Required
            key,
            value,
            originURL
        });
        // Invalid if key is empty.
        if (!cookie.key) {
            return null;
        }
        for (const part of parts) {
            const index = part.indexOf('=');
            const key = index !== -1 ? part.slice(0, index).trim().toLowerCase() : part.trim().toLowerCase();
            const value = index !== -1 ? part.slice(index + 1).trim() : '';
            switch (key) {
                case 'expires':
                    cookie.expires = new Date(value);
                    break;
                case 'max-age':
                    cookie.expires = new Date(parseInt(value, 10) * 1000 + Date.now());
                    break;
                case 'domain':
                    cookie.domain = value;
                    break;
                case 'path':
                    cookie.path = value[0] === '/' ? value : `/${value}`;
                    break;
                case 'httponly':
                    cookie.httpOnly = true;
                    break;
                case 'secure':
                    cookie.secure = true;
                    break;
                case 'samesite':
                    switch (value.toLowerCase()) {
                        case 'strict':
                            cookie.sameSite = CookieSameSiteEnum_js_1.default.strict;
                            break;
                        case 'lax':
                            cookie.sameSite = CookieSameSiteEnum_js_1.default.lax;
                            break;
                        case 'none':
                            cookie.sameSite = CookieSameSiteEnum_js_1.default.none;
                    }
                    break;
            }
        }
        const lowerKey = cookie.key.toLowerCase();
        // Invalid if __secure- prefix is used and cookie is not secure.
        if (lowerKey.startsWith('__secure-') && !cookie.secure) {
            return null;
        }
        // Invalid if __host- prefix is used and cookie is not secure, not on root path or has a domain.
        if (lowerKey.startsWith('__host-') &&
            (!cookie.secure || cookie.path !== '/' || cookie.domain)) {
            return null;
        }
        return cookie;
    }
    /**
     * Returns cookie string with key and value.
     *
     * @param cookies Cookies.
     * @returns Cookie string.
     */
    static cookiesToString(cookies) {
        const cookieString = [];
        for (const cookie of cookies) {
            if (cookie.value !== null) {
                cookieString.push(`${cookie.key}=${cookie.value}`);
            }
            else {
                cookieString.push(cookie.key);
            }
        }
        return cookieString.join('; ');
    }
}
exports.default = CookieStringUtility;
//# sourceMappingURL=CookieStringUtility.cjs.map