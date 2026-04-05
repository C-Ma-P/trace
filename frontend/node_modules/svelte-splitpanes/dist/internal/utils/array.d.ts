export declare function sumPartial<T>(arr: ReadonlyArray<T>, start: number, end: number, valueFunction: (element: T, index: number) => number): number;
export declare function forEachPartial<T>(arr: ReadonlyArray<T>, start: number, end: number, callback: (element: T, index: number) => void): void;
