import CookieSameSiteEnum from './enums/CookieSameSiteEnum.js';
export default {
    // Required
    key: null,
    originURL: null,
    // Optional
    value: null,
    domain: '',
    path: '',
    expires: null,
    httpOnly: false,
    secure: false,
    sameSite: CookieSameSiteEnum.lax
};
//# sourceMappingURL=DefaultCookie.js.map