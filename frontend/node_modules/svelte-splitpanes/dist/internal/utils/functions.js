/**
 * Generate a callback that is safe to be called even if `callbackObjectGetter()` is nullish on that moment.
 *
 * In the case of the object is nullish, invoking the callback will do nothing.
 */
const carefullCallbackGenerator = (callbackObjectGetter, callbackName) => (value) => {
    const callbackObject = callbackObjectGetter();
    if (callbackObject !== null) {
        callbackObject[callbackName](value);
    }
};
/**
 * This is an source of callbacks that are safe to be called even if the object of `callbackObjectGetter()` is nullish on that moment.
 *
 * In the case of the object is nullish, invoking the callbacks will do nothing.
 */
export const carefullCallbackSource = (callbackObjectGetter) => 
//@ts-expect-error unassignable
carefullCallbackGenerator.bind(null, callbackObjectGetter);
