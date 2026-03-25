import FormData from '../form-data/FormData.cjs';
import HistoryScrollRestorationEnum from './HistoryScrollRestorationEnum.cjs';
export default interface IHistoryItem {
    title: string | null;
    href: string;
    state: any | null;
    scrollRestoration: HistoryScrollRestorationEnum;
    method: string;
    formData: FormData | null;
    isCurrent: boolean;
}
//# sourceMappingURL=IHistoryItem.d.ts.map