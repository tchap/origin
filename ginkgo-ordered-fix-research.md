# Ginkgo Ordered/BeforeAll Fix Research

## Problem

`g.Ordered` and `g.BeforeAll` in openshift/origin Ginkgo tests are effectively useless —
`BeforeAll` behaves like `BeforeEach`. There is even an existing comment in `oidc.go`
acknowledging this: *"currently g.Ordered and g.BeforeAll don't take effect in openshift-tests."*

## Root Cause

Each Ginkgo spec runs in its own isolated subprocess. Ginkgo's `BeforeAll` fires on every
`RunSpec` call because each invocation has no knowledge of sibling specs in the container.
`Ordered` ordering is meaningless because each spec is a separate process.

### Execution path

1. `test_runner.go:340` — outer orchestrator dispatches one test name at a time:
   ```go
   results := test.binary.RunTests(ctx, timeout, testEnv, test.name)
   ```

2. `binary.go:488` — `RunTests` spawns a subprocess:
   ```
   openshift-tests run-test -n <test-name>
   ```

3. `runtest.go:109` — calls `specs.Run(ctx, w, maxConcurrency)` with exactly 1 spec.

4. `spec.go:210` — since `len(specs) == 1`, calls `spec.Run(ctx)` directly in-process.

5. `ginkgo/util.go:83` — calls Ginkgo's single-spec API:
   ```go
   ginkgo.GetSuite().RunSpec(spec, ginkgo.Labels{}, "", cwd, ...)
   ```

6. Ginkgo runs `BeforeAll` as part of that spec's setup chain. Since only one spec is ever
   in this invocation, it runs on every call → **BeforeAll = BeforeEach**.

Ginkgo's `types.TestSpec` has `IsInOrderedContainer bool` (types.go:221), so ordered specs
are detectable during the tree walk.

## Fix Options

### Option A: Group Ordered container specs into a single OTE spec (recommended)

In `BuildExtensionTestSpecsFromOpenShiftGinkgoSuite` (`ginkgo/util.go`), detect
`spec.IsInOrderedContainer == true` during `WalkTests`. Group all specs sharing the same
Ordered container into one OTE `ExtensionTestSpec`. That spec's `Run` calls `RunSpec` for
each container spec in order — all within one process.

**Pros:**
- Preserves `BeforeAll`/`AfterAll` semantics exactly (fires once per container).
- Ordering is maintained.
- Entirely self-contained in origin — no upstream OTE changes needed.

**Cons:**
- The entire container appears as one JUnit test entry, not individual specs.
- Cannot independently retry a single failing spec inside the container.

**Implementation sketch:**
1. In `WalkTests`, detect `spec.IsInOrderedContainer` and extract the container hierarchy
   (from `spec.CodeLocations()` or spec text hierarchy).
2. Group specs by container path.
3. For each Ordered container group, produce one OTE `ExtensionTestSpec` whose `Run`
   iterates through all container specs calling `RunSpec` in sequence, emitting one JSONL
   result per spec (see JUnit implications below).

### Option B: Pass all Ordered container specs in a single `run-test` invocation

The orchestrator detects which tests belong to the same Ordered container and calls
`RunTests` with all container spec names together. OTE then runs them sequentially
in-process rather than spawning parallel subprocesses.

**Pros:** Individual spec names remain as separate JUnit entries.

**Cons:** Requires upstream OTE changes. High effort.

### Option C: Pragmatic workaround (no framework changes)

Make setup idempotent per-spec: check cluster state at the start of each `BeforeEach`
(e.g. check if OIDC is already configured before configuring it). Works today without any
framework changes but is verbose and error-prone.

## JUnit Granularity Implications

Collapsing an Ordered container into a single JUnit entry has real CI consequences:

- **Sippy / Component Readiness** — test pass rates are tracked by exact test name. One
  container entry means per-spec pass rate history is lost. A single flaky spec poisons
  the whole container's statistics.
- **Failure attribution** — the JUnit XML only records the container failed; finding the
  specific failing spec requires digging into log artifacts.
- **Retry granularity** — CI retries by test name. Retrying the "container" reruns all
  specs inside, even those that passed.
- **Flake thresholds** — Sippy flake detection is calibrated per-test-name. Container
  entries have higher variance, making thresholds less precise.

### Mitigation: multi-result JSONL output

`run-test` returns results as JSONL — one result object per test. A single OTE spec can
emit **multiple JSONL results**, one per container spec. This means Option A can run all
container specs in one process (preserving correct `BeforeAll`/`AfterAll` semantics) while
still emitting individual per-spec JUnit entries. Need to verify whether origin's
`run-test` result emitter already supports multi-result output or needs to be extended
(see investigation below).

## Emitter Investigation

**Verdict: multi-result output from a single OTE spec is not supported.**

Relevant code in `vendor/.../extensiontests/`:

- `types.go:57` — `Run func(ctx context.Context) *ExtensionTestResult` returns exactly one
  result.
- `spec.go:247` — `runSpec()` calls `spec.Run(ctx)`, gets one `*ExtensionTestResult`.
- `spec.go:262` — `res.Name = spec.Name` unconditionally overwrites the result name.
- `spec.go:263` — `w.Write(res)` is called once per spec.
- `result_writer.go:133-144` — `JSONResultWriter.Write()` emits one JSONL line per call.

There is no path for a single OTE spec's `Run` to emit more than one JSONL result. The
framework would need upstream changes to support it:
- Change `Run` to return `[]*ExtensionTestResult`, **or**
- Pass `ResultWriter` into `Run` so it can call `w.Write()` multiple times.

**Consequence for Option A:** the JUnit granularity loss is real and cannot be mitigated
without upstream OTE changes. Each Ordered container becomes one JUnit entry.

## Why Option A is still the right call

The JUnit granularity loss is defensible: tests that genuinely need `g.Ordered` +
`g.BeforeAll` are inherently sequential, stateful, and coupled. Treating the whole
container as one CI result accurately reflects that — it's one logical test scenario. If
one spec fails, the rest are meaningless anyway (that's the point of `Ordered`). Pursuing
upstream OTE changes for per-spec JUnit entries adds cross-repo dependency for marginal
benefit.

However, if per-spec JUnit entries are required, see the `RunMulti` extension below.

## Extending OTE for Multiple JUnit Entries (`RunMulti`)

To preserve per-spec JUnit granularity without the single-result constraint, OTE needs a
small additive upstream change.

### OTE change: add `RunMulti` field

**File:** `vendor/.../extensiontests/types.go`

Add an optional field alongside `Run`:

```go
// RunMulti is like Run but returns one result per sub-test. Used for grouped
// containers (e.g. Ginkgo Ordered) that must run in one process but produce
// individual JUnit entries.
RunMulti func(ctx context.Context) []*ExtensionTestResult `json:"-"`
```

Non-breaking — all existing `Run` implementations are unaffected.

### OTE change: update `runSpec` and the scheduler loop

**File:** `vendor/.../extensiontests/spec.go`

Change `runSpec` to return a slice:

```go
func runSpec(ctx context.Context, spec *ExtensionTestSpec, runSingleSpec bool) []*ExtensionTestResult {
    if runSingleSpec || spec.RunParallel == nil {
        if spec.RunMulti != nil {
            return spec.RunMulti(ctx)
        }
        return []*ExtensionTestResult{spec.Run(ctx)}
    }
    return []*ExtensionTestResult{spec.RunParallel(ctx)}
}
```

Change the scheduler loop (`spec.go:247`) to iterate the slice and call `w.Write` per
result. The name-overwrite becomes conditional — use `res.Name` if already set, else fall
back to `spec.Name`:

```go
results := runSpec(ctx, spec, runSingleSpec)
for _, res := range results {
    if res.Name == "" {
        res.Name = spec.Name
    }
    w.Write(res)
    resultChan <- res
}
```

Each `w.Write` call emits one JSONL line → one JUnit `<testcase>`.

### Origin usage: set `RunMulti` on the grouped OTE spec

**File:** `vendor/.../openshift-tests-extension/pkg/ginkgo/util.go`

In `makeOrderedContainerOTESpec`, set `RunMulti` instead of `Run`:

```go
testCase := &ext.ExtensionTestSpec{
    Name:   containerKey,
    Labels: unionLabels,
    RunMulti: func(ctx context.Context) []*ext.ExtensionTestResult {
        enforceSerialExecutionForGinkgo.Lock()
        defer enforceSerialExecutionForGinkgo.Unlock()

        suiteConfig, reporterConfig, _ := configureGinkgo()
        ginkgo.GetSuite().RunOrderedContainerSpecs(containerSpecs, ginkgo.Labels{}, "",
            cwd, ginkgo.GetFailer(), ginkgo.GetWriter(), *suiteConfig, *reporterConfig)

        var results []*ext.ExtensionTestResult
        for _, r := range ginkgo.GetSuite().GetReport().SpecReports {
            results = append(results, specReportToResult(r)) // same logic as existing Run
        }
        return results
    },
    RunParallel: func(ctx context.Context) *ext.ExtensionTestResult {
        return SpawnProcessToRunTest(ctx, containerKey, timeout)
    },
}
```

`RunMulti` calls `RunOrderedContainerSpecs` (Step 1 above) to run all container specs in
one process — `BeforeAll`/`AfterAll` fire correctly — then reads `SpecReports` and returns
one `*ExtensionTestResult` per Ginkgo spec with its individual name set. The scheduler
loop emits one JSONL line per result → full per-spec JUnit granularity preserved.

### Updated file summary (with `RunMulti`)

| File | Change |
|------|--------|
| `vendor/.../ginkgo/v2/internal/suite_patch.go` | +~25 lines: `RunOrderedContainerSpecs` |
| `vendor/.../extensiontests/types.go` | +4 lines: `RunMulti` field |
| `vendor/.../extensiontests/spec.go` | ~15 lines: `runSpec` returns slice, loop iterates |
| `vendor/.../openshift-tests-extension/pkg/ginkgo/util.go` | ~80 lines: two-pass grouping, `makeOrderedContainerOTESpec` using `RunMulti` |
| test file | ~60 lines |

The OTE changes (`types.go` + `spec.go`) are the upstream ask — small, additive, non-breaking.
Once merged and vendor-bumped, origin uses `RunMulti` and gets both correct `BeforeAll`
semantics and full per-spec JUnit granularity.

## Implementation Plan

### Overview

Two vendor files (both already patched by origin — not pristine upstream) plus a test.
~165 lines of new/changed code total.

### Step 1 — Add `RunOrderedContainerSpecs` to `suite_patch.go`

**File:** `vendor/github.com/onsi/ginkgo/v2/internal/suite_patch.go`

Add a method alongside the existing `RunSpec`. `RunSpec` already calls the private
`suite.runSpecs(..., []Spec{spec.(Spec)})` with a single-element slice. The new method
does the same with a multi-element slice — giving Ginkgo the full picture of the container
so `BeforeAll`/`AfterAll` fire correctly:

```go
func (suite *Suite) RunOrderedContainerSpecs(specs []types.TestSpec, suiteLabels Labels, suiteDescription, suitePath string, failer *Failer, writer WriterInterface, suiteConfig types.SuiteConfig, reporterConfig types.ReporterConfig) (bool, bool) {
    if suite.phase != PhaseBuildTree {
        panic("cannot run before building the tree")
    }
    suite.phase = PhaseRun
    // ... same setup as RunSpec (reporter, writer, interceptor, deadline, etc.) ...
    internal := make([]Spec, len(specs))
    for i, s := range specs {
        internal[i] = s.(Spec)
    }
    success := suite.runSpecs(suiteDescription, suiteLabels, SemVerConstraints{}, ComponentSemVerConstraints{}, suitePath, false, internal)
    return success, false
}
```

~25 lines. `runSpecs` (private) is what correctly handles `BeforeAll`/`AfterAll` across the
full slice; `RunSpec` already uses it for single specs.

### Step 2 — Restructure `BuildExtensionTestSpecsFromOpenShiftGinkgoSuite`

**File:** `vendor/github.com/openshift-eng/openshift-tests-extension/pkg/ginkgo/util.go`

Change from a single-pass inline creation to a two-pass approach.

**Pass 1** — collect all Ginkgo specs during `WalkTests`:
```go
type ginkgoSpec struct{ name string; spec types.TestSpec }
var allGinkgoSpecs []ginkgoSpec
ginkgo.GetSuite().WalkTests(func(name string, spec types.TestSpec) {
    allGinkgoSpecs = append(allGinkgoSpecs, ginkgoSpec{name, spec})
})
```

**Pass 2** — separate, group, and build OTE specs:
```go
type containerGroup struct {
    key   string
    specs []types.TestSpec
}
groups := map[string]*containerGroup{}
var orderedKeys []string // preserve first-seen insertion order

for _, gs := range allGinkgoSpecs {
    if !gs.spec.IsInOrderedContainer {
        oteSpecs = append(oteSpecs, makeRegularOTESpec(gs, ...)) // existing logic unchanged
        continue
    }
    key := strings.Join(gs.spec.ContainerHierarchyTexts, " ")
    if _, exists := groups[key]; !exists {
        groups[key] = &containerGroup{key: key}
        orderedKeys = append(orderedKeys, key)
    }
    groups[key].specs = append(groups[key].specs, gs.spec)
}
for _, key := range orderedKeys {
    oteSpecs = append(oteSpecs, makeOrderedContainerOTESpec(key, groups[key].specs, ...))
}
```

~80 lines restructured + new helpers.

### Step 3 — Implement the grouped OTE spec

**Name:** the container hierarchy key (e.g. `"[sig-auth] OIDC configuration"`)  
**Labels:** union of all spec labels in the group  
**Lifecycle:** blocking if any spec is blocking, otherwise informing  
**`Run`:**
```go
Run: func(ctx context.Context) *ext.ExtensionTestResult {
    enforceSerialExecutionForGinkgo.Lock()
    defer enforceSerialExecutionForGinkgo.Unlock()

    suiteConfig, reporterConfig, _ := configureGinkgo()
    ginkgo.GetSuite().RunOrderedContainerSpecs(containerSpecs, ginkgo.Labels{}, "", cwd,
        ginkgo.GetFailer(), ginkgo.GetWriter(), *suiteConfig, *reporterConfig)

    result := &ext.ExtensionTestResult{Name: containerKey, Result: ext.ResultPassed}
    for _, r := range ginkgo.GetSuite().GetReport().SpecReports {
        result.Output += r.CapturedGinkgoWriterOutput
        if r.State == types.SpecStateFailed || r.State == types.SpecStatePanicked ||
           r.State == types.SpecStateTimedout || r.State == types.SpecStateInterrupted {
            result.Result = ext.ResultFailed
            // append failure details to result.Error
        }
    }
    return result
},
```

**`RunParallel`:** `SpawnProcessToRunTest(ctx, containerKey, timeout)` — spawns
`run-test -n <containerKey>` which resolves to the grouped OTE spec's `Run`.

### Step 4 — Add a test

New test that:
1. Defines an ordered container with `BeforeAll` + 2 `It` blocks
2. Runs it through `BuildExtensionTestSpecsFromOpenShiftGinkgoSuite`
3. Asserts: exactly 1 OTE spec produced (not 2), `BeforeAll` called exactly once, both
   `It` blocks ran in order

~60 lines.

### File summary

| File | Change |
|------|--------|
| `vendor/.../ginkgo/v2/internal/suite_patch.go` | +~25 lines: `RunOrderedContainerSpecs` |
| `vendor/.../openshift-tests-extension/pkg/ginkgo/util.go` | ~80 lines restructured + helpers |
| test file | ~60 lines |

Both vendor files are already patched by origin (not pristine upstream), so modifying them
is the established pattern. The trickiest part is result aggregation after
`RunOrderedContainerSpecs` and ensuring `configureGinkgo()` resets Ginkgo state correctly
before each container run (currently done before each `RunSpec` call — same requirement).
