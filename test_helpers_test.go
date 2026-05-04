package guardgo

import "time"

// nowForTest is a fixed timestamp used by property tests that need a
// deterministic anchor for time-dependent assertions.
var nowForTest = time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
