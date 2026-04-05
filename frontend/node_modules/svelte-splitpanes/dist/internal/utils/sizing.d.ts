export interface SidesStart {
    left: number;
    top: number;
}
export interface SidesEnd {
    right: number;
    bottom: number;
}
export type Sides = SidesStart & SidesEnd;
export declare function pxToNumber(pxString: string | undefined): number | undefined;
export declare const getDimensionName: (horizontal: boolean) => "height" | "width";
