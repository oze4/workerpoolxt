// Package workerpoolxt wraps gammazero/workerpool
/*
 * ------------------------------------------------------------------------------------------
 *  Many thanks to
 *              github.com/gammazero/workerpool & github.com/gammazero/deque
 *              github.com/cenkalti/backoff
 *
 *  Please give them a star on GitHub!
 *
 *  They're really doing all the heavy lifting for us
 * ------------------------------------------------------------------------------------------
 */
//
// Response Channel
//
// If our timeout has passed, return an error object, disregarding any response from
// job which timed out. Same reason we check for context error before placing results
// on the response chan. We have no idea what Job.Task will be doing..lets say sending
// a GET request. If the http request takes 11 seconds, but the timeout is 10 seconds,
// our hands are tied.
//
// * We cannot cancel in flight requests, whether long running http or simply `time.Sleep`.
// So we just ignore the response when it eventually comes on the 12th second.
//
// TODO: *
// ...
package workerpoolxt
