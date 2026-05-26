import { Campaign, ICampaignParser, ClickIdParameters, ReferrerParameters, UTMParameters } from '../types/campaign';
export declare class CampaignParser implements ICampaignParser {
    parse(): Promise<Campaign>;
    getUtmParam(): UTMParameters;
    getReferrer(): ReferrerParameters;
    getClickIds(): ClickIdParameters;
}
//# sourceMappingURL=campaign-parser.d.ts.map