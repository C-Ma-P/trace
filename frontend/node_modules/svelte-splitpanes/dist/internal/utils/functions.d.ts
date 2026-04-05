/**
 * This is an source of callbacks that are safe to be called even if the object of `callbackObjectGetter()` is nullish on that moment.
 *
 * In the case of the object is nullish, invoking the callbacks will do nothing.
 */
export declare const carefullCallbackSource: <CallbacksObject extends object>(callbackObjectGetter: () => CallbacksObject | undefined) => (<Callback extends keyof CallbacksObject>(callbackName: Callback) => CallbacksObject[Callback]);
