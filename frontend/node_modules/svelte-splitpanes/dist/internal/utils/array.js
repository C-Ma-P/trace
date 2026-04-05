export function sumPartial(arr, start, end, valueFunction) {
    let sum = 0;
    for (let i = start; i < end; i++) {
        sum += valueFunction(arr[i], i);
    }
    return sum;
}
export function forEachPartial(arr, start, end, callback) {
    for (let i = start; i < end; i++) {
        callback(arr[i], i);
    }
}
