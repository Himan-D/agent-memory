import { AllWindowObservables } from '../autocapture-plugin';
import { DataExtractor } from '../data-extractor';
export declare function trackExposure({ allObservables, onExposure, dataExtractor, exposureDuration, }: {
    allObservables: AllWindowObservables;
    onExposure: (elementPath: string) => void;
    dataExtractor: DataExtractor;
    exposureDuration?: number;
}): {
    unsubscribe: () => void;
    reset: () => void;
};
//# sourceMappingURL=track-exposure.d.ts.map