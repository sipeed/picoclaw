/**
 * Date utility for HTML input elements.
 */
export default class HTMLInputElementDateUtility {
    /**
     * Returns iso week number from given date
     *
     * @see https://stackoverflow.com/a/6117889
     * @param date Date or number.
     * @returns Iso-week string.
     */
    static dateIsoWeek(date: Date | number): string;
    /**
     * Returns a date object for monday of given iso week string (\d\d\d\d-W\d\d)
     *
     * @param isoWeek Iso-week string.
     * @returns Date.
     */
    static isoWeekDate(isoWeek: string): Date;
}
//# sourceMappingURL=HTMLInputElementDateUtility.d.ts.map