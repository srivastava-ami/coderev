package treesitter

import (
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// Test Stream rules (4 tests)

func TestStreamNotPipedReadStream(t *testing.T) {
	src := `const stream = fs.createReadStream('file.txt');`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(f, "nodejs_conventions.stream_not_piped") {
		t.Error("must flag unpipe createReadStream")
	}
}

func TestStreamNotPipedWithPipe(t *testing.T) {
	src := `fs.createReadStream('file.txt').pipe(process.stdout);`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(f, "nodejs_conventions.stream_not_piped") {
		t.Error("must NOT flag piped stream")
	}
}

func TestBackpressureIgnored(t *testing.T) {
	src := `stream.write(data);`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(f, "nodejs_conventions.backpressure_ignored") {
		t.Error("must flag write without return check")
	}
}

func TestBackpressureChecked(t *testing.T) {
	src := `const ok = stream.write(data);`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(f, "nodejs_conventions.backpressure_ignored") {
		t.Error("must NOT flag write with return check")
	}
}

func TestStreamErrorUnhandled(t *testing.T) {
	src := `const stream = fs.createReadStream('file.txt');`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(f, "nodejs_conventions.stream_error_unhandled") {
		t.Error("must flag stream without error handler")
	}
}

func TestStreamErrorHandled(t *testing.T) {
	src := `fs.createReadStream('file.txt').on('error', (err) => console.error(err));`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(f, "nodejs_conventions.stream_error_unhandled") {
		t.Error("must NOT flag stream with error handler")
	}
}

func TestStreamLeak(t *testing.T) {
	src := `stream.on('error', (err) => { console.log(err); });`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(f, "nodejs_conventions.stream_leak") {
		t.Error("must flag error handler without destroy()")
	}
}

func TestStreamLeakDestroyed(t *testing.T) {
	src := `stream.on('error', (err) => { stream.destroy(); });`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(f, "nodejs_conventions.stream_leak") {
		t.Error("must NOT flag stream with destroy() in error handler")
	}
}

// Test Event Emitter rules (3 tests)

func TestEventListenerLeak(t *testing.T) {
	src := `emitter.on('event', handler);`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(f, "nodejs_conventions.event_listener_leak") {
		t.Error("must flag listener without removal")
	}
}

func TestEventListenerRemoved(t *testing.T) {
	src := `emitter.on('event', handler); emitter.removeListener('event', handler);`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(f, "nodejs_conventions.event_listener_leak") {
		t.Error("must NOT flag listener with removal")
	}
}

func TestOnceVsOnReady(t *testing.T) {
	src := `server.on('ready', () => { console.log('ready'); });`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(f, "nodejs_conventions.once_vs_on") {
		t.Error("must flag .on() for 'ready' event")
	}
}

func TestOnceVsOnActuallyOnce(t *testing.T) {
	src := `server.once('ready', () => { console.log('ready'); });`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(f, "nodejs_conventions.once_vs_on") {
		t.Error("must NOT flag .once()")
	}
}

func TestErrorEventUnhandled(t *testing.T) {
	src := `const server = http.createServer((req, res) => {});`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(f, "nodejs_conventions.error_event_unhandled") {
		t.Error("must flag http.createServer without error handler")
	}
}

func TestErrorEventHandled(t *testing.T) {
	src := `http.createServer((req, res) => {}).on('error', (err) => {});`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(f, "nodejs_conventions.error_event_unhandled") {
		t.Error("must NOT flag server with error handler")
	}
}

// Test Async Pattern rules (4 tests)

func TestPromiseSwallowing(t *testing.T) {
	src := `fetch('https://api.example.com');`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(f, "nodejs_conventions.promise_swallowing") {
		t.Error("must flag fetch without .catch() or await")
	}
}

func TestPromiseHandled(t *testing.T) {
	src := `fetch('https://api.example.com').catch(err => console.error(err));`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(f, "nodejs_conventions.promise_swallowing") {
		t.Error("must NOT flag promise with .catch()")
	}
}

func TestPromiseAwait(t *testing.T) {
	src := `const data = await fetch('https://api.example.com');`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(f, "nodejs_conventions.promise_swallowing") {
		t.Error("must NOT flag promise with await")
	}
}

func TestAsyncIteratorIncomplete(t *testing.T) {
	src := `[Symbol.asyncIterator]() { return { next: async () => ({ done: true }) }; }`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(f, "nodejs_conventions.async_iterator_incomplete") {
		t.Error("must flag async iterator without return() method")
	}
}

func TestAsyncIteratorComplete(t *testing.T) {
	src := `[Symbol.asyncIterator]() { return { next: async () => ({ done: true }), return() { } }; }`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(f, "nodejs_conventions.async_iterator_incomplete") {
		t.Error("must NOT flag async iterator with return() method")
	}
}

func TestConcurrentOperationsUnbounded(t *testing.T) {
	src := `Promise.all(tasks);`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(f, "nodejs_conventions.concurrent_operations_unbounded") {
		t.Error("must flag Promise.all without size checks")
	}
}

func TestConcurrentOperationsBounded(t *testing.T) {
	src := `Promise.all(tasks.slice(0, limit));`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(f, "nodejs_conventions.concurrent_operations_unbounded") {
		t.Error("must NOT flag Promise.all with size limit")
	}
}

// Test Performance rules (3 tests)

func TestMemoryLeakTimers(t *testing.T) {
	src := `setInterval(() => { console.log('tick'); }, 1000);`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(f, "nodejs_conventions.memory_leak_timers") {
		t.Error("must flag setInterval without storing ID")
	}
}

func TestMemoryLeakTimersStored(t *testing.T) {
	src := `const timerId = setInterval(() => { console.log('tick'); }, 1000);`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(f, "nodejs_conventions.memory_leak_timers") {
		t.Error("must NOT flag setInterval with stored ID")
	}
}

func TestUnboundedBuffer(t *testing.T) {
	src := `queue.push(item);`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(f, "nodejs_conventions.unbounded_buffer") {
		t.Error("must flag .push() without size checks")
	}
}

func TestUnboundedBufferBounded(t *testing.T) {
	src := `if (queue.length < maxSize) queue.push(item);`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(f, "nodejs_conventions.unbounded_buffer") {
		t.Error("must NOT flag .push() with size check")
	}
}

func TestCpuBlockingReadFileSync(t *testing.T) {
	src := `const data = fs.readFileSync('file.txt', 'utf8');`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(f, "nodejs_conventions.cpu_blocking") {
		t.Error("must flag readFileSync blocking event loop")
	}
}

func TestCpuBlockingAsync(t *testing.T) {
	src := `const data = await fs.promises.readFile('file.txt', 'utf8');`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(f, "nodejs_conventions.cpu_blocking") {
		t.Error("must NOT flag async file read")
	}
}

// Test TypeScript support

func TestStreamNotPipedTypeScript(t *testing.T) {
	src := `const stream: fs.ReadStream = fs.createReadStream('file.txt');`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "nodejs_conventions.stream_not_piped") {
		t.Error("must flag unpipe createReadStream in TypeScript")
	}
}

// Test non-JavaScript languages (should be skipped)

func TestNodejsSkippedForGo(t *testing.T) {
	src := `stream.write(data)`
	f := findingsForSrc(t, src, analysis.LangGo)
	if hasRule(f, "nodejs_conventions.backpressure_ignored") {
		t.Error("must NOT check nodejs conventions in Go code")
	}
}

// Test Callback Hell (1 test)

func TestCallbackHellDeepNesting(t *testing.T) {
	src := `asyncFunc().then(r => asyncFunc2(r).then(r2 => asyncFunc3(r2).then(r3 => asyncFunc4(r3).then(r4 => asyncFunc5(r4)))));`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(f, "nodejs_conventions.callback_hell") {
		t.Error("must flag 4+ levels of .then() nesting")
	}
}

func TestCallbackNotHellShallowNesting(t *testing.T) {
	src := `asyncFunc().then(r => asyncFunc2(r)).then(r2 => asyncFunc3(r2));`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(f, "nodejs_conventions.callback_hell") {
		t.Error("must NOT flag <=3 levels of .then()")
	}
}
