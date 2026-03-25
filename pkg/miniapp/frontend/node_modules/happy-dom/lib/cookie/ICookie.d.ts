import CookieSameSiteEnum from './enums/CookieSameSiteEnum.js';
export default interface ICookie {
    key: string;
    originURL: URL;
    value: string | null;
    domain: string;
    path: string;
    expires: Date | null;
    httpOnly: boolean;
    secure: boolean;
    sameSite: CookieSameSiteEnum;
}
//# sourceMappingURL=ICookie.d.ts.map