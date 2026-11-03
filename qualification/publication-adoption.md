# Publication adoption qualification attempts

The unchanged provider passed all 42 required declaration cases against clean
Platform `9e4e377b7086db1788a1c6a0a63c2909db04f6a9` on 2026-11-03 at
20:35:03 UTC. The [new execution record](publication-adoption-declarations.json)
records 944.363 seconds of fixture execution; the aggregate live test took
951.98 seconds. The [attempt record](publication-adoption.json) also preserves
two failed full runs and one narrower diagnostic. The earlier
[42-case qualification](publication-declarations.md) keeps its original `65f8284`
source attribution. Saga still selects that earlier fixture.

All three attempts used clean provider harness `8d8427a43b219b95bbc4a115637d25d5ffebaf85`,
Terraform 1.15.3 on Linux amd64, and the existing installed provider
`bc1860930924ee5c0a88004273eb85bedf54cfe2`, SHA-256
`ad5897b486805411860785fc4bdf2689ab1f15e1d4bc1bf409c8f170d51646ea`.
The schema remains
`db04fc4ce0e14534c2e44cef932b197e6d9aeb48006f21f7a1523c21def0511f`.
The new loader resolves sibling helpers from the selected Platform checkout;
the provider executable and consumer constraints/checksums are unchanged.

## Complete serial execution

After this session's ordinary provider check/build finished, the same frozen
`8d8427a` harness and `9e4e377` fixture ran the complete gate serially. All 42
required cases passed, including all 24 provider-process deaths and separate
retained native-service restarts, complete history/capacity, denied reads,
credential changes, revision fences and alias isolation. The independent
catalog/nightly/production consumer used the unchanged mirror with all five
committed locks checked readonly and no compiler in Terraform's PATH.

The new record retains exact provider, native node, HTTP gateway, operator and
test-driver hashes. All consumer-lock hashes match the earlier passing record.
No fixture/HTTP schema, native pin, provider bytes, request identity or deadline
changed between the second failed attempt and this pass. This is a complete local
qualification at the recorded tuple, not a diagnosis or correction of the earlier
timeouts and not a performance guarantee under competing workloads.

The selected helper-loader change also passed eight Python checks and the locked
shell's `just fmt`, `just check` and `just build`; mock acceptance completed in
1,368.614 seconds. Both provider PR #18 CI jobs passed at `3ac191b` before the
later evidence-only commits. Current-head CI remains independently observable.

## Failed full runs

1. **Platform `9cf2b705149a1794bc25fc5e93935c395b32ab00`, 818.54 seconds.**
   The independent consumer completed, followed by authentication, denied release,
   binding admission, expiry, capacity, concurrency and application lifecycle.
   Application create recovery passed before acceptance and after completion.
   Its after-acceptance phase recovered the original application, checked its
   identity and an empty plan, then timed out at the existing 45-second cleanup
   deadline. Native logs show canceled storage work, request timeouts,
   `WalRecoveryRequired`, automatic reopening of retained history, and
   authority/capacity refusals. Reopening occurred, but cleanup did not finish
   within its deadline. The cause is not established; no data-loss claim follows.
2. **Platform `9e4e377b7086db1788a1c6a0a63c2909db04f6a9`, 76.91 seconds.**
   The first fixture's `publication-operator provision` exceeded its existing
   60-second deadline before the independent consumer or Terraform cases ran.
   Native commands were succeeding slowly, including a 7.249-second association.
   Local I/O pressure was observed, but its causal role is unproven.

Both outer reports failed with every required case still blocked. Neither
created the detailed passing execution report. Completed subtests are diagnostic
observations, not a partial gate pass. Private failed installations, journals,
matching executables and logs remain retained; the record includes log hashes.
No deadline, request identity, persisted state or failure assertion was relaxed.

## Separate diagnostic and Platform proof

A fresh fixture at the first tuple ran the unchanged application create-recovery
group alone. All three provider-death phases and their separate retained-service
restarts passed in 38.973 seconds on 2026-11-03 at 20:13:03 UTC. Its test driver
and provider bytes were unchanged. This result neither identifies the earlier
failure's cause nor completes the full 42-case gate.

Platform [PR #51](https://github.com/spoke-sh/platform/pull/51) merged bounded
original-key acceptance recovery, with 14 focused tests and 17 locally passing
native HTTP groups. Its later CI failed a mailbox-based history assertion.
[PR #53](https://github.com/spoke-sh/platform/pull/53) replaces that racy observation
with the caller's signed command sequence and real-mutation positive controls.
All 17 native groups and Platform's normal checks passed locally at the revised
fixture. That API proof is separate from installed Terraform qualification.

## Reproduction and remaining gates

Use the [explicit declaration-gate configuration](../docs/platform-http.md#publication-declaration-gate)
with the exact clean harness, fixture and artifact identities above:

```sh
SPOKE_LIVE_CONFIG=/absolute/private/declarations.json \
SPOKE_LIVE_REPORT=/absolute/private/new-result.json \
  nix develop --command just test-acc live
```

Both runs selected private scratch storage and a 1,800-second aggregate deadline.
An installed result must complete all 42 cases, including all 24 process-death
schedules; a fixture merge or isolated retry cannot substitute for that proof.
The successful third run used the same aggregate deadline after this session's
normal checks finished. It preserves the failed results rather than replacing
their source attribution or claiming an established root cause.

Saga's existing consumer Actions run remains blocked by missing read-only
`PLATFORM_SOURCE_TOKEN` source access. The installed provider, bucket pin,
publication schema and Saga consumer fixture remain unchanged. TF18 stays In Progress,
TF19 stays Planned; hosted state, uploads/sealing, delivered bytes and provider
release remain separate gates.
