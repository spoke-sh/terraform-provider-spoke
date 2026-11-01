# Native integration boundary

The native v2 adapter uses the platform's [application contract](https://github.com/spoke-sh/platform/blob/73b43609a1b493bafaee836bdc349c9e55db68ab/contracts/management-native-v2.md)
and the shared [Transit Serving caller](https://github.com/spoke-sh/transit/tree/c2e50061fc8622300e3a517e6e3a82f5c6eb6db1/sdks/go/serving).
The Go module and Nix native caller pin that same source. Both require definition
`8e441078693f1373d69323bfeadd5cfe8c7024ec2015819bacfb2b04eaea3865`.
Platform must adopt the matching runtime; its earlier application-contract
baseline used an incompatible native definition. No old installation is reset
or converted by provider configuration.

| Boundary | Implemented profile | Remaining qualification |
| --- | --- | --- |
| Transport and identity | Shared Zig preparation, original-byte recovery, subject proofs and verified private-CA transport; a bounded Go process adapter | Installed real-authority failure matrix at the selected artifact/placement |
| Recovery | Explicitly initialized, private, version-3 local journal; fsync before dispatch; one persistent flock inode; complete retained identity validation | No distributed/NFS locking, rollback/loss recovery or automatic compaction |
| Application | Native v2 bucket metadata CRUD/import, generations, settlement policy, tombstones and complete scoped discovery | Actual object bytes and broader service graph remain unavailable |
| Terraform state | Only authoritative absence removes state; denied reads, uncertainty and expired receipts remain errors | Selected target runs, including rights changes and process death |
| Distribution | Linux amd64 provider and shared native executable closure; normal mirror and readonly checksum lock | Real service-graph evidence still gates public release |

The provider owns the application intent codec and Terraform adaptation in
`internal/platformclient/`. The independent v2 vectors live in
`testdata/native-v2/vectors.json`. The independent native-v2 contract validators remain in `internal/nativecontract/`; they confer no dispatch authority.
There is no Go Ed25519 implementation or native protocol signer in the adapter.
The additive shared SDK uses an explicit installed process port; it does not
pretend to extend the existing raw-client C ABI. Each invocation is bounded and
joined during normal completion/cancellation. A killed provider can leave its
already-started native attempt finishing within its native timeout; the durable
original command identity remains the recovery authority.

See [native configuration and initialization](../docs/native-v2.md). A missing,
corrupt, foreign or sequence-inconsistent journal fails before dispatch. An
acknowledged business operation and a native ingress receipt have distinct
meanings. Replaying an unacknowledged command uses the SDK's exact exported work,
never a new signature or sequence. The local profile caps retained entries at
256 and serialized journal bytes at 1 MiB; exhaustion preserves original work.

`just test-acc live` is an explicit target preflight, not a fixture fallback.
It validates the installed artifact hash and selected immutable inputs, then
reports the cases whose execution evidence is still required. It no longer
reports the implemented Serving/proof interfaces as missing. The platform-owned
`pnpm test:provider` lane supplies local bootstrap, verified TLS, real native
fault controls and the independent installed Terraform consumer. A metadata pass
cannot satisfy the release manifest's full service-graph cases.

The [earlier qualification](milestones-9-11.md) and [old preflight record](native-preflight.json)
preserve the v1 assessment at their recorded revisions. Their SDK, settlement and
discovery gaps are historical observations, not the current v2 capability matrix.
