export declare const KEY: {};
import type { IPane, IPaneSizingEvent } from './index.js';
interface $$__sveltets_2_IsomorphicComponent<Props extends Record<string, any> = any, Events extends Record<string, any> = any, Slots extends Record<string, any> = any, Exports = {}, Bindings = string> {
    new (options: import('svelte').ComponentConstructorOptions<Props>): import('svelte').SvelteComponent<Props, Events, Slots> & {
        $$bindings?: Bindings;
    } & Exports;
    (internal: unknown, props: Props & {
        $$events?: Events;
        $$slots?: Slots;
    }): Exports & {
        $set?: any;
        $on?: any;
    };
    z_$$bindings?: Bindings;
}
type $$__sveltets_2_PropsWithChildren<Props, Slots> = Props & (Slots extends {
    default: any;
} ? Props extends Record<string, never> ? any : {
    children?: any;
} : {});
declare const Splitpanes: $$__sveltets_2_IsomorphicComponent<$$__sveltets_2_PropsWithChildren<{
    id?: string;
    horizontal?: boolean;
    pushOtherPanes?: boolean;
    dblClickSplitter?: boolean;
    rtl?: boolean | "auto";
    firstSplitter?: boolean;
    style?: string | null;
    theme?: string;
    class?: string;
}, {
    default: {};
}>, {
    /**
     * When clicking (or touching) a pane.
     *
     * Returns the clicked pane object with its dimensions.
     */
    'pane-click': CustomEvent<IPane>;
    /**
     * Fires when you click a splitter.
     *
     * Returns the next pane object (with its dimensions) directly after the clicked splitter.
     *
     * This event is only emitted if dragging did not occur between mousedown and mouseup.
     */
    'splitter-click': CustomEvent<IPane>;
    /**
     * Fires when the pane is maximized (ie. typically by double clicking the splitter).
     *
     * Returns the maximized pane object with its.
     */
    'pane-maximize': CustomEvent<IPane>;
    /**
     * Fires when splitpanes is ready.
     */
    ready: CustomEvent<void>;
    /**
     * Fires while resizing (on mousemove/touchmove).
     *
     * Returns the clicked pane object with its dimensions.
     */
    resize: CustomEvent<IPaneSizingEvent[]>;
    /**
     * Fires once when the resizing stops after user drag (on mouseup/touchend) or when adding or removing a pane.
     *
     * Returns the clicked pane object with its dimensions.
     */
    resized: CustomEvent<IPaneSizingEvent[]>;
    /**
     * Fires when a pane is added.
     */
    'pane-add': CustomEvent<{
        /**
         * The index of the added pane.
         */
        index: number;
        /**
         * The new array of panes after resize.
         */
        panes: IPaneSizingEvent[];
    }>;
    /**
     * Fires when a pane is removed.
     */
    'pane-remove': CustomEvent<{
        /**
         * The removed pane.
         */
        removed: IPane;
        /**
         * An array of all the remaining pane objects with their dimensions (after resize).
         */
        panes: IPaneSizingEvent[];
    }>;
}, {
    default: {};
}, {}, string>;
type Splitpanes = InstanceType<typeof Splitpanes>;
export default Splitpanes;
