# Governor management identity

The experimental governor uses the separate provisional `ci-governor-v1`
[Platform contract](../internal/platformclient/governor-contract/upstream-profile.md).
It manages policy declarations; it supplies no native governor or runner service.

Keep `management_key` stable in configuration. Platform retains each explicit
sequence and full intent before committing policy. Keys are bound to the current
principal and scope; retired keys remain tombstones. Recreating a governor needs
a new key. Identical credentials and copied keys cannot distinguish cloned state
from legitimate recovery, and import does not transfer ownership.

Planning observes capabilities, the latest retained request and the exact
governor. Matching management reads bracket the resource projection; a repeatedly
changing cut is an error. No plan, refresh, import or discovery read commits,
aborts or settles a request.

A saved plan freezes the scope, principal, key, sequence, mutation kind, target,
expected revision and complete policy. Update and destroy use versioned private
plan metadata, including null destroy plans. The locked Framework 1.19 drops
private plan data on create, so creation uses computed `owner_id`,
`operation_sequence` and `operation_fingerprint` alongside the configured policy.
Apply reconstructs the create tuple and checks its scope-bound fingerprint before
dispatch; it cannot choose a different sequence or principal. This metadata is
a cache of a checked decision, not the durable recovery authority. Configuration
and the server's original request recover work even if the provider died before
Terraform saved any resource or private state. No secret is stored in this data.

| Condition | Behavior |
| --- | --- |
| New key | First create selects sequence zero and revision zero. |
| Matching prepared/accepted work | Apply resumes the exact retained intent. Its original sequence and revision never change. |
| Different configuration while work is pending | Return a conflict. Restore the original configuration to recover; do not auto-abort or advance to another sequence. |
| Completed request with a lost response | Recover its existing resource or authoritative retirement absence without another mutation. |
| Fresh change after a terminal request | Select one successor, using the revision observed by that plan. A → B → A uses distinct requests. |
| Refused/aborted request | Return the original outcome for the saved plan. Refresh and create a new plan before selecting a successor. |
| Stale plan or changed owner/target | Fail without rebasing the request. |
| Expired receipt or exhausted capacity | Fail without recycling keys or clearing uncertain state. |

An existing live governor with lost Terraform state can be recovered by its key
only while its original creation is the latest request and configuration matches
the owned policy. Once management has advanced, import its canonical
`organization/project/environment/governor_id`, inspect the plan and then update.
Admission entitlement changes require retirement and a new management key.

Retirement requires deletion protection to have been disabled by a completed
policy update. Outstanding leases, allocated capacity and unsettled usage keep
the original retirement pending. Timeout preserves state and request identity;
it does not cancel accepted work. Cache, artifact and usage references retain
their separate owners. An applied policy can remain `disabled` with
`execution_unqualified`; management success does not qualify physical execution.

See the [implementation plan](roadmap/ci-governor-implementation-plan.md#next-slice-terraform-governor-adapter-and-installed-recovery)
for the independent client, Terraform, native-service and release evidence gates.
