# flash 2.6.2

### Concurrent Map Race in TypeInferrer

`flash gen` with multiple query files crashed with `fatal error: concurrent map read and map write` because `TypeInferrer.cache` was accessed from concurrent goroutines without synchronization. Fixed by adding a `sync.RWMutex` — read lock for lookups, write lock for population.

---
