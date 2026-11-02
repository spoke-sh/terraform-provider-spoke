# Platform bucket profile

`schema.json` and `vectors.json` are byte-for-byte copies of
`contracts/http/bucket-v1.schema.json` and `contracts/http/bucket-v1.vectors.json`
from Platform commit `ebf97fe086a99ba91f6b669b9f922f3964eb5bfa`.
The schema narrows the MIT-licensed provider inventory; its upstream license is
preserved in `LICENSE.provider`. The owning HTTP contract and management profile
are in that Platform commit's `contracts/http/` directory.

`internal/platformclient/profile.json` pins both digests and the source revision.
This development metadata profile is separate from the complete provisional
inventory in `mock-v5.schema.json`. It establishes no object storage or release
qualification. Update the pin and copied sources together after contract review.
